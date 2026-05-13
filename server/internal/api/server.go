// Package api wires the HTTP server, middleware stack, and route handlers.
package api

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/nemvince/fog-next/ent"
	"github.com/nemvince/fog-next/internal/api/handlers"
	"github.com/nemvince/fog-next/internal/api/middleware"
	"github.com/nemvince/fog-next/internal/config"
	"github.com/nemvince/fog-next/internal/plugins"
	"github.com/nemvince/fog-next/internal/ws"
)

//go:embed all:static
var embeddedStatic embed.FS

// Server is the HTTP server for the FOG API and web UI.
type Server struct {
	cfg     *config.Config
	db      *ent.Client
	hub     *ws.Hub
	plugins *plugins.Registry
	router  *chi.Mux
	http    *http.Server
	https   *http.Server
}

// New creates a configured Server ready to serve. It uses the
// plugins.DefaultRegistry for hook dispatch unless WithPlugins is called.
func New(cfg *config.Config, db *ent.Client) *Server {
	s := &Server{cfg: cfg, db: db, hub: ws.New(), plugins: plugins.DefaultRegistry}
	s.router = s.buildRouter()
	s.http = &http.Server{
		Addr:              cfg.Server.HTTP,
		Handler:           s.router,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		// No ReadTimeout or WriteTimeout — image streams can run for many minutes.
	}
	return s
}

// Hub returns the WebSocket broadcast hub so background services can emit events.
func (s *Server) Hub() *ws.Hub { return s.hub }

// WithPlugins replaces the plugin registry used by the server.
// Must be called before Start.
func (s *Server) WithPlugins(reg *plugins.Registry) *Server { s.plugins = reg; return s }

// Start runs the HTTP (and optionally HTTPS) server until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	if s.cfg.Server.TLSCert != "" && s.cfg.Server.TLSKey != "" {
		s.https = &http.Server{
			Addr:              s.cfg.Server.HTTPS,
			Handler:           s.router,
			ReadHeaderTimeout: s.cfg.Server.ReadHeaderTimeout,
		}
		go func() {
			if err := s.https.ListenAndServeTLS(s.cfg.Server.TLSCert, s.cfg.Server.TLSKey); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutCtx)
		if s.https != nil {
			_ = s.https.Shutdown(shutCtx)
		}
		return nil
	}
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── REST API v1 ─────────────────────────────────────────────────
	r.Route("/fog/api/v1", func(r chi.Router) {
                // Public (no auth required) — rate-limited to 10 req/s burst 20 per IP
                authH := handlers.NewAuth(s.cfg, s.db)
                authRL := middleware.NewRateLimiter(10, 20)
                r.With(authRL.Handler).Post("/auth/login", authH.Login)
                r.With(authRL.Handler).Post("/auth/refresh", authH.Refresh)

		// ── Boot API (fos-agent endpoints) ─────────────────────
		// Unauthenticated endpoints (handshake and register are public).
		bootRL := middleware.NewRateLimiter(5, 10)
		bootAPIH := handlers.NewBootAPI(s.cfg, s.db, s.hub)
		r.With(bootRL.Handler).Post("/boot/handshake", bootAPIH.Handshake)
		r.With(bootRL.Handler).Post("/boot/register", bootAPIH.Register)
		// Boot-token-authenticated endpoints.
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthenticateBoot(s.cfg))
			r.Post("/boot/progress", bootAPIH.Progress)
			r.Post("/boot/complete", bootAPIH.Complete)
			r.Post("/boot/logs", bootAPIH.Logs)
			r.Post("/boot/images/meta", bootAPIH.ImageMeta)
			r.Route("/boot/images/{id}", func(r chi.Router) {
				r.Get("/download", bootAPIH.Download)
				r.Put("/upload", bootAPIH.Upload)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(s.cfg))

			r.Post("/auth/logout", authH.Logout)

			hostH := handlers.NewHosts(s.db, s.plugins)
			r.Route("/hosts", func(r chi.Router) {
				r.Get("/", hostH.List)
				r.Post("/", hostH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", hostH.Get)
					r.Put("/", hostH.Update)
					r.Delete("/", hostH.Delete)
					r.Get("/macs", hostH.ListMACs)
					r.Post("/macs", hostH.AddMAC)
					r.Delete("/macs/{macId}", hostH.DeleteMAC)
					r.Get("/inventory", hostH.GetInventory)
					r.Get("/task", hostH.GetActiveTask)
				})
			})
			r.Get("/pending-macs", hostH.ListPendingMACs)

			imgH := handlers.NewImages(s.db, &s.cfg.Storage)
			r.Route("/images", func(r chi.Router) {
				r.Get("/", imgH.List)
				r.Post("/", imgH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", imgH.Get)
					r.Put("/", imgH.Update)
					r.Delete("/", imgH.Delete)
					r.Get("/download", imgH.Download)
					r.Put("/upload", imgH.Upload)
				})
			})

			grpH := handlers.NewGroups(s.db)
			r.Route("/groups", func(r chi.Router) {
				r.Get("/", grpH.List)
				r.Post("/", grpH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", grpH.Get)
					r.Put("/", grpH.Update)
					r.Delete("/", grpH.Delete)
					r.Get("/members", grpH.ListMembers)
					r.Post("/members", grpH.AddMember)
					r.Delete("/members/{hostId}", grpH.RemoveMember)
				})
			})

			taskH := handlers.NewTasks(s.db, s.plugins)
			r.Route("/tasks", func(r chi.Router) {
				r.Get("/", taskH.List)
				r.Post("/", taskH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", taskH.Get)
					r.Delete("/", taskH.Cancel)
					r.Post("/progress", taskH.UpdateProgress)
					r.Get("/logs", handlers.NewLogs(s.db).GetTaskLogs)
				})
			})

			snapH := handlers.NewSnapins(s.cfg, s.db)
			r.Route("/snapins", func(r chi.Router) {
				r.Get("/", snapH.List)
				r.Post("/", snapH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", snapH.Get)
					r.Put("/", snapH.Update)
					r.Delete("/", snapH.Delete)
					r.Post("/upload", snapH.Upload)
				})
			})

			storH := handlers.NewStorage(s.db)
			r.Route("/storage/groups", func(r chi.Router) {
				r.Get("/", storH.ListGroups)
				r.Post("/", storH.CreateGroup)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", storH.GetGroup)
					r.Put("/", storH.UpdateGroup)
					r.Delete("/", storH.DeleteGroup)
					r.Get("/nodes", storH.ListNodes)
					r.Post("/nodes", storH.CreateNode)
				})
			})
			r.Route("/storage/nodes/{id}", func(r chi.Router) {
				r.Get("/", storH.GetNode)
				r.Put("/", storH.UpdateNode)
				r.Delete("/", storH.DeleteNode)
			})

			userH := handlers.NewUsers(s.cfg, s.db)
			r.Route("/users", func(r chi.Router) {
				r.Get("/", userH.List)
				r.Post("/", userH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", userH.Get)
					r.Put("/", userH.Update)
					r.Delete("/", userH.Delete)
					r.Post("/regenerate-token", userH.RegenerateToken)
				})
			})

			settH := handlers.NewSettings(s.db)
			r.Route("/settings", func(r chi.Router) {
				r.Get("/", settH.List)
				r.Put("/{key}", settH.Set)
				r.Delete("/{key}", settH.Delete)
			})

			rptH := handlers.NewReports(s.db)
			r.Route("/reports", func(r chi.Router) {
				r.Get("/imaging", rptH.ImagingHistory)
				r.Get("/inventory", rptH.HostInventory)
			})

			// WebSocket live events
			r.Handle("/ws", ws.NewHandler(s.hub))
		})
	})

	// ── Legacy client endpoints (FOG 1.x client compatibility) ──────
	r.Route("/fog/service", func(r chi.Router) {
		legacyH := handlers.NewLegacy(s.cfg, s.db)
		r.Post("/register.php", legacyH.Register)
		r.Get("/hostinfo.php", legacyH.HostInfo)
		r.Post("/progress.php", legacyH.Progress)
		r.Get("/jobs.php", legacyH.Jobs)
	})
	r.Get("/fog/service/ipxe/boot.php", func(w http.ResponseWriter, req *http.Request) {
		handlers.NewBoot(s.cfg, s.db).ServeHTTP(w, req)
	})
	// Primary iPXE boot endpoint (used by chain-load from DHCP option 67).
	r.Get("/fog/boot", func(w http.ResponseWriter, req *http.Request) {
		handlers.NewBoot(s.cfg, s.db).ServeHTTP(w, req)
	})

	// Kernel/initrd file server — iPXE scripts reference these URLs to load
	// the FOG kernel and init image over HTTP.
	kernelPath := s.cfg.Storage.KernelPath
	if kernelPath == "" {
		kernelPath = "/opt/fog/kernels"
	}
	r.Handle("/fog/kernel/*", http.StripPrefix("/fog/kernel", http.FileServer(http.Dir(kernelPath))))

	// ── Static SPA (served last so API routes take priority) ────────
	static, _ := fs.Sub(embeddedStatic, "static")
	r.Handle("/*", spaHandler(http.FS(static)))

	return r
}

// spaHandler returns a handler that serves the SPA index.html for any
// path that doesn't map to an actual file (React Router history mode).
func spaHandler(fsys http.FileSystem) http.HandlerFunc {
	fileServer := http.FileServer(fsys)
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := fsys.Open(r.URL.Path)
		if err != nil {
			// File not found → serve index.html and let the frontend router handle it.
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	}
}
