// Package plugins defines the compile-time hook interface for FOG Next.
//
// Plugins are Go structs that implement one or more hook interfaces. They are
// registered at startup and called synchronously at the defined extension
// points. There are no runtime .so / .dll plugins — users who need custom
// behaviour implement the interfaces in their own fork or contribute upstream.
//
// # Registering a plugin
//
// In your main package (or a dedicated init file), call Register before
// starting the server:
//
//	func init() {
//	    plugins.Register(&MyPlugin{})
//	}
//
// The Registry is safe for concurrent use after startup — Register must only
// be called during initialisation (before serving requests).
package plugins

import (
	"context"

	"github.com/nemvince/fog-next/ent"
)

// ─── Hook interfaces ──────────────────────────────────────────────────────────

// HostHook is implemented by plugins that want to react to host lifecycle
// events.
type HostHook interface {
	// OnHostRegister is called when a new host is registered via PXE boot.
	OnHostRegister(ctx context.Context, host *ent.Host) error

	// OnHostDelete is called just before a host is permanently removed.
	OnHostDelete(ctx context.Context, host *ent.Host) error
}

// TaskHook is implemented by plugins that want to intercept task lifecycle
// events.
type TaskHook interface {
	// BeforeTaskCreate is called before a new task is persisted.
	// Returning a non-nil error prevents the task from being created.
	BeforeTaskCreate(ctx context.Context, task *ent.Task) error

	// AfterTaskComplete is called after a task transitions to the complete
	// or failed state. The error is informational only — it does not affect
	// the task record.
	AfterTaskComplete(ctx context.Context, task *ent.Task) error
}

// ImageHook is implemented by plugins that want to react to image lifecycle
// events.
type ImageHook interface {
	// AfterImageCapture is called after a successful image capture upload.
	AfterImageCapture(ctx context.Context, image *ent.Image) error

	// AfterImageDeploy is called after a successful image deployment.
	AfterImageDeploy(ctx context.Context, image *ent.Image, host *ent.Host) error
}

// InventoryHook is implemented by plugins that process hardware inventory
// reported by PXE-booting clients.
type InventoryHook interface {
	// OnInventoryUpdate is called whenever a host reports new inventory data.
	OnInventoryUpdate(ctx context.Context, inv *ent.Inventory) error
}

// ─── Registry ─────────────────────────────────────────────────────────────────

// Registry holds all registered plugins and dispatches hook calls.
// The zero value is usable via the DefaultRegistry package-level variable.
type Registry struct {
	hostHooks      []HostHook
	taskHooks      []TaskHook
	imageHooks     []ImageHook
	inventoryHooks []InventoryHook
}

// DefaultRegistry is the global plugin registry used by the server.
var DefaultRegistry = &Registry{}

// Register adds a plugin to the DefaultRegistry. It is safe to call from
// multiple init() functions.
func Register(p any) { DefaultRegistry.Register(p) }

// Register adds a plugin to the registry. Each interface the plugin implements
// is registered independently so a single struct can satisfy multiple hooks.
func (reg *Registry) Register(p any) {
	if h, ok := p.(HostHook); ok {
		reg.hostHooks = append(reg.hostHooks, h)
	}
	if h, ok := p.(TaskHook); ok {
		reg.taskHooks = append(reg.taskHooks, h)
	}
	if h, ok := p.(ImageHook); ok {
		reg.imageHooks = append(reg.imageHooks, h)
	}
	if h, ok := p.(InventoryHook); ok {
		reg.inventoryHooks = append(reg.inventoryHooks, h)
	}
}

// ─── Dispatcher methods ───────────────────────────────────────────────────────
// All dispatcher methods skip remaining plugins if one returns an error.

// OnHostRegister dispatches to all registered HostHook plugins.
func (reg *Registry) OnHostRegister(ctx context.Context, host *ent.Host) error {
	for _, h := range reg.hostHooks {
		if err := h.OnHostRegister(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

// OnHostDelete dispatches to all registered HostHook plugins.
func (reg *Registry) OnHostDelete(ctx context.Context, host *ent.Host) error {
	for _, h := range reg.hostHooks {
		if err := h.OnHostDelete(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

// BeforeTaskCreate dispatches to all registered TaskHook plugins.
// Returns the first error encountered, preventing the task from being created.
func (reg *Registry) BeforeTaskCreate(ctx context.Context, task *ent.Task) error {
	for _, h := range reg.taskHooks {
		if err := h.BeforeTaskCreate(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// AfterTaskComplete dispatches to all registered TaskHook plugins.
func (reg *Registry) AfterTaskComplete(ctx context.Context, task *ent.Task) error {
	for _, h := range reg.taskHooks {
		if err := h.AfterTaskComplete(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// AfterImageCapture dispatches to all registered ImageHook plugins.
func (reg *Registry) AfterImageCapture(ctx context.Context, image *ent.Image) error {
	for _, h := range reg.imageHooks {
		if err := h.AfterImageCapture(ctx, image); err != nil {
			return err
		}
	}
	return nil
}

// AfterImageDeploy dispatches to all registered ImageHook plugins.
func (reg *Registry) AfterImageDeploy(ctx context.Context, image *ent.Image, host *ent.Host) error {
	for _, h := range reg.imageHooks {
		if err := h.AfterImageDeploy(ctx, image, host); err != nil {
			return err
		}
	}
	return nil
}

// OnInventoryUpdate dispatches to all registered InventoryHook plugins.
func (reg *Registry) OnInventoryUpdate(ctx context.Context, inv *ent.Inventory) error {
	for _, h := range reg.inventoryHooks {
		if err := h.OnInventoryUpdate(ctx, inv); err != nil {
			return err
		}
	}
	return nil
}

// ─── Built-in no-op plugin ────────────────────────────────────────────────────

// Noop is a convenience base type that satisfies all hook interfaces with
// no-op implementations. Embed it in your plugin struct to only override
// the hooks you care about.
type Noop struct{}

func (Noop) OnHostRegister(_ context.Context, _ *ent.Host) error    { return nil }
func (Noop) OnHostDelete(_ context.Context, _ *ent.Host) error      { return nil }
func (Noop) BeforeTaskCreate(_ context.Context, _ *ent.Task) error  { return nil }
func (Noop) AfterTaskComplete(_ context.Context, _ *ent.Task) error { return nil }
func (Noop) AfterImageCapture(_ context.Context, _ *ent.Image) error { return nil }
func (Noop) AfterImageDeploy(_ context.Context, _ *ent.Image, _ *ent.Host) error {
	return nil
}
func (Noop) OnInventoryUpdate(_ context.Context, _ *ent.Inventory) error { return nil }
