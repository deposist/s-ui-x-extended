package core

import (
	"errors"
	"fmt"

	CM "github.com/sagernet/sing-box/service/manager/constant"
)

// ErrFleetUnavailable is returned when the running core has no manager service,
// either because the build lacks -tags with_manager, because the controller role
// is not configured, or because the core is between restarts.
var ErrFleetUnavailable = errors.New("fleet manager is not available")

// WithFleetManager runs fn against the manager service of the running core.
//
// The manager pointer must never escape: a concurrent core restart destroys it,
// so every call re-resolves it under withRuntime's read lock, which also keeps
// the runtime stable for the whole callback. Callers must not cache the
// *manager.Service they receive.
func (c *Core) WithFleetManager(fn func(manager CM.Manager) error) error {
	return c.withRuntime(func(rt coreRuntime) error {
		if rt.serviceManager == nil {
			return ErrFleetUnavailable
		}
		found := false
		for _, service := range rt.serviceManager.Services() {
			manager, ok := service.(CM.Manager)
			if !ok {
				continue
			}
			found = true
			if err := fn(manager); err != nil {
				return err
			}
		}
		if !found {
			return fmt.Errorf("%w: no manager service is running", ErrFleetUnavailable)
		}
		return nil
	})
}
