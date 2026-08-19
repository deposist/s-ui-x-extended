package database

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var resetHooks = struct {
	sync.Mutex
	byName map[string]func() error
}{
	byName: map[string]func() error{},
}

func RegisterResetHook(name string, fn func() error) {
	if name == "" {
		return
	}
	resetHooks.Lock()
	defer resetHooks.Unlock()
	if fn == nil {
		delete(resetHooks.byName, name)
		return
	}
	resetHooks.byName[name] = fn
}

func ResetCaches(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	resetHooks.Lock()
	names := make([]string, 0, len(resetHooks.byName))
	for name := range resetHooks.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	hooks := make([]func() error, 0, len(names))
	for _, name := range names {
		hooks = append(hooks, resetHooks.byName[name])
	}
	resetHooks.Unlock()

	var hookErrs []error
	for _, hook := range hooks {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := hook(); err != nil {
			hookErrs = append(hookErrs, err)
		}
	}
	if err := ctx.Err(); err != nil {
		hookErrs = append(hookErrs, err)
	}
	return errors.Join(hookErrs...)
}
