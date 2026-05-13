package handlers

import (
"context"
"net/http"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/api/middleware"
)

// writeAudit records an audit log entry in a fire-and-forget goroutine.
func writeAudit(ctx context.Context, db *ent.Client, r *http.Request, action, resource, resourceID, details string) {
claims := middleware.ClaimsFrom(ctx)
go func() {
c := db.AuditLog.Create().
SetAction(action).
SetResource(resource).
SetResourceID(resourceID).
SetDetails(details).
SetIPAddress(r.RemoteAddr)
if claims != nil {
c = c.SetUserID(claims.UserID)
}
_ = c.Exec(context.Background())
}()
}
