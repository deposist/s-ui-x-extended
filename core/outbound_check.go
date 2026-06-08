package core

import (
	"context"
	"time"

	urltest "github.com/sagernet/sing-box/common/urltest"
)

const checkTimeout = 15 * time.Second

type CheckOutboundResult struct {
	OK    bool
	Delay uint16
	Error string
}

func (c *Core) CheckOutbound(ctx context.Context, tag string, link string) (result CheckOutboundResult) {
	outboundManager := c.OutboundManager()
	if outboundManager == nil {
		result.Error = "core not running"
		return result
	}
	ob, ok := outboundManager.Outbound(tag)
	if !ok {
		result.Error = "outbound not found"
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	delay, err := urltest.URLTest(ctx, link, ob)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.OK = true
	result.Delay = delay
	return result
}
