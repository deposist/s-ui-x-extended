package database

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type beta9Node struct {
	Id               uint   `gorm:"primaryKey;autoIncrement"`
	UUID             string `gorm:"uniqueIndex;not null"`
	Name             string
	EnrollmentDigest string
	LastPeerSync     int64
	PeerRevision     string
	PeerBlocked      int
	PeerDisconnect   bool
	LinkHost         string
	Enabled          bool `gorm:"default:true;not null"`
	Inventory        json.RawMessage
	LimiterRoles     json.RawMessage `gorm:"column:limiters"`
	UpdatedAt        int64
}

func (beta9Node) TableName() string { return "nodes" }

type beta9NodePanel struct {
	AppliedRevision string
	ManagedUsers    json.RawMessage
	ID              uint `gorm:"primaryKey;autoIncrement:false"`
	Role            string
	MasterURL       string
	UUID            string
	Token           string
	LastSync        int64
	SyncError       string
	PublicHost      string
}

func (beta9NodePanel) TableName() string { return "node_panels" }

func TestInitDBPreservesBeta9NodeRowsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dir)
	path := filepath.Join(dir, "s-ui.db")
	seed, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	pool, err := seed.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.AutoMigrate(&beta9Node{}, &beta9NodePanel{}); err != nil {
		_ = pool.Close()
		t.Fatal(err)
	}
	if err := seed.Create(&beta9Node{UUID: "peer", Name: "Legacy", LastPeerSync: 123, PeerRevision: "legacy-revision", Inventory: json.RawMessage(`{}`)}).Error; err != nil {
		_ = pool.Close()
		t.Fatal(err)
	}
	if err := seed.Create(&beta9NodePanel{ID: 1, Role: "slave", UUID: "peer", LastSync: 123, AppliedRevision: "legacy-revision", ManagedUsers: json.RawMessage(`[]`)}).Error; err != nil {
		_ = pool.Close()
		t.Fatal(err)
	}
	var seedInventory string
	if err := seed.Raw("SELECT hex(inventory) FROM nodes WHERE uuid = 'peer'").Scan(&seedInventory).Error; err != nil || seedInventory != "7B7D" {
		t.Fatalf("beta9 fixture inventory = %q, error: %v", seedInventory, err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := InitDB(path); err != nil {
			t.Fatal(err)
		}
		func() {
			defer closeMainDB(t)
			db := GetDB()
			var peer model.Node
			var panel model.NodePanel
			if err := db.Where("uuid = ?", "peer").Take(&peer).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Take(&panel, 1).Error; err != nil {
				t.Fatal(err)
			}
			if peer.Name != "Legacy" || peer.LastPeerSync != 123 || peer.PeerRevision != "legacy-revision" ||
				string(peer.Inventory) != "{}" || panel.LastSync != 123 || string(panel.ManagedUsers) != "[]" {
				var raw string
				readErr := db.Raw("SELECT hex(inventory) FROM nodes WHERE uuid = 'peer'").Scan(&raw).Error
				t.Fatalf("legacy node data changed during initialization: name=%q nodeSync=%d revision=%q inventory=%q raw=%q (%v) panelSync=%d users=%q",
					peer.Name, peer.LastPeerSync, peer.PeerRevision, peer.Inventory, raw, readErr, panel.LastSync, panel.ManagedUsers)
			}
			if !db.Migrator().HasTable(&model.PeerTrafficSnapshot{}) || !db.Migrator().HasTable(&model.PeerProbeSnapshot{}) {
				t.Fatal("node diagnostics tables are missing")
			}
			if attempt == 0 {
				if err := db.Model(&peer).UpdateColumn("peer_access_until", 456).Error; err != nil {
					t.Fatal(err)
				}
			} else if peer.PeerAccessUntil != 456 {
				t.Fatal("new access metadata was lost after restart")
			}
		}()
	}
}
