//go:build with_ccm && (!darwin || cgo)

package core

import (
	"github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/service/ccm"
)

func registerCCMService(registry *service.Registry) {
	ccm.RegisterService(registry)
}
