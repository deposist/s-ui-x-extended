//go:build with_masque

package core

import (
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/protocol/masque"
)

func registerMASQUEOutbound(registry *outbound.Registry) {
	masque.RegisterOutbound(registry)
}
