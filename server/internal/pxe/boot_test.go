package pxe_test

import (
"strings"
"testing"

"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
enttask "github.com/nemvince/fog-next/ent/task"
"github.com/nemvince/fog-next/internal/pxe"
)

func baseHost() *ent.Host {
return &ent.Host{
ID:   uuid.New(),
Name: "test-host",
}
}

func taskOf(typ enttask.Type) *ent.Task {
return &ent.Task{
ID:   uuid.New(),
Type: typ,
}
}

func mustGenerate(t *testing.T, p pxe.BootParams) string {
t.Helper()
b, err := pxe.GenerateScript(p)
if err != nil {
t.Fatalf("GenerateScript: %v", err)
}
return string(b)
}

func TestGenerateScript_Register(t *testing.T) {
p := pxe.BootParams{
Host:      nil,
Task:      nil,
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.HasPrefix(script, "#!ipxe") {
t.Error("script does not start with #!ipxe")
}
if !strings.Contains(script, "fog_action=register") {
t.Error("register script should contain fog_action=register")
}
if !strings.Contains(script, "fog_server=http://10.0.0.1") {
t.Error("script should contain fog_server")
}
}

func TestGenerateScript_Deploy(t *testing.T) {
p := pxe.BootParams{
Host:      baseHost(),
Task:      taskOf(enttask.TypeDeploy),
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.Contains(script, "fog_action=deploy") {
t.Error("deploy script should contain fog_action=deploy")
}
if !strings.Contains(script, "fog_host=test-host") {
t.Error("deploy script should contain fog_host")
}
}

func TestGenerateScript_Capture(t *testing.T) {
p := pxe.BootParams{
Host:      baseHost(),
Task:      taskOf(enttask.TypeCapture),
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.Contains(script, "fog_action=capture") {
t.Error("capture script should contain fog_action=capture")
}
}

func TestGenerateScript_Multicast(t *testing.T) {
p := pxe.BootParams{
Host:      baseHost(),
Task:      taskOf(enttask.TypeMulticast),
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.Contains(script, "fog_action=multicast") {
t.Error("multicast script should contain fog_action=multicast")
}
}

func TestGenerateScript_Debug(t *testing.T) {
for _, typ := range []enttask.Type{enttask.TypeDebugDeploy, enttask.TypeDebugCapture} {
p := pxe.BootParams{
Host:      baseHost(),
Task:      taskOf(typ),
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.Contains(script, "fog_action=debug") {
t.Errorf("debug script for %s should contain fog_action=debug", typ)
}
}
}

func TestGenerateScript_MemTest(t *testing.T) {
p := pxe.BootParams{
Host:      baseHost(),
Task:      taskOf(enttask.TypeMemtest),
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.HasPrefix(script, "#!ipxe") {
t.Error("memtest script does not start with #!ipxe")
}
}

func TestGenerateScript_LocalBoot(t *testing.T) {
p := pxe.BootParams{
Host:      baseHost(),
Task:      nil,
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.HasPrefix(script, "#!ipxe") {
t.Error("localBoot script does not start with #!ipxe")
}
if strings.Contains(script, "fog_action=deploy") {
t.Error("local boot script should not contain fog_action=deploy")
}
}

func TestGenerateScript_KernelArgsIncludeCustomArgs(t *testing.T) {
h := baseHost()
h.KernelArgs = "nomodeset console=ttyS0"
p := pxe.BootParams{
Host:      h,
Task:      taskOf(enttask.TypeDeploy),
ServerURL: "http://10.0.0.1",
KernelURL: "http://10.0.0.1/fog/kernel",
}
script := mustGenerate(t, p)
if !strings.Contains(script, "nomodeset") {
t.Error("script should include host KernelArgs")
}
if !strings.Contains(script, "console=ttyS0") {
t.Error("script should include host KernelArgs")
}
}
