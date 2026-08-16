package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// AWGEndpointManager routes device operations to an isolated manager per
// endpoint. This keeps UAPI commands serialized without sharing endpoint state.
type AWGEndpointManager struct {
	runtime   *Runtime
	mu        sync.Mutex
	accepting bool
	managers  map[uint]*AWGManager
}

func NewAWGEndpointManager(runtime *Runtime) *AWGEndpointManager {
	return &AWGEndpointManager{runtime: runtimeOrDefault(runtime), managers: make(map[uint]*AWGManager)}
}

// Start admits device and maintenance work for this application generation.
func (s *AWGEndpointManager) Start() error {
	if s == nil {
		return ErrAWGManagerStopped
	}
	s.mu.Lock()
	s.accepting = true
	s.mu.Unlock()
	return nil
}

func (s *AWGEndpointManager) manager(ctx context.Context, endpointID uint) (*AWGManager, error) {
	if s == nil || endpointID == 0 {
		return nil, ErrAWGEndpointAccessDenied
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if !s.accepting {
		s.mu.Unlock()
		return nil, ErrAWGManagerStopped
	}
	if manager := s.managers[endpointID]; manager != nil {
		s.mu.Unlock()
		return manager, nil
	}
	s.mu.Unlock()

	db := database.GetDB()
	if db == nil {
		return nil, errors.New("AWG database is unavailable")
	}
	endpoint, _, err := LoadAWGEndpointByID(db.WithContext(ctx), endpointID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting {
		return nil, ErrAWGManagerStopped
	}
	if manager := s.managers[endpointID]; manager != nil {
		return manager, nil
	}
	deps := defaultAWGManagerDeps()
	deps.EndpointID = endpointID
	deps.LoadSettings = func() (AWGSettings, error) {
		_, current, loadErr := LoadAWGEndpointByID(database.GetDB(), endpointID)
		return current, loadErr
	}
	manager := NewAWGManagerWithDeps(s.runtime, NewAWGProvisioner(s.runtime, endpoint.Tag), 64, deps)
	if err := manager.Start(); err != nil {
		return nil, err
	}
	s.managers[endpointID] = manager
	return manager, nil
}

func (s *AWGEndpointManager) CreateDevice(ctx context.Context, clientID, endpointID uint, requestKey, name string, expiresAt int64) (AWGDeviceInfo, error) {
	_, _, limit, err := EffectiveAWGEndpointAccess(database.GetDB(), clientID, endpointID)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	manager, err := s.manager(ctx, endpointID)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	return manager.CreateDevice(ctx, clientID, requestKey, name, limit, expiresAt)
}

func (s *AWGEndpointManager) ListDevices(clientID, endpointID uint) ([]AWGDeviceInfo, error) {
	if _, _, _, err := EffectiveAWGEndpointAccess(database.GetDB(), clientID, endpointID); err != nil {
		return nil, err
	}
	manager, err := s.manager(context.Background(), endpointID)
	if err != nil {
		return nil, err
	}
	return manager.ListDevices(clientID)
}

func (s *AWGEndpointManager) GetOwnedDevice(clientID, endpointID, deviceID uint) (AWGDeviceInfo, error) {
	manager, err := s.manager(context.Background(), endpointID)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	return manager.GetOwnedDevice(deviceID, clientID)
}

func (s *AWGEndpointManager) RenderOwnedConfig(ctx context.Context, clientID, endpointID, deviceID uint) ([]byte, error) {
	if _, _, _, err := EffectiveAWGEndpointAccess(database.GetDB(), clientID, endpointID); err != nil {
		return nil, err
	}
	manager, err := s.manager(ctx, endpointID)
	if err != nil {
		return nil, err
	}
	return manager.RenderOwnedConfig(ctx, deviceID, clientID)
}

func (s *AWGEndpointManager) RotateOwnedDevice(ctx context.Context, clientID, endpointID, deviceID uint, requestKey string) (AWGDeviceInfo, error) {
	if _, _, _, err := EffectiveAWGEndpointAccess(database.GetDB(), clientID, endpointID); err != nil {
		return AWGDeviceInfo{}, err
	}
	manager, err := s.manager(ctx, endpointID)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	return manager.RotateOwnedDevice(ctx, deviceID, clientID, requestKey)
}

func (s *AWGEndpointManager) RevokeOwnedDevice(ctx context.Context, clientID, endpointID, deviceID uint) error {
	manager, err := s.manager(ctx, endpointID)
	if err != nil {
		return err
	}
	return manager.RevokeOwnedDevice(ctx, deviceID, clientID)
}

func (s *AWGEndpointManager) ReconcileAll(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	db := database.GetDB()
	if db == nil {
		return errors.New("AWG database is unavailable")
	}
	endpointIDs, err := ListManagedAWGEndpoints(db.WithContext(ctx))
	if err != nil {
		return err
	}
	var result error
	for _, endpointID := range endpointIDs {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		manager, managerErr := s.manager(ctx, endpointID)
		if managerErr != nil {
			result = errors.Join(result, fmt.Errorf("endpoint %d: %w", endpointID, managerErr))
			continue
		}
		_, managerErr = manager.Reconcile(ctx)
		if managerErr != nil {
			result = errors.Join(result, fmt.Errorf("endpoint %d: %w", endpointID, managerErr))
		}
	}
	return result
}

// CollectStatsAll aggregates traffic counters for every managed endpoint.
func (s *AWGEndpointManager) CollectStatsAll(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	db := database.GetDB()
	if db == nil {
		return errors.New("AWG database is unavailable")
	}
	endpointIDs, err := ListManagedAWGEndpoints(db.WithContext(ctx))
	if err != nil {
		return err
	}
	var result error
	for _, endpointID := range endpointIDs {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		manager, managerErr := s.manager(ctx, endpointID)
		if managerErr != nil {
			result = errors.Join(result, managerErr)
			continue
		}
		_, managerErr = manager.CollectStats(ctx)
		result = errors.Join(result, managerErr)
	}
	return result
}

// SuspendClients and ResumeClient intentionally reuse full reconciliation:
// eligibility is derived from committed client state, and each endpoint worker
// serializes the live changes with create/rotate/stats.
func (s *AWGEndpointManager) SuspendClients(ctx context.Context, _ []uint) error {
	return s.ReconcileAll(ctx)
}

func (s *AWGEndpointManager) ResumeClient(ctx context.Context, _ uint) error {
	return s.ReconcileAll(ctx)
}

// StopAll closes admission before canceling every worker. Holding mu while the
// snapshot is stopped prevents Start or a racing lazy manager creation from
// publishing a worker after shutdown has begun.
func (s *AWGEndpointManager) StopAll(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	s.accepting = false
	managers := make([]*AWGManager, 0, len(s.managers))
	for _, manager := range s.managers {
		managers = append(managers, manager)
	}
	s.managers = make(map[uint]*AWGManager)
	var result error
	for _, manager := range managers {
		result = errors.Join(result, manager.Stop(ctx))
	}
	s.mu.Unlock()
	return result
}

func ListManagedAWGEndpoints(db *gorm.DB) ([]uint, error) {
	if db == nil {
		return nil, errors.New("AWG database is unavailable")
	}
	var endpoints []model.Endpoint
	if err := db.Order("id").Find(&endpoints).Error; err != nil {
		return nil, err
	}
	result := make([]uint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		metadata, err := parseAWGEndpointMetadata(endpoint)
		if err != nil {
			return nil, err
		}
		if metadata.Managed {
			result = append(result, endpoint.Id)
		}
	}
	return result, nil
}
