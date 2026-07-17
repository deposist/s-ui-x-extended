package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

type ClientService struct {
	Runtime *Runtime
}

// clientIsActiveAt is the shared source of truth for eligibility decisions that
// must agree with depletion and AWG reconciliation. Expiry is an exclusive Unix
// seconds boundary, and reaching the traffic quota exhausts it. The subtraction
// form avoids overflowing when Up+Down exceeds int64.
func clientIsActiveAt(client model.Client, now int64) bool {
	if !client.Enable || (client.Expiry > 0 && client.Expiry <= now) {
		return false
	}
	if client.Up < 0 || client.Down < 0 {
		return false
	}
	if client.Volume <= 0 {
		return true
	}
	return client.Up < client.Volume &&
		client.Down < client.Volume &&
		client.Up < client.Volume-client.Down
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
// `[]`, never `null` (the NULL Links class of bug - see decodeClientLinks).
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
	var clients []model.Client
	err := db.Model(model.Client{}).Where("id in ?", strings.Split(id, ",")).Scan(&clients).Error
	if err != nil {
		return nil, err
	}
	for index := range clients {
		var accesses []model.ClientEndpointAccess
		if err := db.Select("endpoint_id").Where("client_id = ?", clients[index].Id).Order("endpoint_id").Find(&accesses).Error; err != nil {
			return nil, err
		}
		clients[index].AWGEndpoints = make([]uint, len(accesses))
		for accessIndex := range accesses {
			clients[index].AWGEndpoints[accessIndex] = accesses[accessIndex].EndpointId
		}
	}
	return &clients, nil
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
		if err = ensureClientSudokuKeys(tx, &client); err != nil {
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
		if err := ReplaceClientAWGEndpointAccess(tx, client.Id, client.AWGEndpoints); err != nil {
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
			if err = ensureClientSudokuKeys(tx, client); err != nil {
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
		sudokuDeliveryChanged := false
		err = json.Unmarshal(data, &clients)
		if err != nil {
			return nil, err
		}
		for _, client := range clients {
			err = s.prepareClientSubSecret(tx, client, true)
			if err != nil {
				return nil, err
			}
			// Bulk limit/inbound edits may omit Config. Preserve it before
			// validating so existing credentials are neither erased nor rejected.
			if len(bytes.TrimSpace(client.Config)) == 0 {
				var oldConfig model.Client
				if err = tx.Model(model.Client{}).Select("config").Where("id = ?", client.Id).First(&oldConfig).Error; err != nil {
					return nil, err
				}
				client.Config = oldConfig.Config
			}
			if err = ensureClientSudokuKeys(tx, client); err != nil {
				return nil, err
			}
			var oldClient model.Client
			if err = tx.Model(model.Client{}).Select("config").Where("id = ?", client.Id).First(&oldClient).Error; err != nil {
				return nil, err
			}
			if !clientSudokuDeliveryEqual(oldClient.Config, client.Config) {
				sudokuDeliveryChanged = true
			}
			changedInboundIds, err := s.findInboundsChanges(tx, client, true)
			if err != nil {
				return nil, err
			}
			if len(changedInboundIds) > 0 {
				inboundIds = common.UnionUintArray(inboundIds, changedInboundIds)
			}
		}
		if len(inboundIds) > 0 || sudokuDeliveryChanged {
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
			// An id already gone (concurrent delete / stale client list) is a
			// no-op, not a failure: deleting an absent client still leaves the
			// caller with the intended end state (client absent).
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
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
		// Deleting a client that is already gone (a stale UI row, a concurrent
		// delete from another session, or a resubmitted request) is an
		// idempotent no-op instead of a "record not found" failure - the
		// intended end state (client absent) already holds.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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
	// Inbounds - not from clients[0], which would corrupt subscriptions for every
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
					Where("id in ?", inboundIds).
					Find(&inbounds).Error; err != nil {
					return err
				}
			}
			inboundCache[cacheKey] = inbounds
		}

		for _, inbound := range inbounds {
			config, backfilled, err := backfillClientProtocol(client.Config, inbound.Type, client.Name)
			if err != nil {
				return err
			}
			if backfilled {
				clients[index].Config = config
				client.Config = config
			}
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

// linkBackfillInboundTypes are the inbound types that gained a local URI link in
// a later build than the one that first stored a client's links. A client
// assigned to one of these before the upgrade has a links blob with no entry for
// it, and nothing regenerates that until the operator next saves the client.
// RegenerateMissingLocalLinks closes that gap at startup.
var linkBackfillInboundTypes = []string{"sudoku", "mieru"}

// RegenerateMissingLocalLinks is an idempotent startup backfill: for every client
// assigned to an inbound whose type only recently started producing a local link
// (sudoku, mieru), it regenerates the client's local links and persists them if
// they changed. Clients that already have the links, or are assigned to no such
// inbound, are left untouched and cause no write.
//
// It runs once per startup after InitDB. hostname is the fallback advertised host
// used only when an inbound has no explicit Addrs; the same value the request
// path passes to LinkGenerator.
func (s *ClientService) RegenerateMissingLocalLinks(hostname string) error {
	db := database.GetDB()

	// Inbound ids of the backfill types, plus a set for quick membership tests.
	var backfillInbounds []model.Inbound
	if err := db.Model(model.Inbound{}).Preload("Tls").
		Where("type in ?", linkBackfillInboundTypes).
		Find(&backfillInbounds).Error; err != nil {
		return err
	}
	if len(backfillInbounds) == 0 {
		return nil // no sudoku/mieru inbounds; nothing to backfill
	}
	backfillInboundIDs := make(map[uint]struct{}, len(backfillInbounds))
	for _, in := range backfillInbounds {
		backfillInboundIDs[in.Id] = struct{}{}
	}

	var clients []model.Client
	if err := db.Model(model.Client{}).Find(&clients).Error; err != nil {
		return err
	}

	inboundCache := map[string][]model.Inbound{}
	updated := 0
	for i := range clients {
		client := &clients[i]
		var inboundIds []uint
		if err := json.Unmarshal(client.Inbounds, &inboundIds); err != nil {
			logger.Warningf("link backfill: skipped client %d with invalid inbounds: %v", client.Id, err)
			continue
		}
		// Only touch clients assigned to at least one backfill inbound.
		hasBackfill := false
		for _, id := range inboundIds {
			if _, ok := backfillInboundIDs[id]; ok {
				hasBackfill = true
				break
			}
		}
		if !hasBackfill {
			continue
		}

		cacheKey := string(client.Inbounds)
		inbounds, cached := inboundCache[cacheKey]
		if !cached {
			if len(inboundIds) > 0 {
				if err := db.Model(model.Inbound{}).Preload("Tls").
					Where("id in ?", inboundIds).
					Find(&inbounds).Error; err != nil {
					return err
				}
			}
			inboundCache[cacheKey] = inbounds
		}

		// Backfill any missing per-protocol credentials first (mieru needs a
		// name/password to produce a link; sudoku is keyless and unaffected).
		config := client.Config
		configChanged := false
		for _, inbound := range inbounds {
			newConfig, backfilled, err := backfillClientProtocol(config, inbound.Type, client.Name)
			if err != nil {
				return err
			}
			if backfilled {
				config = newConfig
				configChanged = true
			}
		}

		links, ok, err := rebuildClientLinks(client.Id, config, client.Links, inbounds, hostname, func(link map[string]string) bool {
			return link["type"] != "local"
		}, "startup link backfill")
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		// Persist only when something actually changed, so a second startup is a
		// no-op and untouched clients are never rewritten.
		if !configChanged && bytes.Equal(normalizeLinksJSON(client.Links), normalizeLinksJSON(links)) {
			continue
		}
		client.Config = config
		client.Links = links
		if err := db.Model(model.Client{}).Where("id = ?", client.Id).
			Updates(map[string]any{"config": client.Config, "links": client.Links}).Error; err != nil {
			return err
		}
		updated++
	}
	if updated > 0 {
		logger.Infof("link backfill: regenerated local links for %d client(s) with sudoku/mieru inbounds", updated)
	}
	return nil
}

// normalizeLinksJSON re-marshals a links blob into a canonical form so the
// change check compares content, not whitespace. Invalid JSON is returned as-is
// so a malformed blob still counts as "changed" and gets rewritten.
func normalizeLinksJSON(raw json.RawMessage) []byte {
	var v []map[string]string
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

// linkBackfillOnce guards the lazy link backfill so it runs at most once per
// process, on the first data load that carries a usable hostname.
var linkBackfillOnce sync.Once

// RegenerateMissingLocalLinksOnce runs the sudoku/mieru link backfill a single
// time per process, but only once it has a non-empty hostname. The startup path
// has no request host and settings.webDomain is usually blank, which left links
// with an empty server and produced nothing; the panel's data-load path does
// carry the host the operator reached the panel on (the same value a manual
// re-save uses), so this is called from there. An empty hostname is ignored so
// the Once is not consumed before a real host is available.
func (s *ClientService) RegenerateMissingLocalLinksOnce(hostname string) {
	if strings.TrimSpace(hostname) == "" {
		return
	}
	linkBackfillOnce.Do(func() {
		if err := s.RegenerateMissingLocalLinks(hostname); err != nil {
			logger.Warning("link backfill (lazy): ", err)
		}
	})
}

// RegenerateAllClientLinks force-rebuilds the local links for every client on
// all of their assigned inbounds, keeping their non-local (external) links. It
// backs the manual "regenerate client links and QR" button, so unlike the lazy
// backfill it always runs, covers every link type, and uses the host the
// operator reached the panel on. Returns the number of clients processed.
func (s *ClientService) RegenerateAllClientLinks(hostname string) (int, error) {
	if strings.TrimSpace(hostname) == "" {
		return 0, common.NewError("cannot regenerate links without a hostname")
	}
	db := database.GetDB()
	var clients []model.Client
	if err := db.Model(model.Client{}).Find(&clients).Error; err != nil {
		return 0, err
	}
	if len(clients) == 0 {
		return 0, nil
	}
	ptrs := make([]*model.Client, len(clients))
	for i := range clients {
		ptrs[i] = &clients[i]
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := s.updateLinksWithFixedInbounds(tx, ptrs, hostname); err != nil {
			return err
		}
		return database.SaveInBatchesSafe(tx, ptrs)
	})
	if err != nil {
		return 0, err
	}
	logger.Infof("regenerated client links for %d client(s) on operator request", len(ptrs))
	return len(ptrs), nil
}

var mtProtoFrontHosts = []string{
	"www.microsoft.com", "www.apple.com", "www.cloudflare.com", "www.amazon.com",
	"aws.amazon.com", "dl.google.com", "www.icloud.com", "www.bing.com", "www.tesla.com",
}

func randomMTProtoSecret() (string, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	host := mtProtoFrontHosts[int(key[0])%len(mtProtoFrontHosts)]
	return "ee" + hex.EncodeToString(key) + hex.EncodeToString([]byte(host)), nil
}

func randomSSPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// backfillClientProtocol injects a missing per-protocol credentials block into a
// client's JSON config. It is safe to call on existing or malformed configs; it
// returns ok=true only if a new block was added.
func backfillClientProtocol(config json.RawMessage, inboundType string, clientName string) (json.RawMessage, bool, error) {
	field, ok := userJSONField[inboundType]
	if !ok {
		return config, false, nil // Protocol has no user config
	}

	var cfg map[string]map[string]any
	if len(bytes.TrimSpace(config)) == 0 {
		cfg = make(map[string]map[string]any)
	} else if err := json.Unmarshal(config, &cfg); err != nil {
		return config, false, nil // Malformed config, skip backfill
	}

	if _, exists := cfg[field]; exists {
		return config, false, nil
	}

	mixedPassword := common.Random(10)

	newObj := map[string]any{"password": mixedPassword}
	switch field {
	case "mixed", "socks", "http", "naive":
		newObj["username"] = clientName
	case "shadowsocks":
		var ssPw string
		if inboundType == "shadowsocks16" {
			ss, err := randomSSPassword(16)
			if err != nil {
				return config, false, err
			}
			ssPw = ss
		} else {
			ss, err := randomSSPassword(32)
			if err != nil {
				return config, false, err
			}
			ssPw = ss
		}
		newObj["password"] = ssPw
		newObj["name"] = clientName
	case "shadowtls":
		ss, err := randomSSPassword(32)
		if err != nil {
			return config, false, err
		}
		newObj["password"] = ss
		newObj["name"] = clientName
	case "vmess":
		u, err := uuid.NewV4()
		if err != nil {
			return config, false, err
		}
		newObj = map[string]any{"name": clientName, "uuid": u.String(), "alterId": 0}
	case "vless":
		u, err := uuid.NewV4()
		if err != nil {
			return config, false, err
		}
		newObj = map[string]any{"name": clientName, "uuid": u.String(), "flow": "xtls-rprx-vision"}
	case "tuic":
		u, err := uuid.NewV4()
		if err != nil {
			return config, false, err
		}
		newObj["name"] = clientName
		newObj["uuid"] = u.String()
	case "hysteria":
		newObj = map[string]any{"name": clientName, "auth_str": mixedPassword}
	case "mtproxy":
		mtSecret, err := randomMTProtoSecret()
		if err != nil {
			return config, false, err
		}
		newObj = map[string]any{"name": clientName, "secret": mtSecret}
	default:
		newObj["name"] = clientName
	}

	cfg[field] = newObj
	marshaled, err := json.MarshalIndent(cfg, "", "  ")
	return marshaled, true, err
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

		// SR-019: guarantee per-protocol credentials exist before rebuilding links
		// and generating core users, to prevent `users is empty` validation errors
		// on newly supported protocols or legacy clients.
		config, backfilled, err := backfillClientProtocol(client.Config, inbound.Type, client.Name)
		if err != nil {
			return err
		}
		if backfilled {
			client.Config = config
		}
		if err := ensureClientSudokuKeys(tx, &client); err != nil {
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
		if err := ensureClientSudokuKeys(tx, &client); err != nil {
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
			// SR-019: guarantee per-protocol credentials exist before rebuilding links
			// and generating core users, to prevent `users is empty` validation errors.
			config, backfilled, err := backfillClientProtocol(client.Config, inbound.Type, client.Name)
			if err != nil {
				return err
			}
			if backfilled {
				client.Config = config
			}
			if err := ensureClientSudokuKeys(tx, &client); err != nil {
				return err
			}

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
	var depletedClientIDs []uint

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
			if len(depletedClientIDs) > 0 {
				if hook := s.runtime().AWGClientStateHook(); hook != nil {
					if hookErr := hook.SuspendClients(context.Background(), depletedClientIDs); hookErr != nil {
						logger.Warning("AWG suspend after client depletion failed: ", hookErr)
					}
				}
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

	// Deplete clients using the same overflow-safe boundary predicate as other
	// client eligibility consumers.
	err = tx.Model(model.Client{}).Where("enable = true").Scan(&clients).Error
	if err != nil {
		return nil, err
	}
	inactiveClients := clients[:0]
	for _, client := range clients {
		if !clientIsActiveAt(client, dt) {
			inactiveClients = append(inactiveClients, client)
		}
	}
	clients = inactiveClients

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
		clientIDs := make([]uint, 0, len(clients))
		for _, client := range clients {
			clientIDs = append(clientIDs, client.Id)
		}
		depletedClientIDs = append(depletedClientIDs, clientIDs...)
		err = tx.Model(model.Client{}).Where("enable = true AND id IN ?", clientIDs).Update("enable", false).Error
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

// clientResetPeriodDays returns a client's periodic-reset interval in days,
// clamped to at least 1. The API save path persists model.Client verbatim, so an
// apiv2/import caller can set auto_reset=true with reset_days=0 (the Vue UI forbids
// it). Without the clamp NextReset == dt, and the @every-1m DepleteJob re-matches
// and zeroes the client's traffic every minute, permanently defeating quota.
func clientResetPeriodDays(resetDays int) int64 {
	if resetDays < 1 {
		return 1
	}
	return int64(resetDays)
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
		client.NextReset = dt + (clientResetPeriodDays(client.ResetDays) * 86400)
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
		client.NextReset = dt + (clientResetPeriodDays(client.ResetDays) * 86400)
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
		if len(bytes.TrimSpace(client.Links)) == 0 {
			client.Links = oldClient.Links
		}
		if len(bytes.TrimSpace(client.Config)) == 0 {
			client.Config = oldClient.Config
		}
	}
	err = json.Unmarshal(oldClient.Inbounds, &oldInboundIds)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(client.Inbounds, &newInboundIds)
	if err != nil {
		return nil, err
	}

	// A Sudoku split-key edit changes only links/subscriptions. It must not
	// schedule a server inbound hot reload because Sudoku has no server users.
	if !clientCoreConfigEqual(oldClient.Config, client.Config) ||
		oldClient.Name != client.Name ||
		oldClient.Enable != client.Enable {
		return common.UnionUintArray(oldInboundIds, newInboundIds), nil
	}

	// Check client.Inbounds changes
	diffInbounds := common.DiffUintArray(oldInboundIds, newInboundIds)

	return diffInbounds, nil
}
