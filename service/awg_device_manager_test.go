package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/netip"
	"sync"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

type recordingAWGCipher struct {
	plaintext [][]byte
}

func (c *recordingAWGCipher) Encrypt(_ uint, _ []byte, plaintext []byte) ([]byte, error) {
	c.plaintext = append(c.plaintext, append([]byte(nil), plaintext...))
	return append([]byte("sealed:"), plaintext...), nil
}

func (c *recordingAWGCipher) Decrypt(_ uint, _ []byte, ciphertext []byte) ([]byte, error) {
	if !bytes.HasPrefix(ciphertext, []byte("sealed:")) {
		return nil, errors.New("invalid ciphertext")
	}
	return append([]byte(nil), ciphertext[7:]...), nil
}

func newAWGCreateTestManager(t *testing.T, provisioner AWGProvisioner) (*AWGManager, *recordingAWGCipher) {
	t.Helper()
	initSettingTestDB(t)
	db := database.GetDB()
	if err := db.AutoMigrate(&model.AWGDevice{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_client_create_request
		ON awg_devices(client_id, create_request_key) WHERE create_request_key != ''`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_awg_devices_active_ipv4
		ON awg_devices(ipv4_address) WHERE desired_enabled = 1`).Error; err != nil {
		t.Fatal(err)
	}
	cipher := &recordingAWGCipher{}
	manager := NewAWGManagerWithDeps(nil, provisioner, 8, AWGManagerDeps{
		DB: db,
		LoadSettings: func() (AWGSettings, error) {
			return AWGSettings{Enabled: true, EndpointTag: "awg", Subnet: netip.MustParsePrefix("10.77.0.0/29")}, nil
		},
		LoadEndpoint: func(*gorm.DB, AWGSettings) (AWGManagedEndpoint, error) {
			return AWGManagedEndpoint{ServerAddress: netip.MustParseAddr("10.77.0.1")}, nil
		},
		GenerateKeys: func() (AWGGeneratedKeys, error) {
			return AWGGeneratedKeys{
				PrivateKey: bytes.Repeat([]byte{0x11}, 32),
				PublicKey:  bytes.Repeat([]byte{0x22}, 32),
				PSK:        bytes.Repeat([]byte{0x33}, 32),
			}, nil
		},
		NewCryptoContext: func() ([]byte, error) { return bytes.Repeat([]byte{0x44}, awgCryptoContextSize), nil },
		Cipher:           cipher,
		Now:              func() int64 { return 1_700_000_000 },
	})
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })
	return manager, cipher
}

func createAWGEligibleClient(t *testing.T) model.Client {
	t.Helper()
	client := model.Client{Enable: true, Name: "client", Expiry: 1_700_000_001}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	return client
}

func TestAWGCreateDevicePersistsThenProvisionsAndFinalizes(t *testing.T) {
	var adds []AWGPeerSpec
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(_ context.Context, peer AWGPeerSpec) error { adds = append(adds, peer); return nil },
	}
	manager, cipher := newAWGCreateTestManager(t, provisioner)
	client := createAWGEligibleClient(t)

	got, err := manager.CreateDevice(context.Background(), client.Id, " update-42 ", "  My   Phone  ", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "My Phone" || got.IPv4Address != "10.77.0.2" || !got.Provisioned || got.SyncState != "in_sync" {
		t.Fatalf("unexpected device info: %#v", got)
	}
	if got.PublicKey != base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x22}, 32)) {
		t.Fatalf("unexpected public key: %q", got.PublicKey)
	}
	if len(cipher.plaintext) != 2 || !bytes.Equal(cipher.plaintext[0], bytes.Repeat([]byte{0x11}, 32)) ||
		!bytes.Equal(cipher.plaintext[1], bytes.Repeat([]byte{0x33}, 32)) {
		t.Fatalf("cipher did not receive raw private key and PSK: %#v", cipher.plaintext)
	}
	if len(adds) != 1 || adds[0].PublicKey != got.PublicKey ||
		adds[0].PresharedKey != base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, 32)) {
		t.Fatalf("unexpected provisioned peer: %#v", adds)
	}

	var stored model.AWGDevice
	if err := database.GetDB().First(&stored, got.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.CreateRequestKey != "update-42" || string(stored.PrivateKeyEnc[:7]) != "sealed:" ||
		stored.SyncState != "in_sync" || !stored.Provisioned || stored.LastError != "" {
		t.Fatalf("unexpected durable device: %#v", stored)
	}
}

func TestAWGCreateDeviceReplayPrecedesEligibilityAndLimit(t *testing.T) {
	var adds []AWGPeerSpec
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(_ context.Context, peer AWGPeerSpec) error { adds = append(adds, peer); return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	client := createAWGEligibleClient(t)
	first, err := manager.CreateDevice(context.Background(), client.Id, "request-1", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.Client{}).Where("id = ?", client.Id).Updates(map[string]any{"enable": false, "expiry": int64(1)}).Error; err != nil {
		t.Fatal(err)
	}
	manager.deps.LoadSettings = func() (AWGSettings, error) { return AWGSettings{}, errors.New("settings unavailable") }
	replayed, err := manager.CreateDevice(context.Background(), client.Id, " request-1 ", "Phone", 0, 0)
	if err != nil {
		t.Fatalf("durable replay must succeed before eligibility and limit checks: %v", err)
	}
	if replayed.ID != first.ID || len(adds) != 1 {
		t.Fatalf("replay created or provisioned a second device: first=%#v replay=%#v adds=%d", first, replayed, len(adds))
	}
}

func TestAWGCreateDeviceReplayRejectsDifferentName(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
	})
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "request-name", "Phone", 1, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CreateDevice(context.Background(), client.Id, "request-name", "Laptop", 1, 0); !errors.Is(err, ErrAWGIdempotencyConflict) {
		t.Fatalf("different-name replay error = %v, want ErrAWGIdempotencyConflict", err)
	}
}

func TestAWGCreateDevicePendingReplayReusesKeyMaterial(t *testing.T) {
	var addCalls int
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add: func(context.Context, AWGPeerSpec) error {
			addCalls++
			if addCalls == 1 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	var generateCalls int
	originalGenerate := manager.deps.GenerateKeys
	manager.deps.GenerateKeys = func() (AWGGeneratedKeys, error) {
		generateCalls++
		return originalGenerate()
	}
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "request-retry", "Phone", 1, 0); !errors.Is(err, ErrAWGProvisioningFailed) {
		t.Fatalf("first create error = %v, want ErrAWGProvisioningFailed", err)
	}
	got, err := manager.CreateDevice(context.Background(), client.Id, "request-retry", "Phone", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Provisioned || generateCalls != 1 || addCalls != 2 {
		t.Fatalf("replay result=%#v generateCalls=%d addCalls=%d", got, generateCalls, addCalls)
	}
}

func TestAWGCreateDevicePendingReplayRechecksEligibility(t *testing.T) {
	var addCalls int
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add: func(context.Context, AWGPeerSpec) error {
			addCalls++
			return errors.New("temporary failure")
		},
	})
	client := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), client.Id, "pending-inactive", "Phone", 1, 0); !errors.Is(err, ErrAWGProvisioningFailed) {
		t.Fatalf("first create error = %v", err)
	}
	if err := database.GetDB().Model(&model.Client{}).Where("id = ?", client.Id).Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CreateDevice(context.Background(), client.Id, "pending-inactive", "Phone", 1, 0); !errors.Is(err, ErrAWGClientInactive) {
		t.Fatalf("inactive pending replay error = %v", err)
	}
	if addCalls != 1 {
		t.Fatalf("inactive replay invoked Add: calls=%d", addCalls)
	}
}

func TestAWGDeviceNameAndRequestKeyValidation(t *testing.T) {
	if got, err := normalizeAWGDeviceName("  Cafe\u0301   Phone  "); err != nil || got != "Café Phone" {
		t.Fatalf("normalized name = %q, %v", got, err)
	}
	for _, invalid := range []string{"bad\x00name", "bad\u202ename", string([]byte{0xff})} {
		if _, err := normalizeAWGDeviceName(invalid); !errors.Is(err, ErrAWGInvalidDeviceName) {
			t.Fatalf("invalid name %q error = %v", invalid, err)
		}
	}
	for _, valid := range []string{"tg:123:456", "update-42_retry.1"} {
		if got, err := normalizeAWGRequestKey(valid); err != nil || got != valid {
			t.Fatalf("request key %q = %q, %v", valid, got, err)
		}
	}
	for _, invalid := range []string{"space key", "ключ", "bad\nkey"} {
		if _, err := normalizeAWGRequestKey(invalid); !errors.Is(err, ErrAWGInvalidRequestKey) {
			t.Fatalf("invalid request key %q error = %v", invalid, err)
		}
	}
}

func TestAWGCreateDeviceConcurrentSameRequestConverges(t *testing.T) {
	var mu sync.Mutex
	var addCalls, generateCalls int
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add: func(context.Context, AWGPeerSpec) error {
			mu.Lock()
			addCalls++
			mu.Unlock()
			return nil
		},
	})
	originalGenerate := manager.deps.GenerateKeys
	manager.deps.GenerateKeys = func() (AWGGeneratedKeys, error) {
		mu.Lock()
		generateCalls++
		mu.Unlock()
		return originalGenerate()
	}
	client := createAWGEligibleClient(t)
	start := make(chan struct{})
	results := make(chan AWGDeviceInfo, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			info, err := manager.CreateDevice(context.Background(), client.Id, "same-request", "Phone", 2, 0)
			results <- info
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var id uint
	for info := range results {
		if id == 0 {
			id = info.ID
		} else if info.ID != id {
			t.Fatalf("same request returned device %d and %d", id, info.ID)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if generateCalls != 1 || addCalls != 1 {
		t.Fatalf("generateCalls=%d addCalls=%d, want 1/1", generateCalls, addCalls)
	}
}

func TestAWGCreateDeviceProvisionFailureRetainsSanitizedPendingState(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, 32))
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return errors.New("uapi failed: " + secret) },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	client := createAWGEligibleClient(t)

	got, err := manager.CreateDevice(context.Background(), client.Id, "request-fail", "Phone", 1, 0)
	if err == nil {
		t.Fatal("expected provisioning failure")
	}
	if got.ID != 0 {
		t.Fatalf("failed create returned device data: %#v", got)
	}
	var stored model.AWGDevice
	if dbErr := database.GetDB().Where("client_id = ? AND create_request_key = ?", client.Id, "request-fail").First(&stored).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if stored.SyncState != "pending_add" || stored.Provisioned || stored.LastError != "AWG provisioning failed" {
		t.Fatalf("unexpected failure state: %#v", stored)
	}
	if bytes.Contains([]byte(stored.LastError), []byte(secret)) {
		t.Fatal("secret leaked into durable error")
	}
}

func TestAWGCreateDeviceRejectsInactiveAndLimitWithoutDurableRow(t *testing.T) {
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
	})
	inactive := model.Client{Enable: false, Name: "inactive"}
	if err := database.GetDB().Create(&inactive).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CreateDevice(context.Background(), inactive.Id, "inactive-request", "Phone", 1, 0); !errors.Is(err, ErrAWGClientInactive) {
		t.Fatalf("inactive error = %v", err)
	}
	active := createAWGEligibleClient(t)
	if _, err := manager.CreateDevice(context.Background(), active.Id, "limit-request", "Phone", 0, 0); !errors.Is(err, ErrAWGDeviceLimitReached) {
		t.Fatalf("limit error = %v", err)
	}
	var count int64
	if err := database.GetDB().Model(&model.AWGDevice{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rejected creates persisted %d rows", count)
	}
}

func TestAWGCreateDeviceConcurrentRequestsRespectLimit(t *testing.T) {
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
	}
	manager, _ := newAWGCreateTestManager(t, provisioner)
	client := createAWGEligibleClient(t)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, requestKey := range []string{"concurrent-1", "concurrent-2"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			<-start
			_, err := manager.CreateDevice(context.Background(), client.Id, key, "Phone", 1, 0)
			errs <- err
		}(requestKey)
	}
	close(start)
	wg.Wait()
	close(errs)

	var successes, limited int
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrAWGDeviceLimitReached):
			limited++
		default:
			t.Fatalf("unexpected concurrent create error: %v", err)
		}
	}
	if successes != 1 || limited != 1 {
		t.Fatalf("successes=%d limited=%d, want 1/1", successes, limited)
	}
}
