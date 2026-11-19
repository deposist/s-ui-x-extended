package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

const PeerLeaseHeader = "X-Node-Lease-V1"

type peerLease struct {
	Until int64  `json:"until"`
	Proof string `json:"proof"`
}

func (l peerLease) Header() string {
	return strconv.FormatInt(l.Until, 10) + "." + l.Proof
}

func signPeerLease(key []byte, node *model.Node, revision string, until int64) peerLease {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("peer-access-v1:" + node.UUID + ":" + revision + ":" + strconv.FormatInt(until, 10)))
	return peerLease{Until: until, Proof: hex.EncodeToString(mac.Sum(nil))}
}

func confirmedPeerLease(key []byte, node *model.Node, revision, receipt string, now time.Time) int64 {
	stamp, _, ok := strings.Cut(receipt, ".")
	if !ok || len(receipt) > 96 || revision == "" {
		return 0
	}
	until, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || until <= now.Unix() || until > now.Add(model.PeerAccessLease).Unix() {
		return 0
	}
	if !hmac.Equal([]byte(receipt), []byte(signPeerLease(key, node, revision, until).Header())) {
		return 0
	}
	return until
}

func (s *NodeService) peerTime() time.Time {
	if s.peerNow != nil {
		return s.peerNow()
	}
	return time.Now()
}

func nodePeerAccessActive(node *model.Node, now time.Time) bool {
	if node.PeerRevision == "" {
		return false
	}
	if node.PeerLeaseCapable {
		return now.Unix() < node.PeerAccessUntil
	}
	return node.LastPeerSync > now.Add(-model.PeerAccessLease).Unix()
}
