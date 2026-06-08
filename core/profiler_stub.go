//go:build !with_profiler

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

func registerProfilerService(registry *service.Registry) {
	service.Register[option.ProfilerServiceOptions](registry, C.TypeProfiler, func(ctx context.Context, logger log.ContextLogger, tag string, options option.ProfilerServiceOptions) (adapter.Service, error) {
		return nil, E.New(`Profiler is not included in this build, rebuild with -tags with_profiler`)
	})
}
