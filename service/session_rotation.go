package service

import "sync"

// SessionRegenerateKey is a session value the login flow sets to ask the session
// store to mint a fresh session ID on the next Save, preventing session fixation
// (the pre-auth CSRF session would otherwise keep its ID after authentication).
// It lives here so both the api login handler and the web session store share a
// single source of truth without an import cycle (web imports api).
const SessionRegenerateKey = "__sui_session_regenerate__"

var wsTokenInvalidationHooks = struct {
	sync.Mutex
	byName map[string]func() int
}{
	byName: map[string]func() int{},
}

func RegisterWSTokenInvalidationHook(name string, fn func() int) {
	if name == "" {
		return
	}
	wsTokenInvalidationHooks.Lock()
	defer wsTokenInvalidationHooks.Unlock()
	if fn == nil {
		delete(wsTokenInvalidationHooks.byName, name)
		return
	}
	wsTokenInvalidationHooks.byName[name] = fn
}

func invalidateWSTokensForSessionRotation() int {
	wsTokenInvalidationHooks.Lock()
	hooks := make([]func() int, 0, len(wsTokenInvalidationHooks.byName))
	for _, hook := range wsTokenInvalidationHooks.byName {
		hooks = append(hooks, hook)
	}
	wsTokenInvalidationHooks.Unlock()

	total := 0
	for _, hook := range hooks {
		total += hook()
	}
	return total
}
