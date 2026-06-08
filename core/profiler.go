//go:build with_profiler

package core

import (
	"github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/service/profiler"
)

func registerProfilerService(registry *service.Registry) {
	profiler.RegisterService(registry)
}
