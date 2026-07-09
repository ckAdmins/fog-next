// Package netup handles NIC enumeration and network readiness detection.
// It does NOT perform DHCP or link bringup — those are handled by OpenRC
// networking services.  The agent only waits for the network to become
// available before proceeding with the handshake.
package netup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ckAdmins/fog-next/agent/internal/tui"
)

const (
	sysNetPath   = "/sys/class/net"
	pollInterval = 500 * time.Millisecond
)

// NIC holds information about a single network interface.
type NIC struct {
	Name string
	MAC  string
}

// ListNICs returns all non-loopback NICs present on the system, read from sysfs.
func ListNICs() ([]NIC, error) {
	entries, err := os.ReadDir(sysNetPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sysNetPath, err)
	}
	var nics []NIC
	for _, e := range entries {
		name := e.Name()
		if name == "lo" {
			continue
		}
		mac, err := os.ReadFile(filepath.Join(sysNetPath, name, "address"))
		if err != nil {
			continue
		}
		nics = append(nics, NIC{Name: name, MAC: strings.TrimSpace(string(mac))})
	}
	return nics, nil
}

// WaitForNetwork blocks until at least one NIC has a non-loopback,
// non-link-local IPv4 address, or ctx is cancelled.  Unlike the old
// BringUp, it does NOT run ip-link or udhcpc — those are handled by
// the init system (OpenRC).
// Returns the primary MAC (from the first interface that gets an address).
// If p is non-nil, NIC state changes are reflected in the TUI.
func WaitForNetwork(ctx context.Context, p *tea.Program) (primaryMAC string, err error) {
	nics, err := ListNICs()
	if err != nil {
		return "", err
	}
	if len(nics) == 0 {
		return "", errors.New("no non-loopback NICs found")
	}

	if p != nil {
		ifaces := make([]tui.NetIface, len(nics))
		for i, n := range nics {
			ifaces[i] = tui.NetIface{Name: n.Name, MAC: n.MAC, Up: true}
		}
		tui.SendMsg(p, tui.InterfacesMsg(ifaces))
		tui.SendMsg(p, tui.PhaseMsg("Waiting for network (DHCP)..."))
	}

	slog.Info("waiting for network (DHCP handled by OpenRC)")
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
			mac, ok := firstAddressedMAC(nics)
			if ok {
				slog.Info("network up", "mac", mac)
				if p != nil {
					tui.SendMsg(p, tui.PrimaryIPMsg(mac))
					tui.SendMsg(p, tui.PhaseMsg("Network up — starting handshake..."))
					updateInterfaceIPs(p, nics)
				}
				return mac, nil
			}
		}
	}
}

func firstAddressedMAC(nics []NIC) (string, bool) {
	for _, nic := range nics {
		iface, err := net.InterfaceByName(nic.Name)
		if err != nil {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				return nic.MAC, true
			}
		}
	}
	return "", false
}

func updateInterfaceIPs(p *tea.Program, nics []NIC) {
	for _, nic := range nics {
		iface, err := net.InterfaceByName(nic.Name)
		if err != nil {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ips []string
		for _, addr := range addrs {
			ips = append(ips, addr.String())
		}
		tui.SendMsg(p, tui.InterfaceIPsMsg{Name: nic.Name, IPs: ips})
	}
}
