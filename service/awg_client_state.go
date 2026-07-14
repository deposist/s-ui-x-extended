package service

import "context"

// SuspendClients and ResumeClient intentionally reuse full reconciliation:
// eligibility is derived from committed client state, and the single worker
// serializes the live changes with create/rotate/stats.
func (m *AWGManager) SuspendClients(ctx context.Context, _ []uint) error {
	_, err := m.Reconcile(ctx)
	return err
}

func (m *AWGManager) ResumeClient(ctx context.Context, _ uint) error {
	_, err := m.Reconcile(ctx)
	return err
}
