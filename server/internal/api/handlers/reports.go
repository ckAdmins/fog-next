package handlers

import (
"net/http"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/api/response"
)

// Reports handles read-only reporting endpoints.
type Reports struct{ db *ent.Client }

func NewReports(db *ent.Client) *Reports { return &Reports{db} }

type hostWithInventory struct {
*ent.Host
Inventory interface{} `json:"inventory"`
}

// ImagingHistory returns a paginated list of imaging log entries.
func (h *Reports) ImagingHistory(w http.ResponseWriter, r *http.Request) {
limit := parseLimitParam(r, 50)
logs, err := h.db.ImagingLog.Query().Limit(limit).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(logs))
}

// HostInventory returns a paginated list of hosts with their inventory data.
func (h *Reports) HostInventory(w http.ResponseWriter, r *http.Request) {
limit := parseLimitParam(r, 50)
hosts, err := h.db.Host.Query().Limit(limit).All(r.Context())
if err != nil {
response.InternalError(w)
return
}

result := make([]hostWithInventory, 0, len(hosts))
for _, host := range hosts {
inv, _ := host.QueryInventory().Only(r.Context())
result = append(result, hostWithInventory{Host: host, Inventory: inv})
}
response.OK(w, response.ListOf(result))
}

// parseLimitParam reads the ?limit= query param, capping at 200.
func parseLimitParam(r *http.Request, defaultVal int) int {
v := r.URL.Query().Get("limit")
if v == "" {
return defaultVal
}
n := 0
for _, c := range v {
if c < '0' || c > '9' {
return defaultVal
}
n = n*10 + int(c-'0')
}
if n <= 0 || n > 200 {
return defaultVal
}
return n
}
