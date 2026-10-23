package service

import (
	"encoding/json"
	"path/filepath"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"

	"gorm.io/gorm"
)

// FleetControllerServiceTag is the tag of the manager service the panel adds to
// its own core config. Fixed, because the panel looks the manager up by type
// and a controller runs exactly one.
const FleetControllerServiceTag = "fleet"

// InjectFleetController adds the manager service to the config the controller's
// core starts with.
//
// It stays out of the way until the first server is added: an installation that
// never uses Nodes gets exactly the config it had before, no extra database and
// no extra listener. Once a server exists the manager must be there, otherwise
// no node could ever connect.
//
// The panel owns the fleet tables' lifecycle but not their storage: the manager
// keeps them in its own file next to s-ui.db, so a config edit or a database
// restore never half-migrates the fleet.
//
// A manager configured by hand wins over this one, so an operator who built the
// service themselves keeps full control. A build without -tags with_manager gets
// nothing added, because a service the binary does not contain would stop the
// core from starting at all.
func InjectFleetController(db *gorm.DB, services []json.RawMessage) []json.RawMessage {
	if !capabilities.BuildTags()["with_manager"] {
		return services
	}
	for _, raw := range services {
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}
		if probe.Type == "manager" {
			return services
		}
	}
	if !HasNodes(db) {
		return services
	}
	controller := map[string]any{
		"type": "manager",
		"tag":  FleetControllerServiceTag,
		"database": map[string]any{
			"driver": "sqlite",
			"dsn":    filepath.ToSlash(filepath.Join(config.GetDBFolderPath(), "fleet.db")),
		},
	}
	encoded, err := json.Marshal(controller)
	if err != nil {
		logger.Warning("fleet controller omitted from core config: ", err)
		return services
	}
	return append(services, json.RawMessage(encoded))
}

// HasNodes reports whether any server has been added, i.e. whether the fleet
// has anything to do at all.
func HasNodes(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	var count int64
	if err := db.Model(model.Node{}).Limit(1).Count(&count).Error; err != nil {
		logger.Warning("fleet node count failed: ", err)
		return false
	}
	return count > 0
}
