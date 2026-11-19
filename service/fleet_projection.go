package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	"gorm.io/gorm"
)

const FleetSquadPrefix = "node-"

// projectionMu serialises whole-fleet reconciliation. Two saves at once would
// otherwise interleave manager reads and writes with divergent snapshots.
var projectionMu sync.Mutex

type fleetNode struct {
	row     model.Node
	squadID int
	tags    map[string]string
	roles   model.NodeLimiterTags
}

func (s *NodeService) ProjectFleet() error {
	projectionMu.Lock()
	defer projectionMu.Unlock()
	state, err := s.loadFleetState()
	if err != nil || state == nil {
		return err
	}
	return s.withManager(func(manager CM.Manager) error { return state.apply(manager) })
}

func (s *NodeService) ProjectClients(_ []uint) error { return s.ProjectFleet() }

func (s *NodeService) projectFleetOrWarn(action string) {
	if err := s.ProjectFleet(); err != nil {
		logger.Warning("fleet projection after ", action, " failed: ", err)
	}
}

func (s *NodeService) withManager(fn func(CM.Manager) error) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil {
		return nil
	}
	err := coreInstance.WithFleetManager(fn)
	if errors.Is(err, core.ErrFleetUnavailable) {
		return nil
	}
	return err
}

type fleetState struct {
	db            *gorm.DB
	nodes         []fleetNode
	clients       []model.Client
	assign        map[uint][]string
	limits        map[uint]model.ClientLimits
	managedSquads map[int]struct{}
	ownership     map[string]int
}

func (s *NodeService) loadFleetState() (*fleetState, error) {
	db := s.db()
	if db == nil {
		return nil, nil
	}
	state := &fleetState{db: db, assign: map[uint][]string{}, limits: map[uint]model.ClientLimits{}, ownership: map[string]int{}}
	var ownership []model.FleetNodeOwnership
	if err := db.Find(&ownership).Error; err != nil {
		return nil, err
	}
	for _, record := range ownership {
		state.ownership[record.NodeUUID] = record.SquadID
	}
	var nodes []model.Node
	if err := db.Model(model.Node{}).Where("COALESCE(enrollment_digest, '') = ''").Find(&nodes).Error; err != nil {
		return nil, err
	}
	for _, node := range nodes {
		announced := DecodeNodeInventory(node.Inventory)
		tags := make(map[string]string, len(announced.Inbounds))
		for _, inbound := range announced.Inbounds {
			if inbound.Tag != "" {
				tags[inbound.Tag] = inbound.Type
			}
		}
		var roles model.NodeLimiterTags
		if len(node.LimiterRoles) > 0 {
			if err := json.Unmarshal(node.LimiterRoles, &roles); err != nil {
				logger.Warning("node ", node.Name, " limiter roles unreadable, ignored: ", err)
			}
		}
		state.nodes = append(state.nodes, fleetNode{row: node, tags: tags, roles: roles})
	}
	if err := db.Model(model.Client{}).Where("enable = ?", true).Find(&state.clients).Error; err != nil {
		return nil, err
	}
	var assignments []model.ClientNode
	if err := db.Model(model.ClientNode{}).Find(&assignments).Error; err != nil {
		return nil, err
	}
	for _, a := range assignments {
		state.assign[a.ClientId] = append(state.assign[a.ClientId], a.NodeUUID)
	}
	var limits []model.ClientLimits
	if err := db.Model(model.ClientLimits{}).Find(&limits).Error; err != nil {
		return nil, err
	}
	for _, limit := range limits {
		state.limits[limit.ClientId] = limit
	}
	return state, nil
}

func (s *fleetState) apply(manager CM.Manager) error {
	if err := s.ensureSquads(manager); err != nil {
		return err
	}
	users := map[string]CM.UserCreate{}
	bandwidth := map[string]CM.BandwidthLimiterCreate{}
	connections := map[string]CM.ConnectionLimiterCreate{}
	for _, client := range s.clients {
		if err := s.projectClient(client, users, bandwidth, connections); err != nil {
			return err
		}
	}
	if err := syncFleetUsers(manager, s.managedSquads, users); err != nil {
		return err
	}
	if err := syncFleetLimiters(manager, s.managedSquads, bandwidth, connections); err != nil {
		return err
	}
	return s.cleanupDeletedNodes(manager)
}

func (s *fleetState) cleanupDeletedNodes(manager CM.Manager) error {
	active := make(map[string]struct{}, len(s.nodes))
	for _, node := range s.nodes {
		active[node.row.UUID] = struct{}{}
	}
	squads, err := manager.GetSquads(nil)
	if err != nil {
		return err
	}
	existing := make(map[int]struct{}, len(squads))
	for _, squad := range squads {
		existing[squad.ID] = struct{}{}
	}
	nodes, err := manager.GetNodes(nil)
	if err != nil {
		return err
	}
	for uuid, id := range s.ownership {
		if _, present := active[uuid]; present {
			continue
		}
		for _, node := range nodes {
			if slices.Contains(node.SquadIDs, id) {
				return fmt.Errorf("manager squad %d still belongs to node %q; refusing to delete it", id, node.UUID)
			}
		}
		if _, present := existing[id]; present {
			if _, err := manager.DeleteSquad(id); err != nil {
				return err
			}
		}
		if err := s.db.Where("node_uuid = ?", uuid).Delete(&model.FleetNodeOwnership{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *fleetState) ensureSquads(manager CM.Manager) error {
	squads, err := manager.GetSquads(nil)
	if err != nil {
		return err
	}
	byName, byID := make(map[string]CM.Squad, len(squads)), make(map[int]CM.Squad, len(squads))
	s.managedSquads = make(map[int]struct{}, len(s.ownership))
	for _, squad := range squads {
		byName[squad.Name], byID[squad.ID] = squad, squad
	}
	for _, id := range s.ownership {
		s.managedSquads[id] = struct{}{}
	}
	managerNodes, err := manager.GetNodes(nil)
	if err != nil {
		return err
	}
	known := make(map[string]CM.Node, len(managerNodes))
	local := make(map[string]struct{}, len(s.nodes))
	for _, node := range managerNodes {
		known[node.UUID] = node
	}
	for _, node := range s.nodes {
		local[node.row.UUID] = struct{}{}
	}
	for _, node := range s.nodes {
		if _, exists := known[node.row.UUID]; exists {
			if _, owned := s.ownership[node.row.UUID]; !owned {
				return fmt.Errorf("manager node %q already exists and is not panel-owned; refusing to adopt it", node.row.UUID)
			}
		} else if _, owned := s.ownership[node.row.UUID]; !owned && node.row.Enabled {
			if _, collision := byName[FleetSquadPrefix+node.row.UUID]; collision {
				return fmt.Errorf("manager squad %q already exists and is not panel-owned; refusing to adopt it", FleetSquadPrefix+node.row.UUID)
			}
		}
	}
	for _, node := range managerNodes {
		if _, present := local[node.UUID]; present {
			continue
		}
		if squadID, owned := s.ownership[node.UUID]; owned {
			if !slices.Contains(node.SquadIDs, squadID) {
				return fmt.Errorf("manager node %q no longer belongs to its recorded squad; refusing to delete it", node.UUID)
			}
			if len(node.SquadIDs) != 1 {
				return fmt.Errorf("manager node %q has unrelated squad membership; refusing to delete it", node.UUID)
			}
			if _, err := manager.DeleteNode(node.UUID); err != nil {
				return err
			}
			delete(known, node.UUID)
		}
	}
	for i := range s.nodes {
		node := &s.nodes[i]
		if !node.row.Enabled {
			if squadID, owned := s.ownership[node.row.UUID]; owned {
				if current, ok := known[node.row.UUID]; ok {
					if !sameSquadSet(current.SquadIDs, []int{squadID}) {
						return fmt.Errorf("manager node %q has unrelated squad membership; refusing to delete it", node.row.UUID)
					}
					if _, err := manager.DeleteNode(node.row.UUID); err != nil {
						return err
					}
					delete(known, node.row.UUID)
				}
			}
			continue
		}
		name := FleetSquadPrefix + node.row.UUID
		squadID, owned := s.ownership[node.row.UUID]
		squad, exists := byID[squadID]
		if owned && !exists {
			if _, ok := byName[name]; ok {
				return fmt.Errorf("manager squad %q conflicts with recorded ownership; refusing to adopt it", name)
			}
			squad, err = manager.CreateSquad(CM.SquadCreate{Name: name})
			if err != nil {
				return err
			}
			byName[name], byID[squad.ID] = squad, squad
		} else if !owned {
			squad, err = manager.CreateSquad(CM.SquadCreate{Name: name})
			if err != nil {
				return err
			}
			byName[name], byID[squad.ID] = squad, squad
		}
		node.squadID = squad.ID
		s.managedSquads[squad.ID] = struct{}{}
		current, exists := known[node.row.UUID]
		if exists && !sameSquadSet(current.SquadIDs, []int{squad.ID}) {
			for _, id := range current.SquadIDs {
				if _, managed := s.managedSquads[id]; !managed {
					return fmt.Errorf("manager node %q has unrelated squad membership; refusing to replace it", node.row.UUID)
				}
			}
			if _, err := manager.DeleteNode(node.row.UUID); err != nil {
				return err
			}
			exists = false
		}
		if exists {
			if current.Name != node.row.Name {
				if _, err := manager.UpdateNode(node.row.UUID, CM.NodeUpdate{Name: node.row.Name}); err != nil {
					return err
				}
			}
			continue
		}
		if _, err := manager.CreateNode(CM.NodeCreate{UUID: node.row.UUID, Name: node.row.Name, SquadIDs: []int{squad.ID}}); err != nil {
			return err
		}
		if !owned {
			if err := s.db.Create(&model.FleetNodeOwnership{NodeUUID: node.row.UUID, SquadID: squad.ID}).Error; err != nil {
				return err
			}
			s.ownership[node.row.UUID] = squad.ID
		} else if squadID != squad.ID {
			if err := s.db.Model(&model.FleetNodeOwnership{}).Where("node_uuid = ?", node.row.UUID).Update("squad_id", squad.ID).Error; err != nil {
				return err
			}
			s.ownership[node.row.UUID] = squad.ID
		}
	}
	return nil
}

func sameSquadSet(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]int(nil), a...)
	y := append([]int(nil), b...)
	sort.Ints(x)
	sort.Ints(y)
	return slices.Equal(x, y)
}
