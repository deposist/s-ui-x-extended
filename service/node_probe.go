package service

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util"
	"github.com/deposist/s-ui-x-extended/util/ssrf"
	"github.com/gofrs/uuid/v5"
	M "github.com/sagernet/sing/common/metadata"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const PeerProbeHeader = "X-Node-Probe-V1"

const (
	nodeProbeDuration = 90 * time.Second
	nodeProbeTimeout  = 12 * time.Second
	maxNodeProbes     = 4
)

type PeerProbeResult struct {
	InboundID uint   `json:"inboundId"`
	Stage     string `json:"stage"`
	Result    string `json:"result"`
	CheckedAt int64  `json:"checkedAt"`
	LatencyMS int64  `json:"latencyMs,omitempty"`
	From      string `json:"from"`
}

type peerProbeAccess struct {
	ID          string `json:"id"`
	InboundID   uint   `json:"inboundId"`
	Credential  string `json:"credential"`
	Destination string `json:"destination"`
	Until       int64  `json:"until"`
}

func (a peerProbeAccess) user() peerUser {
	return peerUser{InboundID: a.InboundID, ClientID: 1, Type: "vless", Name: "node-probe-" + a.ID,
		UUID: a.Credential, Expiry: a.Until, Connections: 1, ProbeDestination: a.Destination}
}

func activePeerProbeHeader(data json.RawMessage) string {
	var users []peerUser
	if json.Unmarshal(data, &users) == nil {
		for _, user := range users {
			if user.ProbeDestination != "" && user.Expiry > time.Now().Unix() {
				return strings.TrimPrefix(user.Name, "node-probe-")
			}
		}
	}
	return "1"
}

func newPeerProbeAccess(inbound PeerInbound, destination netip.AddrPort, expiry time.Time) (peerProbeAccess, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return peerProbeAccess{}, err
	}
	return peerProbeAccess{ID: rand.Text(), Credential: id.String(), InboundID: inbound.ID,
		Destination: destination.String(), Until: expiry.Unix()}, nil
}

func peerProbeEligible(inbound PeerInbound) bool {
	return inbound.Type == "vless" && inbound.Link != nil && inbound.Link.valid(inbound.Type) &&
		(inbound.Link.Transport == nil || inbound.Link.Transport.Type == "tcp")
}

type peerProbeKey struct {
	nodeUUID  string
	inboundID uint
}

type peerProbeJob struct {
	result          PeerProbeResult
	inventoryDigest string
	access          *peerProbeAccess
	ready           chan struct{}
	db              *gorm.DB
}

type peerProbes struct {
	sync.Mutex
	jobs map[peerProbeKey]*peerProbeJob
}

func (p *peerProbes) results(nodeUUID, digest string) []PeerProbeResult {
	p.Lock()
	defer p.Unlock()
	var results []PeerProbeResult
	for key, job := range p.jobs {
		if key.nodeUUID == nodeUUID && job.inventoryDigest == digest && job.db == database.GetDB() {
			results = append(results, job.result)
		}
	}
	return results
}

func (p *peerProbes) exchange(nodeUUID, digest, ack string, now time.Time) *peerProbeAccess {
	p.Lock()
	defer p.Unlock()
	for key, job := range p.jobs {
		if key.nodeUUID != nodeUUID || job.db != database.GetDB() || job.inventoryDigest != digest || job.access == nil || job.access.Until <= now.Unix() {
			continue
		}
		if ack == job.access.ID {
			select {
			case job.ready <- struct{}{}:
			default:
			}
		}
		return job.access
	}
	return nil
}

func (s *NodeService) StartPeerProbe(nodeUUID string, inboundID uint) (PeerProbeResult, error) {
	node, err := editablePeerInbound(s.db(), nodeUUID, inboundID)
	if err != nil {
		return PeerProbeResult{}, err
	}
	var inbound PeerInbound
	for _, candidate := range peerInboundsFromInventory(node.Inventory) {
		if candidate.ID == inboundID {
			inbound = candidate
			break
		}
	}
	if !peerProbeEligible(inbound) || node.LinkHost == "" {
		return PeerProbeResult{}, errors.New("select a published VLESS TCP inbound for the check")
	}
	if !node.PeerLeaseCapable {
		return PeerProbeResult{}, errors.New("the Slave must support confirmed access before it can run a check")
	}
	control, err := url.Parse(os.Getenv("SUI_NODE_PROBE_URL"))
	if err != nil || control.Scheme != "https" || control.User != nil || control.Hostname() == "" || control.Fragment != "" {
		return PeerProbeResult{}, errors.New("configure the Master HTTPS control address in SUI_NODE_PROBE_URL")
	}
	var roots *x509.CertPool
	if file := os.Getenv("SUI_NODE_PROBE_CA_FILE"); file != "" {
		pem, err := os.ReadFile(file) // #nosec G304 G703 -- The Master process operator selects the control CA file.
		roots = x509.NewCertPool()
		if err != nil || !roots.AppendCertsFromPEM(pem) {
			return PeerProbeResult{}, errors.New("cannot read HTTPS control CA from SUI_NODE_PROBE_CA_FILE")
		}
	}
	probes := &s.runtime().nodeProbes
	probes.Lock()
	defer probes.Unlock()
	if probes.jobs == nil {
		probes.jobs = make(map[peerProbeKey]*peerProbeJob)
	}
	key := peerProbeKey{nodeUUID: nodeUUID, inboundID: inboundID}
	now := s.peerTime()
	active := 0
	for existing, job := range probes.jobs {
		if job.result.Result == "running" {
			if existing == key {
				return job.result, nil
			}
			if existing.nodeUUID == nodeUUID {
				return PeerProbeResult{}, errors.New("another check is running on this Slave")
			}
			active++
		} else if job.result.CheckedAt < now.Add(-24*time.Hour).Unix() {
			delete(probes.jobs, existing)
		}
	}
	if active >= maxNodeProbes || (len(probes.jobs) >= 512 && probes.jobs[key] == nil) {
		return PeerProbeResult{}, errors.New("too many node checks; try again later")
	}
	if last := probes.jobs[key]; last != nil && now.Unix()-last.result.CheckedAt < 10 {
		return last.result, nil
	}
	result := PeerProbeResult{InboundID: inboundID, Stage: "admission", Result: "running", CheckedAt: now.Unix(), From: "master"}
	job := &peerProbeJob{result: result, inventoryDigest: nodeSecretDigest(node.LinkHost + "\x00" + string(node.Inventory)), ready: make(chan struct{}, 1), db: s.db()}
	probes.jobs[key] = job
	go probes.run(key, job, inbound, node.LinkHost, control, roots, os.Getenv("SUI_NODE_PROBE_LOOPBACK") == "1")
	return result, nil
}

func resolveProbeHost(ctx context.Context, host string, allowLoopback bool) (netip.Addr, error) {
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return netip.Addr{}, errors.New("probe address resolution failed")
	}
	for _, addr := range addresses {
		if ssrf.IsBlockedAddr(addr) && !(allowLoopback && addr.Unmap().IsLoopback()) {
			return netip.Addr{}, errors.New("probe address is not public")
		}
	}
	return addresses[0].Unmap(), nil
}

func (p *peerProbes) run(key peerProbeKey, job *peerProbeJob, inbound PeerInbound, host string, control *url.URL, roots *x509.CertPool, allowLoopback bool) {
	ctx, cancel := context.WithTimeout(context.Background(), nodeProbeDuration)
	defer cancel()
	stage, result, latency := "admission", "address_rejected", int64(0)
	defer func() {
		p.Lock()
		job.access = nil
		job.result.Stage, job.result.Result, job.result.LatencyMS = stage, result, latency
		job.result.CheckedAt = time.Now().Unix()
		finished := job.result
		p.Unlock()
		if err := storePeerProbe(job.db, key, job.inventoryDigest, finished); err != nil {
			logger.Warning("cannot store the node traffic check result")
		}
	}()
	dnsContext, stopDNS := context.WithTimeout(ctx, 5*time.Second)
	peerAddress, err := resolveProbeHost(dnsContext, host, allowLoopback)
	controlAddress, controlErr := resolveProbeHost(dnsContext, control.Hostname(), allowLoopback)
	stopDNS()
	if err != nil || controlErr != nil {
		return
	}
	port := uint64(443)
	if raw := control.Port(); raw != "" {
		port, err = strconv.ParseUint(raw, 10, 16)
		if err != nil || port == 0 {
			return
		}
	}
	destination := netip.AddrPortFrom(controlAddress, uint16(port))
	expiry, _ := ctx.Deadline()
	access, err := newPeerProbeAccess(inbound, destination, expiry)
	if err != nil {
		result = "internal_error"
		return
	}
	p.Lock()
	job.access = &access
	p.Unlock()
	select {
	case <-ctx.Done():
		result = "timeout"
		return
	case <-job.ready:
	}
	p.Lock()
	job.result.Stage = "protocol"
	p.Unlock()
	request, stop := context.WithTimeout(ctx, nodeProbeTimeout)
	defer stop()
	stage, result, latency = runPeerHTTPSProbe(request, inbound, host, peerAddress, control, destination, access, roots)
}

func storePeerProbe(db *gorm.DB, key peerProbeKey, digest string, result PeerProbeResult) error {
	leave, ok := database.TryEnterDBOperation()
	if !ok {
		return nil
	}
	defer leave()
	if db != database.GetDB() {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var node model.Node
		err := tx.Where("uuid = ? AND enabled AND enrollment_digest <> '' AND NOT COALESCE(peer_disconnect, 0)", key.nodeUUID).Take(&node).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if digest != nodeSecretDigest(node.LinkHost+"\x00"+string(node.Inventory)) {
			return nil
		}
		row := model.PeerProbeSnapshot{NodeUUID: key.nodeUUID, InboundID: key.inboundID, Stage: result.Stage, Result: result.Result,
			InventoryDigest: digest, CheckedAt: result.CheckedAt, LatencyMS: result.LatencyMS}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	})
}

func collectNodeProbeResults(db *gorm.DB, tracker *peerProbes, node *model.Node) []PeerProbeResult {
	digest := nodeSecretDigest(node.LinkHost + "\x00" + string(node.Inventory))
	results := tracker.results(node.UUID, digest)
	var stored []model.PeerProbeSnapshot
	if db.Where("node_uuid = ? AND inventory_digest = ?", node.UUID, digest).Order("inbound_id").Find(&stored).Error != nil {
		return results
	}
	seen := make(map[uint]bool, len(results))
	for _, current := range results {
		seen[current.InboundID] = current.Result != "running"
	}
	for _, row := range stored {
		if !seen[row.InboundID] {
			results = append(results, PeerProbeResult{InboundID: row.InboundID, Stage: row.Stage, Result: row.Result, CheckedAt: row.CheckedAt, LatencyMS: row.LatencyMS, From: "master"})
		}
	}
	return results
}

func runPeerHTTPSProbe(ctx context.Context, inbound PeerInbound, host string, address netip.Addr, control *url.URL, destination netip.AddrPort, access peerProbeAccess, roots *x509.CertPool) (string, string, int64) {
	credential, err := json.Marshal(map[string]peerUser{inbound.Type: access.user()}) // #nosec G117 -- The transient probe client needs its own short-lived credential.
	if err != nil {
		return "protocol", "internal_error", 0
	}
	options := inbound.Link.subscriptionInbound(inbound)
	links := util.LinkGenerator(credential, &options, host)
	if len(links) != 1 {
		return "protocol", "internal_error", 0
	}
	parsed, tag, err := util.GetOutbound(links[0], 0)
	if err != nil || parsed == nil {
		return "protocol", "internal_error", 0
	}
	(*parsed)["server"] = address.String()
	if inbound.Link.TLS == nil {
		delete(*parsed, "tls")
	} else if tlsOptions, ok := (*parsed)["tls"].(map[string]any); ok && (tlsOptions["server_name"] == nil || tlsOptions["server_name"] == "") {
		tlsOptions["server_name"] = host
	}
	config, err := json.Marshal(map[string]any{"log": map[string]bool{"disabled": true}, "outbounds": []map[string]any{*parsed}})
	if err != nil {
		return "protocol", "internal_error", 0
	}
	clientCore := core.NewCore()
	if err := clientCore.Start(config); err != nil {
		return "protocol", "internal_error", 0
	}
	defer clientCore.Stop() //nolint:errcheck // Disposing the isolated outbound does not change the measured result.
	outbound, ok := clientCore.GetInstance().Outbound().Outbound(tag)
	if !ok {
		return "protocol", "internal_error", 0
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}, MaxResponseHeaderBytes: 32 << 10,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return outbound.DialContext(ctx, network, M.ParseSocksaddr(destination.String()))
		}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, control.String(), nil) // #nosec G704 -- Control URL comes from process config. Only the validated pinned address is dialed.
	if err != nil {
		return "https", "internal_error", 0
	}
	started := time.Now()
	response, err := client.Do(request) // #nosec G704 -- The transport uses the scoped proxy to dial the validated address and forbids redirects.
	if err != nil {
		if ctx.Err() != nil {
			return "protocol", "timeout", 0
		}
		var certificateError *tls.CertificateVerificationError
		if errors.As(err, &certificateError) {
			return "https", "tls_failed", 0
		}
		return "protocol", "connection_failed", 0
	}
	_ = response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "https", "unexpected_response", 0
	}
	return "https", "success", time.Since(started).Milliseconds()
}
