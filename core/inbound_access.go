package core

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
	"golang.org/x/time/rate"
)

type inboundAccessRouter struct{ adapter.Router }

var activePeerConnections = struct {
	sync.Mutex
	byUser map[string]*peerConnectionPool
}{byUser: make(map[string]*peerConnectionPool)}

type peerConnectionPool struct {
	policy ipmonitor.ConnectionPolicy
	count  uint32
	up     *rate.Limiter
	down   *rate.Limiter
}

func registerAccessInbound[Options any](registry *inbound.Registry, protocol string, constructor inbound.ConstructorFunc[Options]) {
	inbound.Register(registry, protocol, func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options Options) (adapter.Inbound, error) {
		return constructor(ctx, inboundAccessRouter{router}, logger, tag, options)
	})
}

func (router inboundAccessRouter) RouteConnectionEx(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, onClose network.CloseHandlerFunc) {
	if pool, done, allowed := admitInbound(ctx, conn, metadata, onClose); allowed {
		if pool.metersTraffic() {
			limitCtx, cancel := context.WithCancel(ctx)
			conn = peerRateConn{Conn: conn, ctx: limitCtx, cancel: cancel, pool: pool}
			done = network.AppendClose(done, func(error) { cancel() })
		}
		router.Router.RouteConnectionEx(ctx, conn, metadata, done)
	}
}

func (router inboundAccessRouter) RoutePacketConnectionEx(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, onClose network.CloseHandlerFunc) {
	if pool, done, allowed := admitInbound(ctx, conn, metadata, onClose); allowed {
		if pool.metersTraffic() {
			limitCtx, cancel := context.WithCancel(ctx)
			conn = peerRatePacketConn{PacketConn: conn, ctx: limitCtx, cancel: cancel, pool: pool}
			done = network.AppendClose(done, func(error) { cancel() })
		}
		router.Router.RoutePacketConnectionEx(ctx, conn, metadata, done)
	}
}

func admitInbound(ctx context.Context, conn io.Closer, metadata adapter.InboundContext, onClose network.CloseHandlerFunc) (*peerConnectionPool, network.CloseHandlerFunc, bool) {
	allowed, expiry, policy := ipmonitor.ObserveConnection(ctx, metadata.User, sourceIPFromMetadata(metadata))
	var pool *peerConnectionPool
	if allowed && policy != (ipmonitor.ConnectionPolicy{}) {
		activePeerConnections.Lock()
		pool = activePeerConnections.byUser[metadata.User]
		if pool == nil || pool.policy != policy {
			pool = &peerConnectionPool{policy: policy, up: peerLimiter(policy.Upload), down: peerLimiter(policy.Download)}
			if policy.Speed != 0 {
				pool.up = peerLimiter(policy.Speed)
				pool.down = pool.up
			}
			activePeerConnections.byUser[metadata.User] = pool
		}
		allowed = policy.Connections == 0 || pool.count < policy.Connections
		if allowed {
			pool.count++
		}
		activePeerConnections.Unlock()
	}
	if !allowed {
		_ = conn.Close()
		if onClose != nil {
			onClose(net.ErrClosed)
		}
		return nil, nil, false
	}
	if expiry == 0 && pool == nil {
		return nil, onClose, true
	}
	timer := closeAtExpiry(conn, expiry)
	return pool, network.OnceClose(func(err error) {
		if timer != nil {
			timer.Stop()
		}
		if pool != nil {
			activePeerConnections.Lock()
			pool.count--
			if pool.count == 0 {
				time.AfterFunc(time.Second, func() {
					activePeerConnections.Lock()
					defer activePeerConnections.Unlock()
					if pool.count == 0 && activePeerConnections.byUser[metadata.User] == pool {
						delete(activePeerConnections.byUser, metadata.User)
					}
				})
			}
			activePeerConnections.Unlock()
		}
		if onClose != nil {
			onClose(err)
		}
	}), true
}

func (pool *peerConnectionPool) metersTraffic() bool {
	return pool != nil && (pool.up != nil || pool.down != nil || pool.policy.Volume > 0)
}

func (pool *peerConnectionPool) charge(ctx context.Context, limiter *rate.Limiter, length int) error {
	if err := waitPeerRate(ctx, limiter, length); err != nil {
		return err
	}
	policy := pool.policy
	if length == 0 || policy.Volume == 0 {
		return nil
	}
	if int64(length) > policy.Volume || policy.ClientID == 0 || policy.NodeUUID == "" {
		return errors.New("node traffic quota exhausted")
	}
	leave, ok := database.TryEnterDBOperation()
	if !ok {
		return errors.New("node traffic accounting unavailable")
	}
	defer leave()
	db := database.GetDB()
	if db == nil {
		return errors.New("node traffic accounting unavailable")
	}
	result := db.WithContext(ctx).Exec(`INSERT INTO peer_traffics(node_uuid, client_id, used) VALUES (?, ?, ?)
		ON CONFLICT(node_uuid, client_id) DO UPDATE SET used = used + excluded.used WHERE used <= ?`,
		policy.NodeUUID, policy.ClientID, length, policy.Volume-int64(length))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("node traffic quota exhausted")
	}
	return nil
}

func peerLimiter(speed uint64) *rate.Limiter {
	if speed == 0 {
		return nil
	}
	burst := 64 << 10
	if speed < 64<<10 {
		burst = int(speed)
	}
	return rate.NewLimiter(rate.Limit(speed), burst)
}

func waitPeerRate(ctx context.Context, limiter *rate.Limiter, length int) error {
	if limiter == nil {
		return nil
	}
	for length > 0 {
		chunk := min(length, limiter.Burst())
		if err := limiter.WaitN(ctx, chunk); err != nil {
			return err
		}
		length -= chunk
	}
	return nil
}

type peerRateConn struct {
	net.Conn
	ctx    context.Context
	cancel context.CancelFunc
	pool   *peerConnectionPool
}

func (conn peerRateConn) Upstream() any { return conn.Conn }

func (conn peerRateConn) Close() error {
	conn.cancel()
	return conn.Conn.Close()
}

func (conn peerRateConn) Read(data []byte) (int, error) {
	length, err := conn.Conn.Read(data)
	if limitErr := conn.pool.charge(conn.ctx, conn.pool.up, length); limitErr != nil {
		return 0, limitErr
	}
	return length, err
}

func (conn peerRateConn) Write(data []byte) (int, error) {
	if err := conn.pool.charge(conn.ctx, conn.pool.down, len(data)); err != nil {
		return 0, err
	}
	return conn.Conn.Write(data)
}

type peerRatePacketConn struct {
	network.PacketConn
	ctx    context.Context
	cancel context.CancelFunc
	pool   *peerConnectionPool
}

func (conn peerRatePacketConn) Upstream() any { return conn.PacketConn }

func (conn peerRatePacketConn) Close() error {
	conn.cancel()
	return conn.PacketConn.Close()
}

func (conn peerRatePacketConn) ReadPacket(buffer *buf.Buffer) (metadata.Socksaddr, error) {
	destination, err := conn.PacketConn.ReadPacket(buffer)
	if err == nil {
		err = conn.pool.charge(conn.ctx, conn.pool.up, buffer.Len())
	}
	return destination, err
}

func (conn peerRatePacketConn) WritePacket(buffer *buf.Buffer, destination metadata.Socksaddr) error {
	if err := conn.pool.charge(conn.ctx, conn.pool.down, buffer.Len()); err != nil {
		buffer.Release()
		return err
	}
	return conn.PacketConn.WritePacket(buffer, destination)
}
