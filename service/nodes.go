package service

import (
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/gofrs/uuid/v5"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	"gorm.io/gorm"
)

// NodeService owns the controller's record of the servers it manages.
//
// The row holds only what the controller knows: identity, the address clients
// dial and the last inventory a node announced. The fleet tables themselves
// (squads, fleet users, limiters) live in the manager's own database and are
// reached through core.WithFleetManager; duplicating them here would be a
// second source of truth.
type NodeService struct {
	Runtime *Runtime
}

// NodeView is a node row plus what only the running core knows.
type NodeView struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	LinkHost string `json:"linkHost"`
	Enabled  bool   `json:"enabled"`

	// Online reports whether the node is connected right now. An unreachable
	// controller (no manager service running) reports every node offline
	// instead of failing, because the list must still render.
	Online bool `json:"online"`

	// Inbounds and Limiters come from the last announced inventory, so the UI
	// can show what a node can actually accept before assigning anything.
	Inbounds []string `json:"inbounds"`
	Limiters []string `json:"limiters"`

	// ClientCount is how many clients are currently assigned to this node.
	ClientCount int64 `json:"clientCount"`

	UpdatedAt int64 `json:"updatedAt"`
}

// ClientView is one row of the per-server client list. It carries only what the
// node page renders so a page of 100 stays small.
type ClientView struct {
	Id     uint   `json:"id"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	Up     int64  `json:"up"`
	Down   int64  `json:"down"`
	Volume int64  `json:"volume"`
	Expiry int64  `json:"expiry"`
	Group  string `json:"group"`
}

// ClientPage is one page of the per-server client list. Total is the count
// under the current filter, not just this page, so the pager can show pages
// the client has not fetched yet.
type ClientPage struct {
	Items []ClientView `json:"items"`
	Total int64        `json:"total"`
}

const (
	defaultClientPageSize = 25
	maxClientPageSize     = 200
)

func (s *NodeService) db() *gorm.DB {
	return database.GetDB()
}

func (s *NodeService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

// List returns every node with its live status and assigned client count.
func (s *NodeService) List() ([]NodeView, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	var nodes []model.Node
	if err := db.Model(model.Node{}).Order("name").Find(&nodes).Error; err != nil {
		return nil, err
	}
	views := make([]NodeView, 0, len(nodes))
	for index := range nodes {
		views = append(views, s.view(db, &nodes[index]))
	}
	return views, nil
}

// Get returns one node by uuid.
func (s *NodeService) Get(uuid string) (*NodeView, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	var node model.Node
	err := db.Model(model.Node{}).Where("uuid = ?", uuid).First(&node).Error
	if err != nil {
		if database.IsNotFound(err) {
			return nil, common.NewError("node not found")
		}
		return nil, err
	}
	view := s.view(db, &node)
	return &view, nil
}

// view decorates a stored row with the live status and the assigned count.
func (s *NodeService) view(db *gorm.DB, node *model.Node) NodeView {
	view := NodeView{
		UUID:      node.UUID,
		Name:      node.Name,
		LinkHost:  node.LinkHost,
		Enabled:   node.Enabled,
		UpdatedAt: node.UpdatedAt,
	}
	inventory := DecodeNodeInventory(node.Inventory)
	for _, inbound := range inventory.Inbounds {
		view.Inbounds = append(view.Inbounds, inbound.Tag)
	}
	for _, limiter := range DecodeNodeLimiters(node.Inventory) {
		view.Limiters = append(view.Limiters, limiter.Tag)
	}
	if online, ok := NodeOnline(node.UUID, s.runtime()); ok {
		view.Online = online
	}
	count, err := NodeClientCount(db, node.UUID)
	if err == nil {
		view.ClientCount = count
	}
	return view
}

// NodeOnline asks the running manager whether a node is connected. Without a
// manager service every node reads as offline rather than as an error, because
// the list must still render when the core is down or the build lacks
// with_manager.
func NodeOnline(uuid string, rt *Runtime) (bool, bool) {
	coreInstance := rt.Core()
	if coreInstance == nil {
		return false, false
	}
	online := false
	err := coreInstance.WithFleetManager(func(manager CM.Manager) error {
		status, statusErr := manager.GetNodeStatus(uuid)
		online = statusErr == nil && status == "online"
		return nil
	})
	if err != nil {
		return false, false
	}
	return online, true
}

// NodeClientCount returns how many clients are assigned to a node.
func NodeClientCount(db *gorm.DB, uuid string) (int64, error) {
	if db == nil {
		return 0, nil
	}
	var count int64
	err := db.Model(model.ClientNode{}).Where("node_uuid = ?", uuid).Count(&count).Error
	return count, err
}

// Create stores a node. A node key is generated when the caller does not
// choose one, so two nodes can never share an identity.
func (s *NodeService) Create(node model.Node) (*NodeView, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	node.Name = strings.TrimSpace(node.Name)
	if node.Name == "" {
		return nil, common.NewError("node name is required")
	}
	node.LinkHost = strings.TrimSpace(node.LinkHost)
	if node.UUID == "" {
		generated, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		node.UUID = generated.String()
	} else if _, err := uuid.FromString(node.UUID); err != nil {
		return nil, common.NewError("node uuid is not a uuid")
	}
	if err := db.Create(&node).Error; err != nil {
		return nil, err
	}
	view := s.view(db, &node)
	return &view, nil
}

// Update changes the fields an operator owns. The uuid is the identity a node
// announces, so it is never rewritten here.
func (s *NodeService) Update(uuid string, name string, linkHost string, enabled bool) (*NodeView, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, common.NewError("node name is required")
	}
	result := db.Model(model.Node{}).Where("uuid = ?", uuid).Updates(map[string]any{
		"name":       name,
		"link_host":  strings.TrimSpace(linkHost),
		"enabled":    enabled,
		"updated_at": time.Now().Unix(),
	})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, common.NewError("node not found")
	}
	return s.Get(uuid)
}

// Delete removes a node and every assignment to it, so no client keeps a link
// to a server that no longer exists.
func (s *NodeService) Delete(uuid string) error {
	db := s.db()
	if db == nil {
		return common.NewError("database is not initialized")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("node_uuid = ?", uuid).Delete(model.ClientNode{}).Error; err != nil {
			return err
		}
		return tx.Where("uuid = ?", uuid).Delete(model.Node{}).Error
	})
}

// Clients returns one server-paginated page of the clients assigned to a node.
//
// The filter and the page are applied by the database, not by slicing a full
// slice in the browser: a controller with thousands of clients must not ship
// them all to render 25 rows. query filters on the client name.
func (s *NodeService) Clients(uuid string, limit int, offset int, query string) (*ClientPage, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	if limit <= 0 {
		limit = defaultClientPageSize
	}
	if limit > maxClientPageSize {
		limit = maxClientPageSize
	}
	if offset < 0 {
		offset = 0
	}
	build := func() *gorm.DB {
		tx := db.Model(model.Client{}).
			Joins("JOIN client_nodes ON client_nodes.client_id = clients.id").
			Where("client_nodes.node_uuid = ?", uuid)
		if pattern := clientNamePattern(query); pattern != "" {
			tx = tx.Where("clients.name LIKE ? ESCAPE '\\'", pattern)
		}
		return tx
	}
	page := &ClientPage{Items: []ClientView{}}
	if err := build().Count(&page.Total).Error; err != nil {
		return nil, err
	}
	if page.Total == 0 {
		return page, nil
	}
	err := build().
		Select(`clients.id, clients.name, clients.enable, clients.up, clients.down, clients.volume, clients.expiry, clients."group"`).
		Order("clients.name").
		Limit(limit).
		Offset(offset).
		Scan(&page.Items).Error
	if err != nil {
		return nil, err
	}
	return page, nil
}

// clientNamePattern turns a user query into a LIKE pattern. Wildcards in the
// query are escaped so a search for "50%" matches that text instead of
// everything.
func clientNamePattern(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(query)
	return "%" + escaped + "%"
}

// Assign puts clients on a node. Repeating an existing assignment is a no-op so
// a bulk call that overlaps a previous one stays idempotent.
func (s *NodeService) Assign(uuid string, clientIDs []uint) (int, error) {
	return s.assign(uuid, clientIDs, true)
}

// Unassign takes clients off a node.
func (s *NodeService) Unassign(uuid string, clientIDs []uint) (int, error) {
	return s.assign(uuid, clientIDs, false)
}

func (s *NodeService) assign(uuid string, clientIDs []uint, assigned bool) (int, error) {
	db := s.db()
	if db == nil {
		return 0, common.NewError("database is not initialized")
	}
	if len(clientIDs) == 0 {
		return 0, nil
	}
	var exists int64
	if err := db.Model(model.Node{}).Where("uuid = ?", uuid).Count(&exists).Error; err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, common.NewError("node not found")
	}
	changed := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, clientID := range clientIDs {
			if clientID == 0 {
				continue
			}
			if !assigned {
				result := tx.Where("client_id = ? AND node_uuid = ?", clientID, uuid).Delete(model.ClientNode{})
				if result.Error != nil {
					return result.Error
				}
				changed += int(result.RowsAffected)
				continue
			}
			var present int64
			if err := tx.Model(model.ClientNode{}).
				Where("client_id = ? AND node_uuid = ?", clientID, uuid).
				Count(&present).Error; err != nil {
				return err
			}
			if present > 0 {
				continue
			}
			if err := tx.Create(&model.ClientNode{ClientId: clientID, NodeUUID: uuid}).Error; err != nil {
				return err
			}
			changed++
		}
		return nil
	})
	return changed, err
}

// ClientNodeUUIDs returns the nodes a client is assigned to, for the checkboxes
// in the client card.
func (s *NodeService) ClientNodeUUIDs(clientID uint) ([]string, error) {
	db := s.db()
	if db == nil {
		return nil, nil
	}
	var rows []model.ClientNode
	if err := db.Model(model.ClientNode{}).Where("client_id = ?", clientID).Find(&rows).Error; err != nil {
		return nil, err
	}
	uuids := make([]string, 0, len(rows))
	for _, row := range rows {
		uuids = append(uuids, row.NodeUUID)
	}
	return uuids, nil
}

// SetClientNodes replaces a client's assignment set in one transaction, so the
// checkboxes in the client card are saved atomically.
func (s *NodeService) SetClientNodes(clientID uint, uuids []string) error {
	db := s.db()
	if db == nil {
		return common.NewError("database is not initialized")
	}
	if clientID == 0 {
		return common.NewError("client id is required")
	}
	wanted := make([]string, 0, len(uuids))
	for _, nodeUUID := range uuids {
		if strings.TrimSpace(nodeUUID) == "" {
			continue
		}
		wanted = append(wanted, nodeUUID)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if len(wanted) == 0 {
			return tx.Where("client_id = ?", clientID).Delete(model.ClientNode{}).Error
		}
		if err := tx.Where("client_id = ? AND node_uuid NOT IN ?", clientID, wanted).Delete(model.ClientNode{}).Error; err != nil {
			return err
		}
		for _, nodeUUID := range wanted {
			var present int64
			if err := tx.Model(model.ClientNode{}).
				Where("client_id = ? AND node_uuid = ?", clientID, nodeUUID).
				Count(&present).Error; err != nil {
				return err
			}
			if present > 0 {
				continue
			}
			if err := tx.Create(&model.ClientNode{ClientId: clientID, NodeUUID: nodeUUID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ClearClientNodes drops every assignment for a client. It is the delete path:
// a removed client must not linger on any server.
func ClearClientNodes(tx *gorm.DB, clientID uint) error {
	if tx == nil || clientID == 0 {
		return nil
	}
	return tx.Where("client_id = ?", clientID).Delete(model.ClientNode{}).Error
}
