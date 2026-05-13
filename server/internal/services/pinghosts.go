package services

import (
"context"
"log/slog"
"net"
"sync"
"time"

"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/config"
)

// PingHosts periodically pings all registered hosts and updates their last_contact timestamp.
type PingHosts struct {
cfg *config.Config
db  *ent.Client
}

func NewPingHosts(cfg *config.Config, db *ent.Client) *PingHosts {
return &PingHosts{cfg, db}
}

func (p *PingHosts) Name() string { return "PingHosts" }

func (p *PingHosts) Run(ctx context.Context) error {
ticker := time.NewTicker(p.cfg.Services.PingInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil
case <-ticker.C:
p.pingAll(ctx)
}
}
}

func (p *PingHosts) pingAll(ctx context.Context) {
hosts, err := p.db.Host.Query().Limit(10000).All(ctx)
if err != nil {
slog.Error("pinghosts: list hosts", "error", err)
return
}

var wg sync.WaitGroup
sem := make(chan struct{}, 50)

for _, h := range hosts {
if h.IP == "" {
continue
}

wg.Add(1)
go func(id uuid.UUID, ip, hostname string) {
defer wg.Done()
sem <- struct{}{}
defer func() { <-sem }()

if tcpPing(ctx, ip) {
slog.Debug("pinghosts: host reachable", "host", hostname, "ip", ip)
_ = p.db.Host.UpdateOneID(id).SetLastContact(time.Now()).Exec(ctx)
}
}(h.ID, h.IP, h.Name)
}

wg.Wait()
}

func tcpPing(ctx context.Context, ip string) bool {
dialer := net.Dialer{Timeout: 2 * time.Second}
conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, "22"))
if err != nil {
return false
}
conn.Close()
return true
}
