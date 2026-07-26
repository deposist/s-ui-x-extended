package ssrf

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strconv"

	"github.com/deposist/s-ui-x-extended/util/common"
)

// LookupNetIPFunc resolves a hostname for the requested IP family.
type LookupNetIPFunc func(context.Context, string, string) ([]netip.Addr, error)

// DialContextFunc is the shape used by net/http transports.
type DialContextFunc func(context.Context, string, string) (net.Conn, error)

// NewPublicDialContext returns a dialer that resolves each hostname exactly once,
// rejects the complete answer if any address is unsafe, and then dials only the
// validated IP literals. This closes the DNS-rebinding gap between URL
// validation and the actual TCP connection.
func NewPublicDialContext(lookup LookupNetIPFunc, dial DialContextFunc) DialContextFunc {
	if lookup == nil {
		lookup = net.DefaultResolver.LookupNetIP
	}
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, portText, err := net.SplitHostPort(address)
		if err != nil {
			return nil, common.NewErrorf("invalid dial address: %v", err)
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return nil, common.NewError("invalid dial port")
		}

		lookupNetwork, err := lookupNetworkFor(network)
		if err != nil {
			return nil, err
		}
		addresses, err := resolvePublicAddresses(ctx, lookup, lookupNetwork, host)
		if err != nil {
			return nil, err
		}

		var dialErr error
		for _, addr := range addresses {
			conn, err := dial(ctx, network, net.JoinHostPort(addr.String(), portText))
			if err == nil {
				return conn, nil
			}
			dialErr = errors.Join(dialErr, err)
			if ctx.Err() != nil {
				break
			}
		}
		if dialErr == nil {
			dialErr = common.NewError("host did not resolve to a usable IP")
		}
		return nil, dialErr
	}
}

func lookupNetworkFor(network string) (string, error) {
	switch network {
	case "tcp":
		return "ip", nil
	case "tcp4":
		return "ip4", nil
	case "tcp6":
		return "ip6", nil
	default:
		return "", common.NewErrorf("unsupported dial network: %s", network)
	}
}

func resolvePublicAddresses(ctx context.Context, lookup LookupNetIPFunc, network, host string) ([]netip.Addr, error) {
	if literal, err := netip.ParseAddr(host); err == nil {
		literal = literal.Unmap()
		if literal.Zone() != "" || !matchesLookupNetwork(literal, network) || IsBlockedAddr(literal) {
			return nil, common.NewError("dial host is not allowed")
		}
		return []netip.Addr{literal}, nil
	}

	addresses, err := lookup(ctx, network, host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, common.NewError("dial host did not resolve")
	}
	seen := make(map[netip.Addr]struct{}, len(addresses))
	validated := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if !address.IsValid() || address.Zone() != "" || !matchesLookupNetwork(address, network) || IsBlockedAddr(address) {
			return nil, common.NewError("dial host resolves to a disallowed IP")
		}
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		validated = append(validated, address)
	}
	return validated, nil
}

func matchesLookupNetwork(address netip.Addr, network string) bool {
	switch network {
	case "ip4":
		return address.Is4()
	case "ip6":
		return address.Is6()
	default:
		return address.Is4() || address.Is6()
	}
}
