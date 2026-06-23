package core

import (
	"errors"
	"fmt"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/protocol/group"
)

var (
	// ErrGroupNotFound is returned when the named outbound group is not present
	// in the running core (e.g. between a config edit and the core restart).
	ErrGroupNotFound = errors.New("failover group not found")
	// ErrNotASelectorGroup is returned when the named outbound exists but is not
	// a selector-backed group, so its active member cannot be switched.
	ErrNotASelectorGroup = errors.New("outbound is not a selector group")
	// ErrMemberNotInGroup is returned when the requested member is not part of
	// the group's member list.
	ErrMemberNotInGroup = errors.New("member not in group")
)

// SelectGroupMember points the named selector group at memberTag. It is the
// ONLY place s-ui-x type-asserts to the concrete *group.Selector — exactly as
// sing-box's own clash API does (experimental/clashapi/proxies.go) — because
// SelectOutbound is not exposed on the adapter.OutboundGroup interface. The
// whole resolve+assert+select runs under withRuntime's read lock so it cannot
// race a core Stop()/restart. Callers must NOT cache the resolved selector:
// re-invoke each probe cycle so a post-restart selector is re-resolved.
func (c *Core) SelectGroupMember(groupTag, memberTag string) error {
	return c.withRuntime(func(rt coreRuntime) error {
		ob, ok := rt.outboundManager.Outbound(groupTag)
		if !ok {
			return fmt.Errorf("%w: %q", ErrGroupNotFound, groupTag)
		}
		selector, ok := ob.(*group.Selector)
		if !ok {
			return fmt.Errorf("%w: %q", ErrNotASelectorGroup, groupTag)
		}
		if !selector.SelectOutbound(memberTag) {
			return fmt.Errorf("%w: %q in %q", ErrMemberNotInGroup, memberTag, groupTag)
		}
		return nil
	})
}

// GroupNow returns the group's currently active member as the running core sees
// it (selector.Now()), read through the adapter.OutboundGroup interface (Now()
// IS on the interface; only the write path needs the concrete type). ok is
// false when the tag is not a running outbound group.
func (c *Core) GroupNow(groupTag string) (active string, ok bool) {
	_ = c.withRuntime(func(rt coreRuntime) error {
		ob, found := rt.outboundManager.Outbound(groupTag)
		if !found {
			return nil
		}
		grp, isGroup := ob.(adapter.OutboundGroup)
		if !isGroup {
			return nil
		}
		active = grp.Now()
		ok = true
		return nil
	})
	return active, ok
}
