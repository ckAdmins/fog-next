// Package pxe generates iPXE boot scripts for PXE-booting clients.
// The generated scripts are returned to the iPXE firmware over HTTP and
// drive the boot menu, imaging tasks, and host registration workflow.
package pxe

import (
"bytes"
"context"
"fmt"
"strings"
"text/template"

"github.com/nemvince/fog-next/ent"
enttask "github.com/nemvince/fog-next/ent/task"
"github.com/nemvince/fog-next/ent/hostmac"
)

// BootParams contains everything the boot script generator needs.
type BootParams struct {
Host       *ent.Host  // nil if unregistered
Task       *ent.Task  // nil if no pending task
ServerURL  string     // e.g. "http://10.0.0.1"
KernelURL  string     // base URL for kernel/initrd
KernelArgs string     // pre-computed kernel command-line args
}

// GenerateScript returns the iPXE script for the given parameters.
func GenerateScript(p BootParams) ([]byte, error) {
var tmplStr string

switch {
case p.Host == nil:
p.KernelArgs = buildKernelArgs(p, "fog_action=register")
tmplStr = registerScript
case p.Task != nil && p.Task.Type == enttask.TypeDeploy:
p.KernelArgs = buildKernelArgs(p, "fog_action=deploy")
tmplStr = deployScript
case p.Task != nil && p.Task.Type == enttask.TypeCapture:
p.KernelArgs = buildKernelArgs(p, "fog_action=capture")
tmplStr = captureScript
case p.Task != nil && p.Task.Type == enttask.TypeMulticast:
p.KernelArgs = buildKernelArgs(p, "fog_action=multicast")
tmplStr = multicastScript
case p.Task != nil && p.Task.Type == enttask.TypeDebugDeploy,
p.Task != nil && p.Task.Type == enttask.TypeDebugCapture:
p.KernelArgs = buildKernelArgs(p, "fog_action=debug")
tmplStr = debugScript
case p.Task != nil && p.Task.Type == enttask.TypeMemtest:
p.KernelArgs = buildKernelArgs(p, "")
tmplStr = memtestScript
default:
tmplStr = localBootScript
}

t, err := template.New("ipxe").Parse(tmplStr)
if err != nil {
return nil, fmt.Errorf("pxe template parse: %w", err)
}

var buf bytes.Buffer
if err := t.Execute(&buf, p); err != nil {
return nil, fmt.Errorf("pxe template execute: %w", err)
}
return buf.Bytes(), nil
}

// BootParamsForMAC resolves the BootParams for a given MAC address.
func BootParamsForMAC(ctx context.Context, db *ent.Client, mac, serverURL string) (BootParams, error) {
p := BootParams{ServerURL: serverURL, KernelURL: serverURL + "/fog/kernel"}

host, err := db.HostMAC.Query().Where(hostmac.MACEQ(mac)).QueryHost().Only(ctx)
if err != nil {
// Not found — unregistered host.
return p, nil
}
p.Host = host

task, err := db.Task.Query().Where(
enttask.HostIDEQ(host.ID),
enttask.StateIn(enttask.StateQueued, enttask.StateActive),
).Order(enttask.ByCreatedAt()).First(ctx)
if err == nil {
p.Task = task
}
return p, nil
}

// buildKernelArgs assembles the kernel command-line string.
func buildKernelArgs(p BootParams, extraArgs string) string {
parts := []string{
"console=tty0",
"console=ttyS0,115200",
"earlyprintk=efi,keep",
fmt.Sprintf("fog_server=%s", p.ServerURL),
}
if p.Host != nil {
parts = append(parts, fmt.Sprintf("fog_host=%s", p.Host.Name))
if p.Host.KernelArgs != "" {
parts = append(parts, p.Host.KernelArgs)
}
}
if p.Task != nil {
parts = append(parts, fmt.Sprintf("fog_task=%s", p.Task.ID))
}
if extraArgs != "" {
parts = append(parts, extraArgs)
}
return strings.Join(parts, " ")
}

// ---------------------------------------------------------------- templates

const registerScript = `#!ipxe
# FOG Next - Host Registration
echo FOG Network Boot - Registering host

kernel {{ .KernelURL }}/bzImage {{ .KernelArgs }}
initrd {{ .KernelURL }}/init.xz
boot || prompt --key s --timeout 30 Press s to drop to shell && shell
`

const deployScript = `#!ipxe
# FOG Next - Deploy Image
echo FOG Network Boot - Deploying image to {{ .Host.Name }}

kernel {{ .KernelURL }}/bzImage {{ .KernelArgs }}
initrd {{ .KernelURL }}/init.xz
boot || prompt --key s --timeout 30 Press s to drop to shell && shell
`

const captureScript = `#!ipxe
# FOG Next - Capture Image
echo FOG Network Boot - Capturing image from {{ .Host.Name }}

kernel {{ .KernelURL }}/bzImage {{ .KernelArgs }}
initrd {{ .KernelURL }}/init.xz
boot || prompt --key s --timeout 30 Press s to drop to shell && shell
`

const multicastScript = `#!ipxe
# FOG Next - Multicast Deploy
echo FOG Network Boot - Multicast imaging {{ .Host.Name }}

kernel {{ .KernelURL }}/bzImage {{ .KernelArgs }}
initrd {{ .KernelURL }}/init.xz
boot || prompt --key s --timeout 30 Press s to drop to shell && shell
`

const debugScript = `#!ipxe
# FOG Next - Debug Shell
echo FOG Network Boot - Debug mode for {{ .Host.Name }}

kernel {{ .KernelURL }}/bzImage {{ .KernelArgs }}
initrd {{ .KernelURL }}/init.xz
boot || shell
`

const memtestScript = `#!ipxe
# FOG Next - Memory Test
echo FOG Network Boot - Running memtest for {{ .Host.Name }}

kernel {{ .KernelURL }}/memdisk raw
initrd {{ .KernelURL }}/memtest.bin
boot || prompt --key s --timeout 30 Press s to drop to shell && shell
`

const localBootScript = `#!ipxe
# FOG Next - Local Boot
echo FOG Network Boot - Booting {{ .Host.Name }} from local disk
sanboot --no-describe --drive 0x80 || exit
`
