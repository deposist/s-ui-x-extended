//go:build !with_oomkiller

package core

import (
	"context"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/service"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func registerOOMKillerService(registry *service.Registry) {
	service.Register[option.OOMKillerServiceOptions](registry, C.TypeOOMKiller, func(ctx context.Context, logger log.ContextLogger, tag string, options option.OOMKillerServiceOptions) (adapter.Service, error) {
		return nil, E.New(`OOM killer is not included in this build, rebuild with -tags with_oomkiller`)
	})
}
