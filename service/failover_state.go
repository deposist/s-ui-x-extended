package service

import (
	"github.com/deposist/s-ui-x-extended/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnsureFailoverSchema creates the observability-only failover_state table. It
// is non-authoritative (sing-box CacheFile is the stickiness authority); losing
// it never affects failover correctness, only post-restart UI history.
func EnsureFailoverSchema(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS failover_state (
		group_tag TEXT NOT NULL,
		member_tag TEXT NOT NULL,
		healthy INTEGER NOT NULL DEFAULT 0,
		consec_up INTEGER NOT NULL DEFAULT 0,
		consec_down INTEGER NOT NULL DEFAULT 0,
		last_probe_at INTEGER NOT NULL DEFAULT 0,
		last_delay_ms INTEGER NOT NULL DEFAULT 0,
		last_error TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (group_tag, member_tag)
	)`).Error; err != nil {
		return err
	}
	MigrateFailoverStateColumns(db)
	return nil
}

// migrateFailoverStateColumns adds columns introduced after the initial schema.
// ALTER TABLE ADD COLUMN in SQLite cannot use IF NOT EXISTS, so errors for
// already-existing columns are silently ignored. This must be called after
// EnsureFailoverSchema to handle databases created with the old schema.
func MigrateFailoverStateColumns(db *gorm.DB) {
	if db == nil {
		return
	}
	_ = db.Exec(`ALTER TABLE failover_state ADD COLUMN last_delay_ms INTEGER NOT NULL DEFAULT 0`).Error
	_ = db.Exec(`ALTER TABLE failover_state ADD COLUMN last_error TEXT NOT NULL DEFAULT ''`).Error
}

// FailoverMemberState is one row of the failover_state table.
type FailoverMemberState struct {
	GroupTag    string `gorm:"column:group_tag;primaryKey"`
	MemberTag   string `gorm:"column:member_tag;primaryKey"`
	Healthy     bool   `gorm:"column:healthy"`
	ConsecUp    int    `gorm:"column:consec_up"`
	ConsecDown  int    `gorm:"column:consec_down"`
	LastProbeAt int64  `gorm:"column:last_probe_at"`
	LastDelayMs uint16 `gorm:"column:last_delay_ms"`
	LastError   string `gorm:"column:last_error"`
}

func (FailoverMemberState) TableName() string { return "failover_state" }

// WriteFailoverMemberStates upserts the per-member health for a group. A nil db
// (e.g. in unit tests without an initialized database) is a no-op.
func WriteFailoverMemberStates(db *gorm.DB, states []FailoverMemberState) error {
	if db == nil || len(states) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group_tag"}, {Name: "member_tag"}},
		UpdateAll: true,
	}).Create(&states).Error
}

// ReadFailoverMemberStates returns the stored health rows for a group.
func ReadFailoverMemberStates(db *gorm.DB, groupTag string) ([]FailoverMemberState, error) {
	var rows []FailoverMemberState
	if db == nil {
		return rows, nil
	}
	err := db.Where("group_tag = ?", groupTag).Find(&rows).Error
	return rows, err
}

// FailoverMemberStatus is the per-member view returned to the panel.
type FailoverMemberStatus struct {
	Tag      string `json:"tag"`
	Healthy  bool   `json:"healthy"`
	Priority int    `json:"priority"`
	DelayMs  uint16 `json:"delayMs,omitempty"`
	Error    string `json:"error,omitempty"`
}

// FailoverStatusEntry is the per-group status returned to the panel: the live
// active member (from the running selector) plus per-member health.
type FailoverStatusEntry struct {
	Tag           string                 `json:"tag"`
	Active        string                 `json:"active"`
	AllDown       bool                   `json:"allDown"`
	AllDownPolicy string                 `json:"allDownPolicy,omitempty"`
	Members       []FailoverMemberStatus `json:"members"`
}

// FailoverStatus reports every failover group's live active member (authoritative,
// read from the running core via GroupNow) plus per-member health from the
// crash-safe failover_state table (so the answer is meaningful immediately after
// a restart, before the first new probe).
func (s *ConfigService) FailoverStatus() ([]FailoverStatusEntry, error) {
	db := database.GetDB()
	groups, err := LoadFailoverGroups(db)
	if err != nil {
		return nil, err
	}
	coreInst := s.coreInstance()
	out := make([]FailoverStatusEntry, 0, len(groups))
	for _, g := range groups {
		entry := FailoverStatusEntry{Tag: g.Tag, AllDownPolicy: g.AllDownPolicy, Members: []FailoverMemberStatus{}}
		if coreInst != nil {
			if active, ok := coreInst.GroupNow(g.Tag); ok {
				entry.Active = active
			}
		}
		states, _ := ReadFailoverMemberStates(db, g.Tag)
		byTag := make(map[string]FailoverMemberState, len(states))
		for _, st := range states {
			byTag[st.MemberTag] = st
		}
		anyUp := false
		for i, m := range g.Members {
			st := byTag[m]
			entry.Members = append(entry.Members, FailoverMemberStatus{Tag: m, Healthy: st.Healthy, Priority: i, DelayMs: st.LastDelayMs, Error: st.LastError})
			if st.Healthy {
				anyUp = true
			}
		}
		entry.AllDown = !anyUp
		out = append(out, entry)
	}
	return out, nil
}
