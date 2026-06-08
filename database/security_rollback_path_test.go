package database

import "testing"

func TestSecurityRollbackPathValidationLivesAtAPIBoundary_XFAILPhase4(t *testing.T) {
	t.Skip("XFAIL Phase4: validateRollbackPath is an api-private boundary helper; executable coverage is in api/security_rollback_path_test.go")
}
