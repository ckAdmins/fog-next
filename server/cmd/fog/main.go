// Package main is the CLI entry point for the fog-next server.
// Usage:
//
//	fog serve              -- start the HTTP server + all background services
//	fog migrate up         -- apply pending schema migrations
//	fog install            -- interactive first-run setup
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	entuser "github.com/ckAdmins/fog-next/ent/user"
	"github.com/ckAdmins/fog-next/internal/api"
	"github.com/ckAdmins/fog-next/internal/auth"
	"github.com/ckAdmins/fog-next/internal/config"
	"github.com/ckAdmins/fog-next/internal/database"
	"github.com/ckAdmins/fog-next/internal/fos"
	"github.com/ckAdmins/fog-next/internal/services"
	"github.com/ckAdmins/fog-next/internal/tftp"
	"golang.org/x/term"
)

var cfgFile string

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// ------------------------------------------------------------------ root ---

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fog",
		Short: "FOG Next — network boot and imaging server",
	}
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: /etc/fog/config.yaml)")
	root.AddCommand(serveCmd(), migrateCmd(), installCmd(), fetchKernelsCmd(), versionCmd())
	return root
}

// ----------------------------------------------------------------- serve ---

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the FOG server",
		RunE:  runServe,
	}
}

func runServe(_ *cobra.Command, _ []string) error {
	cfg := mustConfig()

	setupLogger(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Database
	client, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("database connect: %w", err)
	}
	defer client.Close()

	// Auto-migrate schema on startup
	if err := database.Migrate(ctx, client); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	// TFTP server
	tftpSrv := tftp.New(cfg)
	go func() {
		if err := tftpSrv.ListenAndServe(); err != nil {
			slog.Error("tftp server error", "error", err)
		}
	}()

	// Background services
	mgr := services.New(
		services.NewTaskScheduler(cfg, client),
		services.NewImageReplicator(cfg, client),
		services.NewSnapinReplicator(cfg, client),
		services.NewMulticastManager(cfg, client),
		services.NewPingHosts(cfg, client),
		services.NewImageSize(cfg, client),
		services.NewSnapinHash(cfg, client),
	)
	go mgr.Run(ctx)

	// HTTP API server
	srv := api.New(cfg, client)
	errCh := make(chan error, 1)
	go func() {
		slog.Info("fog server starting",
			"http", cfg.Server.HTTP,
			"https_enabled", cfg.Server.TLSCert != "")
		errCh <- srv.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
		return nil
	case err := <-errCh:
		return err
	}
}

// --------------------------------------------------------------- migrate ---

func migrateCmd() *cobra.Command {
	mc := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database schema migrations",
	}
	mc.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Apply / sync schema (Ent auto-migrate)",
			RunE:  runMigrateUp,
		},
	)
	return mc
}

func runMigrateUp(_ *cobra.Command, _ []string) error {
	cfg := mustConfig()
	ctx := context.Background()
	client, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := database.Migrate(ctx, client); err != nil {
		return err
	}
	fmt.Println("schema migration applied")
	return nil
}

// --------------------------------------------------------------- install ---

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Create an admin user if one does not already exist",
		RunE:  runInstall,
	}
}

func runInstall(_ *cobra.Command, _ []string) error {
	cfg := mustConfig()
	ctx := context.Background()

	client, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("database connect: %w", err)
	}
	defer client.Close()

	if err := database.Migrate(ctx, client); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	existing, err := client.User.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, u := range existing {
		if u.Role == entuser.RoleAdmin {
			fmt.Printf("Admin user %q already exists — nothing to do.\n", u.Username)
			return nil
		}
	}

	fmt.Print("Admin username [fog]: ")
	var adminUser string
	_, _ = fmt.Scanln(&adminUser)
	if adminUser == "" {
		adminUser = "fog"
	}

	adminPass := promptPassword("Admin password")
	if adminPass == "" {
		return fmt.Errorf("admin password must not be empty")
	}

	hash, err := auth.HashPassword(adminPass)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := client.User.Create().
		SetUsername(adminUser).
		SetPasswordHash(hash).
		SetRole(entuser.RoleAdmin).
		SetIsActive(true).
		Save(ctx); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	fmt.Printf("Admin user %q created.\n", adminUser)

	// Download fog-next kernel and initramfs unless explicitly disabled.
	if !cfg.FOS.SkipDownload {
		fmt.Printf("\nDownloading fog-next agent artifacts from %s\n", cfg.FOS.ReleaseURL)
		d := fos.New(cfg.FOS, cfg.Storage.KernelPath)
		if err := d.Download(context.Background()); err != nil {
			fmt.Printf("Warning: agent download failed: %v\n", err)
			fmt.Println("You can retry later with: fog fetch-kernels")
		} else {
			fmt.Printf("agent artifacts installed to %s\n", cfg.Storage.KernelPath)
		}
	} else {
		fmt.Println("Skipping agent download (fos.skip_download = true).")
	}
	return nil
}

// ------------------------------------------------------- fetch-kernels ----

func fetchKernelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch-kernels",
		Short: "Download or re-download the fog-next kernel and initramfs",
		Long: `Downloads the fog-next agent kernel (bzImage) and initramfs (init.xz) from the
configured release URL and installs them into the kernel_path directory.

The release URL is read from fos.release_url in the config file.
Set fos.skip_download: true to permanently disable automatic downloading.

Example:
  fog fetch-kernels
  fog fetch-kernels -c /etc/fog/config.yaml`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := mustConfig()
			if cfg.FOS.SkipDownload {
				fmt.Println("fos.skip_download is true — nothing to do.")
				return nil
			}
			fmt.Printf("Downloading fog-next agent artifacts from %s\n", cfg.FOS.ReleaseURL)
			d := fos.New(cfg.FOS, cfg.Storage.KernelPath)
			if err := d.Download(context.Background()); err != nil {
				return err
			}
			fmt.Printf("agent artifacts installed to %s\n", cfg.Storage.KernelPath)
			return nil
		},
	}
}

// -------------------------------------------------------------- version ---

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the fog version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("fog-next version 0.1.0-dev")
		},
	}
}

// ---------------------------------------------------------------- helpers ---

func mustConfig() *config.Config {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}
	return cfg
}

func setupLogger(cfg *config.Config) {
	level := slog.LevelInfo
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

func promptPassword(label string) string {
	fmt.Printf("  %s: ", label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		var v string
		_, _ = fmt.Scanln(&v)
		return v
	}
	return string(b)
}

