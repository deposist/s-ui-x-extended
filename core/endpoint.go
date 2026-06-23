package core

import (
	"github.com/deposist/s-ui-x-extended/logger"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
)

func (c *Core) AddInbound(config []byte) error {
	return c.withRuntime(func(rt coreRuntime) error {
		var inbound_config option.Inbound
		if err := inbound_config.UnmarshalJSONContext(rt.ctx, config); err != nil {
			return err
		}
		return rt.inboundManager.Create(
			rt.ctx,
			rt.router,
			rt.factory.NewLogger("inbound/"+inbound_config.Type+"["+inbound_config.Tag+"]"),
			inbound_config.Tag,
			inbound_config.Type,
			inbound_config.Options)
	})
}

func (c *Core) RemoveInbound(tag string) error {
	return c.withRuntime(func(rt coreRuntime) error {
		logger.Info("remove inbound: ", tag)
		return rt.inboundManager.Remove(tag)
	})
}

func (c *Core) AddOutbound(config []byte) error {
	return c.withRuntime(func(rt coreRuntime) error {
		var outbound_config option.Outbound
		if err := outbound_config.UnmarshalJSONContext(rt.ctx, config); err != nil {
			return err
		}
		outboundCtx := adapter.WithContext(rt.ctx, &adapter.InboundContext{
			Outbound: outbound_config.Tag,
		})
		return rt.outboundManager.Create(
			outboundCtx,
			rt.router,
			rt.factory.NewLogger("outbound/"+outbound_config.Type+"["+outbound_config.Tag+"]"),
			outbound_config.Tag,
			outbound_config.Type,
			outbound_config.Options)
	})
}

func (c *Core) RemoveOutbound(tag string) error {
	return c.withRuntime(func(rt coreRuntime) error {
		logger.Info("remove outbound: ", tag)
		return rt.outboundManager.Remove(tag)
	})
}

func (c *Core) AddEndpoint(config []byte) error {
	return c.withRuntime(func(rt coreRuntime) error {
		var endpoint_config option.Endpoint
		if err := endpoint_config.UnmarshalJSONContext(rt.ctx, config); err != nil {
			return err
		}
		return rt.endpointManager.Create(
			rt.ctx,
			rt.router,
			rt.factory.NewLogger("endpoint/"+endpoint_config.Type+"["+endpoint_config.Tag+"]"),
			endpoint_config.Tag,
			endpoint_config.Type,
			endpoint_config.Options)
	})
}

func (c *Core) RemoveEndpoint(tag string) error {
	return c.withRuntime(func(rt coreRuntime) error {
		logger.Info("remove endpoint: ", tag)
		return rt.endpointManager.Remove(tag)
	})
}

func (c *Core) AddService(config []byte) error {
	return c.withRuntime(func(rt coreRuntime) error {
		var srv_config option.Service
		if err := srv_config.UnmarshalJSONContext(rt.ctx, config); err != nil {
			return err
		}
		return rt.serviceManager.Create(
			rt.ctx,
			rt.factory.NewLogger("service/"+srv_config.Type+"["+srv_config.Tag+"]"),
			srv_config.Tag,
			srv_config.Type,
			srv_config.Options)
	})
}

func (c *Core) RemoveService(tag string) error {
	return c.withRuntime(func(rt coreRuntime) error {
		logger.Info("remove service: ", tag)
		return rt.serviceManager.Remove(tag)
	})
}
