package core

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestServiceRegistryIncludesOOMKiller(t *testing.T) {
	registry := ServiceRegistry()

	rawOptions, ok := registry.CreateOptions(C.TypeOOMKiller)
	if !ok {
		t.Fatal("oom-killer service options are not registered")
	}

	if _, ok := rawOptions.(*option.OOMKillerServiceOptions); !ok {
		t.Fatalf("oom-killer options type = %T", rawOptions)
	}
}
