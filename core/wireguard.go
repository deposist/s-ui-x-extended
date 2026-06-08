//go:build with_wireguard

package core

import (
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/protocol/warp"
	"github.com/sagernet/sing-box/protocol/wireguard"
)

func registerWireGuardEndpoint(registry *endpoint.Registry) {
	wireguard.RegisterEndpoint(registry)
	warp.RegisterEndpoint(registry)
}
