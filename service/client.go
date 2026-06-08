package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util"
	"github.com/deposist/s-ui-x/util/common"

	"gorm.io/gorm"
)

const clientTrafficOverLimitCondition = "volume > 0 AND (up > volume OR down > volume OR up > volume - down)"

type ClientService struct {
	Runtime *Runtime
}

func (s *ClientService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (s *ClientService) setLastUpdate(value int64) {
	s.runtime().updates().Set(value)
}

func decodeClientInbounds(clientID uint, raw json.RawMessage, operation string) ([]uint, bool) {
	var inbounds []uint
	if err := json.Unmarshal(raw, &inbounds); err != nil {
		logger.Warningf("%s skipped client %d with invalid inbounds: %v", operation, clientID, err)
		return nil, false
	}
	return inbounds, true
}

func decodeClientLinks(clientID uint, raw json.RawMessage, operation string) ([]map[string]string, bool) {
	// A migrated (or freshly inserted) client can have a NULL/empty Links
	// column. Treat that as "no links yet" rather than an error, otherwise the
	// link-regeneration paths skip the client and its inbounds never appear in
	// the subscription even after an inbound edit.
	if len(bytes.TrimSpace(raw)) == 0 {
		return []map[string]string{}, true
	}
	var links []map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		logger.Warningf("%s skipped client %d with invalid links: %v", operation, clientID, err)
		return nil, false
	}
	return links, true
}

// buildLinksForInbounds generates the "local" link entries for the given
// inbounds. The result is always a non-nil slice so an empty result marshals to
// `[]`, never `null` (the NULL Links class of bug — see decodeClientLinks).
func buildLinksForInbounds(config json.RawMessage, inbounds []model.Inbound, hostname string) []map[string]string {
	links := []map[string]string{}
	for i := range inbounds {
		for _, uri := range util.LinkGenerator(config, &inbounds[i], hostname) {
			links = append(links, map[string]string{
				"remark": inbounds[i].Tag,
				"type":   "local",
				"uri":    uri,
			})
		}
	}
	return links
}

// rebuildClientLinks is the single implementation shared by every inbound-driven
// link-regeneration path: regenerate the local links for inbounds, then
// re-append the client's previously stored links that satisfy keep. The only
// thing that differs between call sites is the keep predicate. Returns ok=false
// (and leaves the caller to skip the client) when the stored links are invalid.
func rebuildClientLinks(clientID uint, config, rawLinks json.RawMessage, inbounds []model.Inbound, hostname string, keep func(link map[string]string) bool, operation string) (json.RawMessage, bool, error) {
	clientLinks, ok := decodeClientLinks(clientID, rawLinks, operation)
	if !ok {
		return nil, false, nil
	}
	newClientLinks := buildLinksForInbounds(config, inbounds, hostname)
	for _, clientLink := range clientLinks {
		if keep(clientLink) {
			newClientLinks = append(newClientLinks, clientLink)
		}
	}
	marshaled, err := json.MarshalIndent(newClientLinks, "", "  ")
	if err != nil {
		return nil, true, err
	}
	return marshaled, true, nil
}

func (s *ClientService) Get(id string) (*[]model.Client, error) {
	if id == "" {
		return s.GetAll()
	}
	return s.getById(id)
}

func (s *ClientService) getById(id string) (*[]model.Client, error) {
	db := database.GetDB()
	var client []model.Client
	err := db.Model(model.Client{}).Where("id in ?", strings.Split(id, ",")).Scan(&client).Error
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (s *ClientService) GetAll() (*[]model.Client, error) {
	db := database.GetDB()
	var clients []model.Client
	err := db.Model(model.Client{}).
		Select("`id`, `enable`, `name`, `sub_secret`, `desc`, `group`, `inbounds`, `up`, `down`, `volume`, `expiry`, `limit_ip`, `ip_limit_mode`, `last_online`, `last_ip_count`").
		Scan(&clients).Error
	if err != nil {
		return nil, err
	}
	return &clients, nil
}

func (s *ClientService) Save(tx *gorm.DB, act string, data json.RawMessage, hostname string) ([]uint, error) {
	var err error
	var inboundIds []uint

	switch act {
	case "new", "edit":
		var client model.Client
		err = json.Unmarshal(data, &client)
		if err != nil {
			return nil, err
		}
		err = s.prepareClientSubSecret(tx, &client, act == "edit")
		if err != nil {
			return nil, err
		}
		err = s.updateLinksWithFixedInbounds(tx, []*model.Client{&client}, hostname)
		if err != nil {
			return nil, err
		}
		if act == "edit" {
			// Find changed inbounds
			inboundIds, err = s.findInboundsChanges(tx, &client, false)
			if err != nil {
				return nil, err
			}
		} else {
			err = json.Unmarshal(client.Inbounds, &inboundIds)
			if err != nil {
				return nil, err
			}
		}
		err = tx.Save(&client).Error
		if err != nil {
			return nil, err
		}
	case "addbulk":
		var clients []*model.Client
		err = json.Unmarshal(data, &clients)
		if err != nil {
			return nil, err
		}
		if len(clients) == 0 {
			return inboundIds, nil
		}
		// addbulk clients all share the same inbound set (the frontend forces an
		// identical Inbounds array), so clients[0] is representative here.
		err = json.Unmarshal(clients[0].Inbounds, &inboundIds)
		if err != nil {
			return nil, err
		}
		for _, client := range clients {
			err = s.prepareClientSubSecret(tx, client, false)
			if err != nil {
				return nil, err
			}
		}
		err = s.updateLinksWithFixedInbounds(tx, clients, hostname)
		if err != nil {
			return nil, err
		}
		err = database.SaveInBatchesSafe(tx, clients)
		if err != nil {
			return nil, err
		}
	case "editbulk":
		var clients []*model.Client
		err = json.Unmarshal(data, &clients)
		if err != nil {
			return nil, err
		}
		for _, client := range clients {
			err = s.prepareClientSubSecret(tx, client, true)
			if err != nil {
				return nil, err
			}
			changedInboundIds, err := s.findInboundsChanges(tx, client, true)
			if err != nil {
				return nil, err
			}
			if len(changedInboundIds) > 0 {
				inboundIds = common.UnionUintArray(inboundIds, changedInboundIds)
			}
		}
		if len(inboundIds) > 0 {
			err = s.updateLinksWithFixedInbounds(tx, clients, hostname)
			if err != nil {
				return nil, err
			}
		}
		err = database.SaveInBatchesSafe(tx, clients)
		if err != nil {
			return nil, err
		}
	case "delbulk":
		var ids []uint
		err = json.Unmarshal(data, &ids)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			var client model.Client
			err = tx.Where("id = ?", id).First(&client).Error
			if err != nil {
				return nil, err
			}
			var clientInbounds []uint
			err = json.Unmarshal(client.Inbounds, &clientInbounds)
			if err != nil {
				return nil, err
			}
			inboundIds = common.UnionUintArray(inboundIds, clientInbounds)
		}
		err = tx.Where("id in ?", ids).Delete(model.Client{}).Error
		if err != nil {
			return nil, err
		}
	case "del":
		var id uint
		err = json.Unmarshal(data, &id)
		if err != nil {
			return nil, err
		}
		var client model.Client
		err = tx.Where("id = ?", id).First(&client).Error
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(client.Inbounds, &inboundIds)
		if err != nil {
			return nil, err
		}
		err = tx.Where("id = ?", id).Delete(model.Client{}).Error
		if err != nil {
			return nil, err
		}
	default:
		return nil, common.NewErrorf("unknown action: %s", act)
	}

	return inboundIds, nil
}

// clientChangeNameJSON marshals a client name as a JSON string for the
// Changes.Obj payload. Building it by raw concatenation ("\"" + name + "\"")
// breaks when the name contains a quote, backslash or control character: the
// resulting json.RawMessage is invalid and later fails json.Marshal of the
// whole changes feed (CheckChanges then returns an empty body for all admins).
func clientChangeNameJSON(name string) json.RawMessage {
	b, err := json.Marshal(name)
	if err != nil {
		return json.RawMessage(`""`)
	}
	return b
}

func (s *ClientService) updateLinksWithFixedInbounds(tx *gorm.DB, clients []*model.Client, hostname string) error {
	// Each client may carry a different inbound set (notably act="editbulk", where
	// ClientEditBulk.vue preserves per-client inbounds), so the inbound list used
	// to regenerate a client's local links must come from THAT client's own
	// Inbounds — not from clients[0], which would corrupt subscriptions for every
	// client whose inbound set differs from the first one. Preloaded inbound rows
	// are memoised by the raw Inbounds JSON so the common case of one shared set
	// (act="addbulk", act="new"/"edit") still issues a single query.
	inboundCache := map[string][]model.Inbound{}
	for index, client := range clients {
		var inboundIds []uint
		if err := json.Unmarshal(client.Inbounds, &inboundIds); err != nil {
			return err
		}
		cacheKey := string(client.Inbounds)
		inbounds, cached := inboundCache[cacheKey]
		if !cached {
			// Zero inbounds means removing local links only.
			if len(inboundIds) > 0 {
				if err := tx.Model(model.Inbound{}).Preload("Tls").
					Where("id in ? and type in ?", inboundIds, util.InboundTypeWithLink).
					Find(&inbounds).Error; err != nil {
					return err
				}
			}
			inboundCache[cacheKey] = inbounds
		}
		// Keep links that aren't locally generated; regenerate the local ones for
		// this client's own fixed inbounds.
		links, ok, err := rebuildClientLinks(client.Id, client.Config, client.Links, inbounds, hostname, func(link map[string]string) bool {
			return link["type"] != "local"
		}, "fixed inbound link update")
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		// #nosec G602 -- index is the range index over clients; never out of range.
		clients[index].Links = links
	}
	return nil
}

func (s *ClientService) UpdateClientsOnInboundAdd(tx *gorm.DB, initIds string, inboundId uint, hostname string) error {
	clientIds := strings.Split(initIds, ",")
	var clients []model.Client
	err := tx.Model(model.Client{}).Where("id in ?", clientIds).Find(&clients).Error
	if err != nil {
		return err
	}
	var inbound model.Inbound
	err = tx.Model(model.Inbound{}).Preload("Tls").Where("id = ?", inboundId).Find(&inbound).Error
	if err != nil {
		return err
	}
	for _, client := range clients {
		// Add inbounds
		clientInbounds, ok := decodeClientInbounds(client.Id, client.Inbounds, "inbound add")
		if !ok {
			continue
		}
		clientInbounds = append(clientInbounds, inboundId)
		client.Inbounds, err = json.MarshalIndent(clientInbounds, "", "  ")
		if err != nil {
			return err
		}
		// Regenerate the added inbound's links; keep links for other inbounds.
		links, decoded, lerr := rebuildClientLinks(client.Id, client.Config, client.Links, []model.Inbound{inbound}, hostname, func(link map[string]string) bool {
			return link["remark"] != inbound.Tag
		}, "inbound add")
		if lerr != nil {
			return lerr
		}
		if !decoded {
			continue
		}
		client.Links = links
		if err = tx.Save(&client).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ClientService) UpdateClientsOnInboundDelete(tx *gorm.DB, id uint, tag string) error {
	var clientIds []uint
	err := tx.Raw("SELECT clients.id FROM clients, json_each(clients.inbounds) AS je WHERE je.value = ?", id).Scan(&clientIds).Error
	if err != nil {
		return err
	}
	if len(clientIds) == 0 {
		return nil
	}
	var clients []model.Client
	err = tx.Model(model.Client{}).Where("id IN ?", clientIds).Find(&clients).Error
	if err != nil {
		return err
	}
	for _, client := range clients {
		// Delete inbounds
		clientInbounds, ok := decodeClientInbounds(client.Id, client.Inbounds, "inbound delete")
		if !ok {
			continue
		}
		var newClientInbounds []uint
		for _, clientInbound := range clientInbounds {
			if clientInbound != id {
				newClientInbounds = append(newClientInbounds, clientInbound)
			}
		}
		client.Inbounds, err = json.MarshalIndent(newClientInbounds, "", "  ")
		if err != nil {
			return err
		}
		// Delete links
		clientLinks, ok := decodeClientLinks(client.Id, client.Links, "inbound delete")
		if !ok {
			continue
		}
		var newClientLinks []map[string]string
		for _, clientLink := range clientLinks {
			if clientLink["remark"] != tag {
				newClientLinks = append(newClientLinks, clientLink)
			}
		}
		client.Links, err = json.MarshalIndent(newClientLinks, "", "  ")
		if err != nil {
			return err
		}
		err = tx.Save(&client).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ClientService) UpdateLinksByInboundChange(tx *gorm.DB, inbounds *[]model.Inbound, hostname string, oldTag string) error {
	var err error
	for _, inbound := range *inbounds {
		var clientIds []uint
		err = tx.Raw("SELECT clients.id FROM clients, json_each(clients.inbounds) AS je WHERE je.value = ?", inbound.Id).Scan(&clientIds).Error
		if err != nil {
			return err
		}
		if len(clientIds) == 0 {
			continue
		}
		var clients []model.Client
		err = tx.Model(model.Client{}).Where("id IN ?", clientIds).Find(&clients).Error
		if err != nil {
			return err
		}
		for _, client := range clients {
			// Regenerate this inbound's links; keep non-local links and local
			// links for other inbounds (neither the new tag nor the old tag).
			links, decoded, lerr := rebuildClientLinks(client.Id, client.Config, client.Links, []model.Inbound{inbound}, hostname, func(link map[string]string) bool {
				return link["type"] != "local" || (link["remark"] != inbound.Tag && link["remark"] != oldTag)
			}, "inbound link update")
			if lerr != nil {
				return lerr
			}
			if !decoded {
				continue
			}
			client.Links = links
			if err = tx.Save(&client).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ClientService) DepleteClients() (inboundIds []uint, err error) {
	var clients []model.Client
	var changes []model.Changes

	dt := time.Now().Unix()
	db := database.GetDB()

	tx := db.Begin()
	defer func() {
		if err == nil {
			err = tx.Commit().Error
			if err != nil {
				return
			}
			if err1 := db.Exec("PRAGMA wal_checkpoint(FULL)").Error; err1 != nil {
				logger.Error("Error checkpointing WAL: ", err1.Error())
			}
		} else {
			tx.Rollback()
		}
	}()

	// Reset clients
	inboundIds, err = s.ResetClients(tx, dt)
	if err != nil {
		return nil, err
	}

	// Deplete clients
	err = tx.Model(model.Client{}).Where("enable = true AND (("+clientTrafficOverLimitCondition+") OR (expiry > 0 AND expiry < ?))", dt).Scan(&clients).Error
	if err != nil {
		return nil, err
	}

	for _, client := range clients {
		logger.Debug("Client ", client.Name, " is going to be disabled")
		userInbounds, ok := decodeClientInbounds(client.Id, client.Inbounds, "client deplete")
		if !ok {
			continue
		}
		// Find changed inbounds
		inboundIds = common.UnionUintArray(inboundIds, userInbounds)
		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "DepleteJob",
			Key:      "clients",
			Action:   "disable",
			Obj:      clientChangeNameJSON(client.Name),
		})
	}

	// Save changes
	if len(changes) > 0 {
		err = tx.Model(model.Client{}).Where("enable = true AND (("+clientTrafficOverLimitCondition+") OR (expiry > 0 AND expiry < ?))", dt).Update("enable", false).Error
		if err != nil {
			return nil, err
		}
		err = database.CreateInBatchesSafe(tx.Model(model.Changes{}), &changes)
		if err != nil {
			return nil, err
		}
		s.setLastUpdate(dt)
	}

	return inboundIds, nil
}

func (s *ClientService) ResetClients(tx *gorm.DB, dt int64) ([]uint, error) {
	var err error
	var resetClients []*model.Client
	var changes []model.Changes
	var inboundIds []uint
	// Set delay start without periodic reset
	err = tx.Model(model.Client{}).
		Where("enable = true AND delay_start = true AND auto_reset = false AND (up > 0 OR down > 0)").Find(&resetClients).Error
	if err != nil {
		return nil, err
	}
	for _, client := range resetClients {
		client.Expiry = dt + (int64(client.ResetDays) * 86400)
		client.DelayStart = false
		if err := updateClientResetFields(tx, client.Id, map[string]interface{}{
			"expiry":      client.Expiry,
			"delay_start": client.DelayStart,
		}); err != nil {
			return nil, err
		}
		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "ResetJob",
			Key:      "clients",
			Action:   "reset",
			Obj:      clientChangeNameJSON(client.Name),
		})
	}

	// Set delay start with periodic reset
	resetClients = nil
	err = tx.Model(model.Client{}).
		Where("enable = true AND delay_start = true AND auto_reset = true AND (up > 0 OR down > 0)").Find(&resetClients).Error
	if err != nil {
		return nil, err
	}
	for _, client := range resetClients {
		client.NextReset = dt + (int64(client.ResetDays) * 86400)
		client.DelayStart = false
		if err := updateClientResetFields(tx, client.Id, map[string]interface{}{
			"next_reset":  client.NextReset,
			"delay_start": client.DelayStart,
		}); err != nil {
			return nil, err
		}
		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "ResetJob",
			Key:      "clients",
			Action:   "reset",
			Obj:      clientChangeNameJSON(client.Name),
		})
	}

	// Set periodic reset
	resetClients = nil
	err = tx.Model(model.Client{}).
		Where("delay_start = false AND auto_reset = true AND next_reset < ?", dt).Find(&resetClients).Error
	if err != nil {
		return nil, err
	}
	for _, client := range resetClients {
		if !client.Enable {
			clientInboundIds, ok := decodeClientInbounds(client.Id, client.Inbounds, "client reset")
			if !ok {
				continue
			}
			inboundIds = common.UnionUintArray(inboundIds, clientInboundIds)
		}
		client.NextReset = dt + (int64(client.ResetDays) * 86400)
		client.TotalUp += client.Up
		client.TotalDown += client.Down
		client.Up = 0
		client.Down = 0
		if !client.Enable {
			client.Enable = true
		}
		if err := updateClientResetFields(tx, client.Id, map[string]interface{}{
			"next_reset": client.NextReset,
			"total_up":   client.TotalUp,
			"total_down": client.TotalDown,
			"up":         client.Up,
			"down":       client.Down,
			"enable":     client.Enable,
		}); err != nil {
			return nil, err
		}
	}

	// Save changes
	if len(changes) > 0 {
		err = database.CreateInBatchesSafe(tx.Model(model.Changes{}), &changes)
		if err != nil {
			return nil, err
		}
		s.setLastUpdate(dt)
	}
	return inboundIds, nil
}

func updateClientResetFields(tx *gorm.DB, clientID uint, values map[string]interface{}) error {
	return tx.Model(model.Client{}).Where("id = ?", clientID).Updates(values).Error
}

func (s *ClientService) findInboundsChanges(tx *gorm.DB, client *model.Client, fillOmitted bool) ([]uint, error) {
	var err error
	var oldClient model.Client
	var oldInboundIds, newInboundIds []uint
	err = tx.Model(model.Client{}).Where("id = ?", client.Id).First(&oldClient).Error
	if err != nil {
		return nil, err
	}
	if fillOmitted {
		client.Links = oldClient.Links
		client.Config = oldClient.Config
	}
	err = json.Unmarshal(oldClient.Inbounds, &oldInboundIds)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(client.Inbounds, &newInboundIds)
	if err != nil {
		return nil, err
	}

	// Check client.Config changes
	if !bytes.Equal(oldClient.Config, client.Config) ||
		oldClient.Name != client.Name ||
		oldClient.Enable != client.Enable {
		return common.UnionUintArray(oldInboundIds, newInboundIds), nil
	}

	// Check client.Inbounds changes
	diffInbounds := common.DiffUintArray(oldInboundIds, newInboundIds)

	return diffInbounds, nil
}
