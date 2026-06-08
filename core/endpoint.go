package core

import (
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/option"
)

func (c *Core) AddInbound(config []byte) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	var err error
	var inbound_config option.Inbound
	err = inbound_config.UnmarshalJSONContext(rt.ctx, config)
	if err != nil {
		return err
	}

	err = rt.inboundManager.Create(
		rt.ctx,
		rt.router,
		rt.factory.NewLogger("inbound/"+inbound_config.Type+"["+inbound_config.Tag+"]"),
		inbound_config.Tag,
		inbound_config.Type,
		inbound_config.Options)
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) RemoveInbound(tag string) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	logger.Info("remove inbound: ", tag)
	return rt.inboundManager.Remove(tag)
}

func (c *Core) AddOutbound(config []byte) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	var err error
	var outbound_config option.Outbound

	err = outbound_config.UnmarshalJSONContext(rt.ctx, config)
	if err != nil {
		return err
	}

	outboundCtx := adapter.WithContext(rt.ctx, &adapter.InboundContext{
		Outbound: outbound_config.Tag,
	})

	err = rt.outboundManager.Create(
		outboundCtx,
		rt.router,
		rt.factory.NewLogger("outbound/"+outbound_config.Type+"["+outbound_config.Tag+"]"),
		outbound_config.Tag,
		outbound_config.Type,
		outbound_config.Options)
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) RemoveOutbound(tag string) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	logger.Info("remove outbound: ", tag)
	return rt.outboundManager.Remove(tag)
}

func (c *Core) AddEndpoint(config []byte) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	var err error
	var endpoint_config option.Endpoint

	err = endpoint_config.UnmarshalJSONContext(rt.ctx, config)
	if err != nil {
		return err
	}

	err = rt.endpointManager.Create(
		rt.ctx,
		rt.router,
		rt.factory.NewLogger("endpoint/"+endpoint_config.Type+"["+endpoint_config.Tag+"]"),
		endpoint_config.Tag,
		endpoint_config.Type,
		endpoint_config.Options)
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) RemoveEndpoint(tag string) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	logger.Info("remove endpoint: ", tag)
	return rt.endpointManager.Remove(tag)
}

func (c *Core) AddService(config []byte) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	var err error
	var srv_config option.Service

	err = srv_config.UnmarshalJSONContext(rt.ctx, config)
	if err != nil {
		return err
	}

	err = rt.serviceManager.Create(
		rt.ctx,
		rt.factory.NewLogger("service/"+srv_config.Type+"["+srv_config.Tag+"]"),
		srv_config.Tag,
		srv_config.Type,
		srv_config.Options)
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) RemoveService(tag string) error {
	rt, ok := c.runtime()
	if !ok {
		return common.NewError("sing-box is not running")
	}
	logger.Info("remove service: ", tag)
	return rt.serviceManager.Remove(tag)
}
