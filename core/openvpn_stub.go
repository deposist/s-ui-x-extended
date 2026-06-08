//go:build !with_openvpn

package core

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func registerOpenVPNOutbound(registry *outbound.Registry) {
	outbound.Register[option.OpenVPNOutboundOptions](registry, C.TypeOpenVPN, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.OpenVPNOutboundOptions) (adapter.Outbound, error) {
		return nil, E.New(`OpenVPN outbound is not included in this build, rebuild with -tags with_openvpn`)
	})
}
