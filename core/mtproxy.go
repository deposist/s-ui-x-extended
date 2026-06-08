//go:build with_mtproxy

package core

import (
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/protocol/mtproxy"
)

func registerMTProxyInbound(registry *inbound.Registry) {
	mtproxy.RegisterInbound(registry)
}
