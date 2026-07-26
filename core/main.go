package core

import (
	"context"
	"sync"

	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"

	sb "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	_ "github.com/sagernet/sing-box/experimental/clashapi"
	_ "github.com/sagernet/sing-box/experimental/v2rayapi"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	_ "github.com/sagernet/sing-box/transport/v2rayquic"
	"github.com/sagernet/sing/service"
)

type Core struct {
	access             sync.RWMutex
	wireGuardIPCAccess sync.Mutex
	ctx                context.Context
	isRunning          bool
	instance           *Box
	inboundManager     adapter.InboundManager
	outboundManager    adapter.OutboundManager
	serviceManager     adapter.ServiceManager
	endpointManager    adapter.EndpointManager
	router             adapter.Router
	factory            log.Factory
}

type coreRuntime struct {
	ctx             context.Context
	inboundManager  adapter.InboundManager
	outboundManager adapter.OutboundManager
	serviceManager  adapter.ServiceManager
	endpointManager adapter.EndpointManager
	router          adapter.Router
	factory         log.Factory
}

func NewCore() *Core {
	ctx := context.Background()
	ctx = sb.Context(ctx, InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry())
	return &Core{
		ctx:       ctx,
		isRunning: false,
		instance:  nil,
	}
}

func (c *Core) GetCtx() context.Context {
	c.access.RLock()
	defer c.access.RUnlock()
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *Core) GetInstance() *Box {
	c.access.RLock()
	defer c.access.RUnlock()
	return c.instance
}

func (c *Core) Start(sbConfig []byte) error {
	c.wireGuardIPCAccess.Lock()
	defer c.wireGuardIPCAccess.Unlock()
	var opt option.Options
	ctx := c.GetCtx()
	err := opt.UnmarshalJSONContext(ctx, sbConfig)
	if err != nil {
		// Returning the error (instead of only logging it) is essential: with a
		// zero/partial opt, NewBox+Start succeed on an empty option set and Start
		// would otherwise return nil, so the caller marks the core "running" while
		// no inbound is actually listening — a silent total outage behind a green
		// status. Surfacing the error lets startCoreLocked call markCoreStartFailed.
		logger.Error("Unmarshal config err:", err.Error())
		return err
	}

	instance, err := NewBox(Options{
		Context: ctx,
		Options: opt,
	})
	if err != nil {
		return err
	}

	err = instance.Start()
	if err != nil {
		_ = instance.Close()
		return err
	}

	ctx = service.ContextWith(ctx, c)

	c.access.Lock()
	c.ctx = ctx
	c.instance = instance
	c.isRunning = true
	c.inboundManager = instance.Inbound()
	c.outboundManager = instance.Outbound()
	c.serviceManager = instance.Service()
	c.endpointManager = instance.Endpoint()
	c.router = instance.Router()
	c.factory = instance.LogFactory()
	c.access.Unlock()
	return nil
}

func (c *Core) Stop() error {
	c.wireGuardIPCAccess.Lock()
	defer c.wireGuardIPCAccess.Unlock()
	c.access.Lock()
	c.isRunning = false
	if c.instance == nil {
		c.access.Unlock()
		return nil
	}
	instance := c.instance
	c.instance = nil
	c.inboundManager = nil
	c.outboundManager = nil
	c.serviceManager = nil
	c.endpointManager = nil
	c.router = nil
	c.factory = nil
	c.access.Unlock()
	err := instance.Close()
	return err
}

func (c *Core) IsRunning() bool {
	c.access.RLock()
	defer c.access.RUnlock()
	return c.isRunning
}

func (c *Core) runtime() (coreRuntime, bool) {
	c.access.RLock()
	defer c.access.RUnlock()
	if !c.isRunning || c.instance == nil {
		return coreRuntime{}, false
	}
	return coreRuntime{
		ctx:             c.ctx,
		inboundManager:  c.inboundManager,
		outboundManager: c.outboundManager,
		serviceManager:  c.serviceManager,
		endpointManager: c.endpointManager,
		router:          c.router,
		factory:         c.factory,
	}, true
}

// withRuntime runs fn with a consistent snapshot of the live managers while
// HOLDING the read lock for the whole call. This serializes hot-reload manager
// mutations (Add*/Remove*) against Stop()/startCoreLocked (which take the write
// lock and tear the managers down): without it, a deplete-cron RemoveInbound
// could operate on a manager whose underlying box was concurrently closed,
// panicking inside sing-box. The managers do not call back into Core, so holding
// the read lock across the mutation cannot deadlock.
func (c *Core) withRuntime(fn func(coreRuntime) error) error {
	c.access.RLock()
	defer c.access.RUnlock()
	if !c.isRunning || c.instance == nil {
		return common.NewError("sing-box is not running")
	}
	return fn(coreRuntime{
		ctx:             c.ctx,
		inboundManager:  c.inboundManager,
		outboundManager: c.outboundManager,
		serviceManager:  c.serviceManager,
		endpointManager: c.endpointManager,
		router:          c.router,
		factory:         c.factory,
	})
}

func (c *Core) Router() adapter.Router {
	c.access.RLock()
	defer c.access.RUnlock()
	return c.router
}

func (c *Core) OutboundManager() adapter.OutboundManager {
	c.access.RLock()
	defer c.access.RUnlock()
	return c.outboundManager
}
