package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
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
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	LinkHost       string `json:"linkHost"`
	Enabled        bool   `json:"enabled"`
	Paired         bool   `json:"paired"`
	LastSync       int64  `json:"lastSync,omitempty"`
	PeerBlocked    int    `json:"peerBlocked,omitempty"`
	PeerDisconnect bool   `json:"peerDisconnect,omitempty"`

	// Online reports whether the node is connected right now. An unreachable
	// controller (no manager service running) reports every node offline
	// instead of failing, because the list must still render.
	Online bool `json:"online"`

	// Inbounds and Limiters come from the last announced inventory, so the UI
	// can show what a node can actually accept before assigning anything.
	Inbounds     []string               `json:"inbounds"`
	PeerInbounds []PeerInbound          `json:"peerInbounds,omitempty"`
	Limiters     []NodeInventoryLimiter `json:"limiters"`
	LimiterRoles model.NodeLimiterTags  `json:"limiterRoles"`

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

	// Assigned reports whether this client is already on the node being
	// viewed. The picker lists every client so one dialog can both add and
	// remove, and this flag draws the tick.
	Assigned bool `json:"assigned"`
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
	// Refresh what the connected nodes announced before reading the rows, so
	// the list reflects a node that just reconnected. A core that is down, or a
	// build without the manager, leaves the stored inventory alone.
	if _, err := s.SyncInventory(); err != nil {
		logger.Warning("fleet inventory sync failed: ", err)
	}
	var nodes []model.Node
	if err := db.Model(model.Node{}).Order("name").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return s.views(db, nodes), nil
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

// views decorates stored rows with live status and assigned counts. Both are
// fetched once for the whole list: a fleet of N nodes must not cost N manager
// round-trips and N COUNT queries per refresh.
func (s *NodeService) views(db *gorm.DB, nodes []model.Node) []NodeView {
	uuids := make([]string, 0, len(nodes))
	for index := range nodes {
		uuids = append(uuids, nodes[index].UUID)
	}
	statuses := NodeStatuses(uuids, s.runtime())
	counts := NodeClientCounts(db, uuids)
	views := make([]NodeView, 0, len(nodes))
	for index := range nodes {
		views = append(views, decorateNode(&nodes[index], statuses, counts))
	}
	return views
}

func (s *NodeService) view(db *gorm.DB, node *model.Node) NodeView {
	statuses := NodeStatuses([]string{node.UUID}, s.runtime())
	counts := NodeClientCounts(db, []string{node.UUID})
	return decorateNode(node, statuses, counts)
}

func decorateNode(node *model.Node, statuses map[string]bool, counts map[string]int64) NodeView {
	view := NodeView{
		UUID:        node.UUID,
		Name:        node.Name,
		LinkHost:    node.LinkHost,
		Enabled:     node.Enabled,
		UpdatedAt:   node.UpdatedAt,
		Online:      statuses[node.UUID],
		ClientCount: counts[node.UUID],
	}
	if node.EnrollmentDigest != "" {
		view.Paired = true
		view.LastSync = node.LastPeerSync
		view.PeerBlocked = node.PeerBlocked
		view.PeerDisconnect = node.PeerDisconnect
		view.Online = node.Enabled && node.LastPeerSync > 0 && time.Now().Unix()-node.LastPeerSync < 45
	}
	_ = json.Unmarshal(node.LimiterRoles, &view.LimiterRoles)
	for _, inbound := range DecodeNodeInventory(node.Inventory).Inbounds {
		view.Inbounds = append(view.Inbounds, inbound.Tag)
		if view.Paired && inbound.ID != 0 {
			view.PeerInbounds = append(view.PeerInbounds, PeerInbound{ID: inbound.ID, Tag: inbound.Tag, Type: inbound.Type, Link: inbound.PeerLink})
		}
	}
	view.Limiters = DecodeNodeLimiters(node.Inventory)
	return view
}

// NodeStatuses resolves every requested node in one manager callback. A node
// the manager does not know reads as offline.
func NodeStatuses(uuids []string, rt *Runtime) map[string]bool {
	statuses := make(map[string]bool, len(uuids))
	coreInstance := rt.Core()
	if coreInstance == nil || len(uuids) == 0 {
		return statuses
	}
	_ = coreInstance.WithFleetManager(func(manager CM.Manager) error {
		for _, uuid := range uuids {
			status, err := manager.GetNodeStatus(uuid)
			statuses[uuid] = err == nil && status == "online"
		}
		return nil
	})
	return statuses
}

// NodeClientCounts counts assignments for all nodes in one grouped query.
func NodeClientCounts(db *gorm.DB, uuids []string) map[string]int64 {
	counts := make(map[string]int64, len(uuids))
	if db == nil || len(uuids) == 0 {
		return counts
	}
	type row struct {
		NodeUUID string
		Total    int64
	}
	var rows []row
	if err := db.Model(model.ClientNode{}).
		Select("node_uuid, COUNT(*) AS total").
		Where("node_uuid IN ?", uuids).
		Group("node_uuid").
		Scan(&rows).Error; err != nil {
		return counts
	}
	for _, r := range rows {
		counts[r.NodeUUID] = r.Total
	}
	if err := db.Model(model.PeerAssignment{}).Select("node_uuid, COUNT(DISTINCT client_id) AS total").
		Where("node_uuid IN ?", uuids).Group("node_uuid").Scan(&rows).Error; err == nil {
		for _, row := range rows {
			counts[row.NodeUUID] += row.Total
		}
	}
	return counts
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
	s.projectFleetOrWarn("node creation")
	view := s.view(db, &node)
	return &view, nil
}

// Update changes the fields an operator owns. The uuid is the identity a node
// announces, so it is never rewritten here.
func (s *NodeService) Update(uuid string, name string, linkHost string, enabled bool) (*NodeView, error) {
	return s.update(uuid, name, linkHost, enabled, nil)
}

func (s *NodeService) UpdateWithLimiterRoles(uuid string, name string, linkHost string, enabled bool, roles model.NodeLimiterTags) (*NodeView, error) {
	return s.update(uuid, name, linkHost, enabled, &roles)
}

func (s *NodeService) update(uuid string, name string, linkHost string, enabled bool, roles *model.NodeLimiterTags) (*NodeView, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, common.NewError("node name is required")
	}
	var encoded json.RawMessage
	if roles != nil {
		var node model.Node
		if err := db.Model(model.Node{}).Where("uuid = ?", uuid).First(&node).Error; err != nil {
			if database.IsNotFound(err) {
				return nil, common.NewError("node not found")
			}
			return nil, err
		}
		if err := validateNodeLimiterRoles(DecodeNodeInventory(node.Inventory), *roles); err != nil {
			return nil, err
		}
		var err error
		encoded, err = json.Marshal(roles)
		if err != nil {
			return nil, err
		}
	}
	updates := map[string]any{"name": name, "link_host": strings.TrimSpace(linkHost), "enabled": enabled, "updated_at": time.Now().Unix()}
	if roles != nil {
		updates["limiters"] = encoded
	}
	result := db.Model(model.Node{}).Where("uuid = ?", uuid).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, common.NewError("node not found")
	}
	s.projectFleetOrWarn("node update")
	return s.Get(uuid)
}

func validateNodeLimiterRoles(inventory NodeInventory, roles model.NodeLimiterTags) error {
	available := map[string]string{}
	for _, limiter := range inventory.Limiters {
		available[limiter.Tag] = limiter.Type
	}
	if roles.SpeedDown != "" && roles.SpeedDown == roles.SpeedUp {
		return common.NewError("speedDown and speedUp must be different outbounds")
	}
	for role, outbound := range map[string]string{"speedShared": roles.SpeedShared, "speedDown": roles.SpeedDown, "speedUp": roles.SpeedUp, "connections": roles.Connections} {
		if outbound == "" {
			continue
		}
		type_, ok := available[outbound]
		want := "bandwidth-limiter"
		if role == "connections" {
			want = "connection-limiter"
		}
		if !ok || type_ != want {
			return common.NewErrorf("%s must select an announced %s outbound", role, want)
		}
	}
	return nil
}

// Delete removes a node and every assignment to it, so no client keeps a link
// to a server that no longer exists.
func (s *NodeService) RequestPeerDisconnect(uuid string) error {
	db := s.db()
	if db == nil {
		return common.NewError("database is not initialized")
	}
	updated := db.Model(&model.Node{}).Where("uuid = ? AND enrollment_digest <> ''", uuid).
		UpdateColumns(map[string]any{"peer_disconnect": true, "peer_revision": ""})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return s.Delete(uuid)
	}
	if invalidateSubscriptionCacheAfterSave != nil {
		invalidateSubscriptionCacheAfterSave()
	}
	return nil
}

func (s *NodeService) Delete(uuid string) error {
	db := s.db()
	if db == nil {
		return common.NewError("database is not initialized")
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("node_uuid = ?", uuid).Delete(model.ClientNode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("node_uuid = ?", uuid).Delete(model.PeerAssignment{}).Error; err != nil {
			return err
		}
		return tx.Where("uuid = ?", uuid).Delete(model.Node{}).Error
	}); err != nil {
		return err
	}
	s.projectFleetOrWarn("node deletion")
	if invalidateSubscriptionCacheAfterSave != nil {
		invalidateSubscriptionCacheAfterSave()
	}
	return nil
}

// Clients returns one server-paginated page of clients for a node.
//
// Every client is listed, each marked with whether it is already on this
// server, so one dialog both adds and removes. The filter and the page are
// applied by the database, not by slicing a full slice in the browser: a
// controller with thousands of clients must not ship them all to render 25
// rows. query filters on the client name; only assigned limits the result to
// the server's own clients.
func (s *NodeService) Clients(uuid string, limit int, offset int, query string, onlyAssigned bool) (*ClientPage, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	return nodeClients(db, db.Model(model.ClientNode{}).Where("node_uuid = ?", uuid), limit, offset, query, onlyAssigned)
}

func nodeClients(db, assignments *gorm.DB, limit, offset int, query string, onlyAssigned bool) (*ClientPage, error) {
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
		tx := db.Model(model.Client{})
		if onlyAssigned {
			tx = tx.Where("clients.id IN (?)", assignments.Session(&gorm.Session{}).Select("client_id"))
		}
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
	type clientRow struct {
		Id     uint   `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Enable bool   `gorm:"column:enable"`
		Up     int64  `gorm:"column:up"`
		Down   int64  `gorm:"column:down"`
		Volume int64  `gorm:"column:volume"`
		Expiry int64  `gorm:"column:expiry"`
		Group  string `gorm:"column:group"`
	}
	var rows []clientRow
	err := build().
		Select(`clients.id, clients.name, clients.enable, clients.up, clients.down, clients.volume, clients.expiry, clients."group"`).
		Order("clients.name").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if !onlyAssigned && len(rows) > 0 {
		// Mark the page's rows that are already on this server with one query,
		// so the picker can show ticks without asking per row.
		ids := make([]uint, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.Id)
		}
		var assignedIDs []uint
		if err := assignments.Session(&gorm.Session{}).Where("client_id IN ?", ids).
			Pluck("client_id", &assignedIDs).Error; err != nil {
			return nil, err
		}
		onNode := make(map[uint]struct{}, len(assignedIDs))
		for _, id := range assignedIDs {
			onNode[id] = struct{}{}
		}
		page.Items = make([]ClientView, 0, len(rows))
		for _, row := range rows {
			_, assigned := onNode[row.Id]
			page.Items = append(page.Items, ClientView{
				Id: row.Id, Name: row.Name, Enable: row.Enable,
				Up: row.Up, Down: row.Down, Volume: row.Volume,
				Expiry: row.Expiry, Group: row.Group, Assigned: assigned,
			})
		}
		return page, nil
	}
	page.Items = make([]ClientView, 0, len(rows))
	for _, row := range rows {
		page.Items = append(page.Items, ClientView{
			Id: row.Id, Name: row.Name, Enable: row.Enable,
			Up: row.Up, Down: row.Down, Volume: row.Volume,
			Expiry: row.Expiry, Group: row.Group, Assigned: true,
		})
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
	var node model.Node
	if err := db.Where("uuid = ?", uuid).Take(&node).Error; err != nil {
		return 0, err
	}
	if assigned && node.EnrollmentDigest != "" {
		return 0, common.NewError("paired nodes cannot use legacy client assignments")
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
	if err != nil {
		return changed, err
	}
	if changed > 0 {
		s.projectFleetOrWarn("client node assignment")
	}
	return changed, nil
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
	seen := make(map[string]struct{}, len(uuids))
	for _, nodeUUID := range uuids {
		nodeUUID = strings.TrimSpace(nodeUUID)
		if nodeUUID == "" {
			continue
		}
		if _, done := seen[nodeUUID]; done {
			continue
		}
		seen[nodeUUID] = struct{}{}
		wanted = append(wanted, nodeUUID)
	}
	if len(wanted) > 0 {
		// Reject unknown nodes instead of storing an assignment row that no
		// projection would ever read and no node would ever delete.
		var known int64
		if err := db.Model(model.Node{}).Where("uuid IN ? AND COALESCE(enrollment_digest, '') = ''", wanted).Count(&known).Error; err != nil {
			return err
		}
		if int(known) != len(wanted) {
			return common.NewError("one or more nodes no longer exist or do not support legacy assignments")
		}
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
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
	}); err != nil {
		return err
	}
	s.projectFleetOrWarn("client node assignment")
	return nil
}

// ClearClientNodes drops every assignment for a client. It is the delete path:
// a removed client must not linger on any server.
func ClearClientNodes(tx *gorm.DB, clientID uint) error {
	if tx == nil || clientID == 0 {
		return nil
	}
	if err := tx.Where("client_id = ?", clientID).Delete(model.PeerAssignment{}).Error; err != nil {
		return err
	}
	return tx.Where("client_id = ?", clientID).Delete(model.ClientNode{}).Error
}
