package handlers

import (
"net/http"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/api/response"
"github.com/nemvince/fog-next/internal/config"
"github.com/nemvince/fog-next/internal/pxe"
)

// BootHandler serves iPXE boot scripts based on the MAC address of the
// booting client.
type BootHandler struct {
cfg *config.Config
db  *ent.Client
}

func NewBoot(cfg *config.Config, db *ent.Client) *BootHandler {
return &BootHandler{cfg: cfg, db: db}
}

// ServeHTTP handles GET /fog/boot
func (h *BootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
mac := r.URL.Query().Get("mac")
if mac == "" {
mac = r.URL.Query().Get("client_mac")
}
if mac == "" {
w.Header().Set("Content-Type", "text/plain")
_, _ = w.Write(chainLoadScript(h.cfg.Server.BaseURL))
return
}

params, err := pxe.BootParamsForMAC(r.Context(), h.db, mac, h.cfg.Server.BaseURL)
if err != nil {
response.InternalError(w)
return
}

script, err := pxe.GenerateScript(params)
if err != nil {
response.InternalError(w)
return
}

w.Header().Set("Content-Type", "text/plain; charset=utf-8")
w.WriteHeader(http.StatusOK)
_, _ = w.Write(script)
}

// chainLoadScript returns an iPXE script that chain-loads back to the boot
// endpoint with the MAC address embedded.
func chainLoadScript(baseURL string) []byte {
return []byte("#!ipxe\nchain " + baseURL + "/fog/boot?mac=${net0/mac}\n")
}
