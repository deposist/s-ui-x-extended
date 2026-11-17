package migration

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

func TestMigrateDnsPreservesServerPortAndHTTPSPath(t *testing.T) {
	for _, tc := range []struct {
		address string
		server  string
		port    int
		path    string
	}{
		{"udp://1.1.1.1:5353", "1.1.1.1", 5353, ""},
		{"tls://[2001:db8::1]:8853", "2001:db8::1", 8853, ""},
		{"https://1.1.1.1:8443/custom-dns", "1.1.1.1", 8443, "/custom-dns"},
		{"h3://1.1.1.1:8443/custom-dns", "1.1.1.1", 8443, "/custom-dns"},
	} {
		t.Run(tc.address, func(t *testing.T) {
			db := openMigrationTestDB(t)
			if err := db.AutoMigrate(&model.Setting{}); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(map[string]any{"dns": map[string]any{"servers": []any{map[string]any{"address": tc.address}}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.Setting{Key: "config", Value: string(raw)}).Error; err != nil {
				t.Fatal(err)
			}
			if err := migrate_dns(db); err != nil {
				t.Fatal(err)
			}
			var setting model.Setting
			if err := db.Where("key = ?", "config").First(&setting).Error; err != nil {
				t.Fatal(err)
			}
			var result struct {
				DNS struct {
					Servers []struct {
						Server string `json:"server"`
						Port   int    `json:"server_port"`
						Path   string `json:"path"`
					} `json:"servers"`
				} `json:"dns"`
			}
			if err := json.Unmarshal([]byte(setting.Value), &result); err != nil {
				t.Fatal(err)
			}
			if len(result.DNS.Servers) != 1 {
				t.Fatal("migration lost the DNS server")
			}
			got := result.DNS.Servers[0]
			if got.Server != tc.server || got.Port != tc.port || got.Path != tc.path {
				t.Fatalf("migrated server = %+v; want host=%s port=%d path=%s", got, tc.server, tc.port, tc.path)
			}
		})
	}
}

// TestMigrateDnsNoConfigRowIsNoOp pins the M1 regression: a 1.2-era database that
// has no `config` settings row (managed purely via the entity UIs) must not abort
// the 1.2->1.3 migration. Previously migrate_dns used First() and returned
// gorm.ErrRecordNotFound, rolling back the whole migration/restore.
func TestMigrateDnsNoConfigRowIsNoOp(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	// Only a version row exists; deliberately NO `config` row.
	if err := db.Create(&model.Setting{Key: "version", Value: "1.2"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrate_dns(db); err != nil {
		t.Fatalf("migrate_dns aborted on a config-less DB: %v (want nil)", err)
	}
	if err := migrate_dns(db); err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("migrate_dns still surfaces ErrRecordNotFound on a missing config row")
	}
}

// TestTo13ConfigLessDatabaseSucceeds runs the full 1.2->1.3 step against a
// config-less database to ensure the missing-row no-op holds end-to-end.
func TestTo13ConfigLessDatabaseSucceeds(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.AutoMigrate(&model.Setting{}, &model.Client{}, &model.Outbound{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Setting{Key: "version", Value: "1.2"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := to1_3(db); err != nil {
		t.Fatalf("to1_3 aborted on a config-less DB: %v (want nil)", err)
	}
}
