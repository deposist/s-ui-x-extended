package service

import (
	"context"
	"errors"
	"net/netip"

	"github.com/deposist/s-ui-x-extended/core"
)

var (
	errAWGIPCUnavailable     = errors.New("AWG endpoint is unavailable")
	errAWGIPCRead            = errors.New("AWG peer snapshot read failed")
	errAWGIPCWrite           = errors.New("AWG peer update failed")
	errAWGSnapshotInvalid    = errors.New("AWG peer snapshot is invalid")
	errAWGAllowedIPInvalid   = errors.New("AWG peer allowed IP must be an IPv4 /32")
	errAWGAddVerification    = errors.New("AWG peer add verification failed")
	errAWGRemoveVerification = errors.New("AWG peer remove verification failed")
)

// AWGPeerState is the non-secret live state for one managed peer.
type AWGPeerState struct {
	AllowedIPs    []netip.Prefix
	LastHandshake int64
	ReceiveBytes  uint64
	TransmitBytes uint64
}

// AWGPeerSnapshot is keyed by the canonical base64 public key used in domain
// and database state.
type AWGPeerSnapshot map[string]AWGPeerState

// AWGPeerSpec contains the domain representation of a managed peer.
type AWGPeerSpec struct {
	PublicKey    string
	PresharedKey string
	AllowedIP    netip.Prefix
}

// AWGProvisioner applies and observes the managed endpoint's live peer state.
type AWGProvisioner interface {
	Snapshot(ctx context.Context) (AWGPeerSnapshot, error)
	Add(ctx context.Context, peer AWGPeerSpec) error
	Remove(ctx context.Context, publicKeyBase64 string) error
}

type withAWGIPCFunc func(ctx context.Context, tag string, fn func(core.WireGuardIPC) error) error

type awgProvisioner struct {
	endpointTag string
	runtime     *Runtime
	withIPC     withAWGIPCFunc
}

// NewAWGProvisioner creates the production provisioner. The core facade is the
// only path by which service code obtains a live WireGuard IPC handle.
func NewAWGProvisioner(runtime *Runtime, endpointTag string) AWGProvisioner {
	return &awgProvisioner{endpointTag: endpointTag, runtime: runtime}
}

func newAWGProvisioner(endpointTag string, withIPC withAWGIPCFunc) AWGProvisioner {
	return &awgProvisioner{endpointTag: endpointTag, withIPC: withIPC}
}

func (p *awgProvisioner) Snapshot(ctx context.Context) (AWGPeerSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var snapshot AWGPeerSnapshot
	if err := p.runWithIPC(ctx, func(ipc core.WireGuardIPC) error {
		parsed, err := readAWGPeerSnapshot(ipc)
		if err != nil {
			return err
		}
		snapshot = parsed
		return nil
	}); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (p *awgProvisioner) Add(ctx context.Context, peer AWGPeerSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !peer.AllowedIP.IsValid() || !peer.AllowedIP.Addr().Is4() || peer.AllowedIP.Bits() != 32 {
		return errAWGAllowedIPInvalid
	}
	payload, err := core.BuildWireGuardAddPeerUAPI(peer.PublicKey, peer.PresharedKey, peer.AllowedIP)
	if err != nil {
		return err
	}
	allowedIP := peer.AllowedIP.Masked()
	return p.runWithIPC(ctx, func(ipc core.WireGuardIPC) error {
		if err := ipc.IpcSet(payload); err != nil {
			return errAWGIPCWrite
		}
		snapshot, err := readAWGPeerSnapshot(ipc)
		if err != nil {
			return err
		}
		state, exists := snapshot[peer.PublicKey]
		if !exists || len(state.AllowedIPs) != 1 || state.AllowedIPs[0] != allowedIP {
			return errAWGAddVerification
		}
		return nil
	})
}

func (p *awgProvisioner) Remove(ctx context.Context, publicKeyBase64 string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	payload, err := core.BuildWireGuardRemovePeerUAPI(publicKeyBase64)
	if err != nil {
		return err
	}
	return p.runWithIPC(ctx, func(ipc core.WireGuardIPC) error {
		if err := ipc.IpcSet(payload); err != nil {
			return errAWGIPCWrite
		}
		snapshot, err := readAWGPeerSnapshot(ipc)
		if err != nil {
			return err
		}
		if _, exists := snapshot[publicKeyBase64]; exists {
			return errAWGRemoveVerification
		}
		return nil
	})
}

func (p *awgProvisioner) runWithIPC(ctx context.Context, fn func(core.WireGuardIPC) error) error {
	if p == nil {
		return errAWGIPCUnavailable
	}
	endpointTag := p.endpointTag
	// The legacy global manager is constructed before the admin enables AWG,
	// so an empty tag must resolve against the live awgEndpointTag setting.
	// A non-empty tag (endpoint-scoped managers pass the managed endpoint's
	// own tag) is authoritative: overriding it with the legacy setting would
	// send every scoped reconcile to the wrong endpoint, or to none when the
	// legacy scheme is disabled.
	if endpointTag == "" && p.runtime != nil {
		if settings, err := (&SettingService{}).GetAWGSettings(); err == nil {
			endpointTag = settings.EndpointTag
		}
	}
	if endpointTag == "" {
		return errAWGIPCUnavailable
	}
	withIPC := p.withIPC
	if withIPC == nil && p.runtime != nil {
		coreInstance := p.runtime.Core()
		if coreInstance != nil {
			withIPC = coreInstance.WithWireGuardIPCContext
		}
	}
	if withIPC == nil {
		return errAWGIPCUnavailable
	}
	err := withIPC(ctx, endpointTag, fn)
	if err == nil || isAWGSafeError(err) {
		return err
	}
	return errAWGIPCUnavailable
}

func readAWGPeerSnapshot(ipc core.WireGuardIPC) (AWGPeerSnapshot, error) {
	raw, err := ipc.IpcGet()
	if err != nil {
		return nil, errAWGIPCRead
	}
	peers, err := core.ParseWireGuardPeerSnapshot(raw)
	if err != nil {
		return nil, errAWGSnapshotInvalid
	}
	snapshot := make(AWGPeerSnapshot, len(peers))
	for _, peer := range peers {
		if _, exists := snapshot[peer.PublicKey]; exists {
			return nil, errAWGSnapshotInvalid
		}
		snapshot[peer.PublicKey] = AWGPeerState{
			AllowedIPs:    append([]netip.Prefix(nil), peer.AllowedIPs...),
			LastHandshake: peer.LastHandshake,
			ReceiveBytes:  peer.ReceiveBytes,
			TransmitBytes: peer.TransmitBytes,
		}
	}
	return snapshot, nil
}

func isAWGSafeError(err error) bool {
	return err == context.Canceled ||
		err == context.DeadlineExceeded ||
		err == errAWGIPCRead ||
		err == errAWGIPCWrite ||
		err == errAWGSnapshotInvalid ||
		err == errAWGAddVerification ||
		err == errAWGRemoveVerification
}
