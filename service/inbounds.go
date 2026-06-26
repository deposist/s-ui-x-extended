package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

type InboundService struct {
	ClientService
	Runtime *Runtime
}

func (s *InboundService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

type inboundListItem struct {
	id           uint
	data         map[string]interface{}
	includeUsers bool
}

type inboundUserNameRow struct {
	InboundID uint
	Name      string
}

func (s *InboundService) Get(ids string) (*[]map[string]interface{}, error) {
	if ids == "" {
		return s.GetAll()
	}
	return s.getById(ids)
}

func (s *InboundService) getById(ids string) (*[]map[string]interface{}, error) {
	var inbound []model.Inbound
	var result []map[string]interface{}
	db := database.GetDB()
	err := db.Model(model.Inbound{}).Where("id in ?", strings.Split(ids, ",")).Scan(&inbound).Error
	if err != nil {
		return nil, err
	}
	for _, inb := range inbound {
		inbData, err := inb.MarshalFull()
		if err != nil {
			return nil, err
		}
		result = append(result, *inbData)
	}
	return &result, nil
}

func (s *InboundService) GetAll() (*[]map[string]interface{}, error) {
	db := database.GetDB()
	inbounds := []model.Inbound{}
	err := db.Model(model.Inbound{}).Scan(&inbounds).Error
	if err != nil {
		return nil, err
	}
	items := make([]inboundListItem, 0, len(inbounds))
	userInboundIDs := make([]uint, 0, len(inbounds))
	for _, inbound := range inbounds {
		var shadowtls_version uint
		ss_managed := false
		inbData := map[string]interface{}{
			"id":     inbound.Id,
			"type":   inbound.Type,
			"tag":    inbound.Tag,
			"tls_id": inbound.TlsId,
		}
		if inbound.Options != nil {
			var restFields map[string]json.RawMessage
			if err := json.Unmarshal(inbound.Options, &restFields); err != nil {
				return nil, err
			}
			inbData["listen"] = restFields["listen"]
			inbData["listen_port"] = restFields["listen_port"]
			if inbound.Type == "shadowtls" {
				_ = json.Unmarshal(restFields["version"], &shadowtls_version)
			}
			if inbound.Type == "shadowsocks" {
				_ = json.Unmarshal(restFields["managed"], &ss_managed)
			}
		}
		includeUsers := s.hasUser(inbound.Type) &&
			!(inbound.Type == "shadowtls" && shadowtls_version < 3) &&
			!(inbound.Type == "shadowsocks" && ss_managed)
		if includeUsers {
			userInboundIDs = append(userInboundIDs, inbound.Id)
		}

		items = append(items, inboundListItem{
			id:           inbound.Id,
			data:         inbData,
			includeUsers: includeUsers,
		})
	}
	usersByInbound, err := fetchInboundUserNames(db, userInboundIDs)
	if err != nil {
		return nil, err
	}
	data := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if item.includeUsers {
			item.data["users"] = usersByInbound[item.id]
		}
		data = append(data, item.data)
	}
	return &data, nil
}

func fetchInboundUserNames(db *gorm.DB, inboundIDs []uint) (map[uint][]string, error) {
	usersByInbound := make(map[uint][]string, len(inboundIDs))
	if len(inboundIDs) == 0 {
		return usersByInbound, nil
	}
	for _, id := range inboundIDs {
		usersByInbound[id] = []string{}
	}

	var rows []inboundUserNameRow
	err := db.Raw(`
		SELECT je.value AS inbound_id, clients.name
		FROM clients, json_each(clients.inbounds) AS je
		WHERE je.value IN ?
		ORDER BY clients.id, je.key
	`, inboundIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		usersByInbound[row.InboundID] = append(usersByInbound[row.InboundID], row.Name)
	}
	return usersByInbound, nil
}

func (s *InboundService) FromIds(ids []uint) ([]*model.Inbound, error) {
	db := database.GetDB()
	inbounds := []*model.Inbound{}
	err := db.Model(model.Inbound{}).Where("id in ?", ids).Scan(&inbounds).Error
	if err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) Save(tx *gorm.DB, act string, data json.RawMessage, initUserIds string, hostname string) (*entityCoreChange, error) {
	switch act {
	case "new", "edit":
		return s.saveInboundUpsert(tx, act, data, initUserIds, hostname)
	case "del":
		return s.saveInboundDelete(tx, data)
	default:
		return nil, common.NewErrorf("unknown action: %s", act)
	}
}

// validateBondOrFailoverInbound validates that the inbounds list for a bond or
// native core-failover inbound references existing inbound tags.
func validateBondOrFailoverInbound(tx *gorm.DB, inbound model.Inbound) error {
	opts := optionsMapOf(inbound.Options)
	members, _ := opts["inbounds"].([]any)
	if len(members) == 0 {
		return common.NewErrorf("%s inbound needs at least one inbound", inbound.Type)
	}
	for _, item := range members {
		tag, _ := item.(string)
		if tag == "" {
			return common.NewErrorf("%s inbound member must be a string tag", inbound.Type)
		}
		if tag == inbound.Tag {
			return common.NewErrorf("a %s inbound cannot reference itself", inbound.Type)
		}
		var member model.Inbound
		if err := tx.Model(model.Inbound{}).Where("tag = ?", tag).First(&member).Error; err != nil {
			return common.NewErrorf("%s inbound member %q does not exist", inbound.Type, tag)
		}
		if member.Type == "bond" || member.Type == CoreFailoverType {
			return common.NewErrorf("%s inbound member %q must be a plain inbound, not group type %q", inbound.Type, tag, member.Type)
		}
	}
	return nil
}

func (s *InboundService) saveInboundUpsert(tx *gorm.DB, act string, data json.RawMessage, initUserIds string, hostname string) (*entityCoreChange, error) {
	var inbound model.Inbound
	if err := inbound.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	if inbound.TlsId > 0 {
		if err := tx.Model(model.Tls{}).Where("id = ?", inbound.TlsId).Find(&inbound.Tls).Error; err != nil {
			return nil, err
		}
	}
	var oldTag string
	if act == "edit" {
		if err := tx.Model(model.Inbound{}).Select("tag").Where("id = ?", inbound.Id).Find(&oldTag).Error; err != nil {
			return nil, err
		}
		if oldTag != "" && oldTag != inbound.Tag {
			refs, err := inboundTagReferences(tx, oldTag)
			if err != nil {
				return nil, err
			}
			if len(refs) > 0 {
				return nil, formatTagReferenceError("inbound", oldTag, refs)
			}
		}
	}

	if err := util.FillOutJson(&inbound, hostname); err != nil {
		return nil, err
	}
	if inbound.Type == "bond" || inbound.Type == "core-failover" {
		if err := validateBondOrFailoverInbound(tx, inbound); err != nil {
			return nil, err
		}
	}
	if err := tx.Save(&inbound).Error; err != nil {
		return nil, err
	}
	var err error
	switch act {
	case "new":
		err = s.ClientService.UpdateClientsOnInboundAdd(tx, initUserIds, inbound.Id, hostname)
	case "edit":
		err = s.ClientService.UpdateLinksByInboundChange(tx, &[]model.Inbound{inbound}, hostname, oldTag)
	}
	if err != nil {
		return nil, err
	}

	change := &entityCoreChange{reloadIds: []uint{inbound.Id}}
	if oldTag != "" && oldTag != inbound.Tag {
		change.removeTags = []string{oldTag}
	}
	// ssm-api services capture the managed inbound adapter at construction, so
	// they must be recreated after the inbound itself was hot-reloaded.
	change.cascadeServiceIds, err = ssmCascadeServiceIds(tx, inbound.Tag)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *InboundService) saveInboundDelete(tx *gorm.DB, data json.RawMessage) (*entityCoreChange, error) {
	var tag string
	if err := json.Unmarshal(data, &tag); err != nil {
		return nil, err
	}
	refs, err := inboundTagReferences(tx, tag)
	if err != nil {
		return nil, err
	}
	if len(refs) > 0 {
		return nil, formatTagReferenceError("inbound", tag, refs)
	}
	var id uint
	if err := tx.Model(model.Inbound{}).Select("id").Where("tag = ?", tag).Scan(&id).Error; err != nil {
		return nil, err
	}
	if err := s.ClientService.UpdateClientsOnInboundDelete(tx, id, tag); err != nil {
		return nil, err
	}
	if err := tx.Where("tag = ?", tag).Delete(model.Inbound{}).Error; err != nil {
		return nil, err
	}
	return &entityCoreChange{removeTags: []string{tag}}, nil
}

func (s *InboundService) UpdateOutJsons(tx *gorm.DB, inboundIds []uint, hostname string) error {
	var inbounds []model.Inbound
	err := tx.Model(model.Inbound{}).Preload("Tls").Where("id in ?", inboundIds).Find(&inbounds).Error
	if err != nil {
		return err
	}
	for _, inbound := range inbounds {
		err = util.FillOutJson(&inbound, hostname)
		if err != nil {
			return err
		}
		err = tx.Model(model.Inbound{}).Where("tag = ?", inbound.Tag).Update("out_json", inbound.OutJson).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *InboundService) GetAllConfig(db *gorm.DB) ([]json.RawMessage, error) {
	var inboundsJson []json.RawMessage
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Preload("Tls").Find(&inbounds).Error
	if err != nil {
		return nil, err
	}
	nestedTags, err := nestedInboundMemberTags(inbounds)
	if err != nil {
		return nil, err
	}
	for _, inbound := range inbounds {
		if nestedTags[inbound.Tag] {
			continue
		}
		inboundJson, err := inboundCoreJSON(s, db, *inbound)
		if err != nil {
			return nil, err
		}
		inboundsJson = append(inboundsJson, inboundJson)
	}
	return inboundsJson, nil
}

func (s *InboundService) hasUser(inboundType string) bool {
	_, ok := userJSONField[inboundType]
	return ok
}

// userJSONField maps an inbound type to the JSON path used inside
// clients.config to locate per-user data. Do not extend this map without a
// positive list for both the inbound type and the JSON field value.
var userJSONField = map[string]string{
	"mixed":         "mixed",
	"socks":         "socks",
	"http":          "http",
	"shadowsocks":   "shadowsocks",
	"shadowsocks16": "shadowsocks",
	"vmess":         "vmess",
	"trojan":        "trojan",
	"naive":         "naive",
	"hysteria":      "hysteria",
	"shadowtls":     "shadowtls",
	"tuic":          "tuic",
	"hysteria2":     "hysteria2",
	"vless":         "vless",
	"anytls":        "anytls",
	"mieru":         "mieru",
	"trusttunnel":   "trusttunnel",
	"ssh":           "ssh",
	"mtproxy":       "mtproxy",
}

var allowedUserJSONFields = map[string]struct{}{
	"mixed":       {},
	"socks":       {},
	"http":        {},
	"shadowsocks": {},
	"vmess":       {},
	"trojan":      {},
	"naive":       {},
	"hysteria":    {},
	"shadowtls":   {},
	"tuic":        {},
	"hysteria2":   {},
	"vless":       {},
	"anytls":      {},
	"mieru":       {},
	"trusttunnel": {},
	"ssh":         {},
	"mtproxy":     {},
}

func (s *InboundService) addUsers(db *gorm.DB, inboundJson []byte, inboundId uint, inboundType string) ([]byte, error) {
	if !s.hasUser(inboundType) {
		return inboundJson, nil
	}

	var inbound map[string]interface{}
	err := json.Unmarshal(inboundJson, &inbound)
	if err != nil {
		return nil, err
	}

	// A Trojan inbound authenticates per user; sing-box has no top-level
	// "password" field for it (only "users") and rejects the whole config
	// (`unknown field "password"`) if one is present. The inbound editor used to
	// write one for inbounds, so drop any leftover before emitting.
	if inboundType == "trojan" {
		delete(inbound, "password")
	}

	condition := "? IN (SELECT json_each.value FROM json_each(clients.inbounds))"
	inbound["users"], err = s.fetchUsersByCondition(db, inboundType, condition, inbound, inboundId)
	if err != nil {
		return nil, err
	}

	return json.Marshal(inbound)
}

func (s *InboundService) fetchUsersByCondition(db *gorm.DB, inboundType string, condition string, inbound map[string]interface{}, args ...interface{}) ([]json.RawMessage, error) {
	if inboundType == "shadowtls" {
		version, _ := inbound["version"].(float64)
		if int(version) < 3 {
			return nil, nil
		}
	}
	if inboundType == "shadowsocks" {
		method, _ := inbound["method"].(string)
		if method == "2022-blake3-aes-128-gcm" {
			inboundType = "shadowsocks16"
		}
	}

	field, ok := userJSONField[inboundType]
	if !ok {
		return nil, common.NewErrorf("unsupported inbound type for user lookup: %s", inboundType)
	}
	if _, ok := allowedUserJSONFields[field]; !ok {
		return nil, common.NewErrorf("unsupported user JSON field for user lookup: %s", field)
	}

	var userRows []sql.NullString
	// `field` is constrained to a static allow-list above, so embedding it
	// directly into the JSON path is safe. The dynamic condition is fed
	// through the query parameter slot to remain SQL-injection free. Missing
	// per-protocol client config is scanned as SQL NULL and skipped below.
	query := fmt.Sprintf(`SELECT json_extract(clients.config, '$.%s') FROM clients WHERE enable = true AND %s`, field, condition)
	err := db.Raw(query, args...).Scan(&userRows).Error
	if err != nil {
		return nil, err
	}
	// `xtls-rprx-vision` is strictly TCP. Xray-core rejects any vless
	// inbound that advertises the flow over a non-TCP transport (grpc,
	// ws, http, httpupgrade, ...) or without TLS. Strip the flow string
	// here so a single client UUID can be reused across multiple vless
	// inbounds with different transports without breaking the non-TCP
	// inbound (issue #1127).
	stripVisionFlow := false
	if inboundType == "vless" {
		if inbound["tls"] == nil {
			stripVisionFlow = true
		} else if transport, ok := inbound["transport"].(map[string]interface{}); ok {
			if tt, _ := transport["type"].(string); tt != "" && tt != "tcp" {
				stripVisionFlow = true
			}
		}
	}
	usersJson := make([]json.RawMessage, 0, len(userRows))
	for _, userRow := range userRows {
		if !userRow.Valid {
			continue
		}
		user := userRow.String
		if stripVisionFlow {
			user = strings.Replace(user, "xtls-rprx-vision", "", -1)
		}
		usersJson = append(usersJson, json.RawMessage(user))
	}
	return usersJson, nil
}

// RemoveInboundsFromCore removes the given inbound tags from the running core
// and closes their tracked connections. Missing tags are tolerated so
// removals stay idempotent; with no running core there is nothing to remove.
func (s *InboundService) RemoveInboundsFromCore(tags []string) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	for _, tag := range tags {
		if err := coreInstance.RemoveInbound(tag); err != nil && err != os.ErrInvalid {
			return err
		}
		// The core may have been stopped concurrently, so guard the instance.
		if instance := coreInstance.GetInstance(); instance != nil {
			if tracker := instance.ConnTracker(); tracker != nil {
				tracker.CloseConnByInbound(tag)
			}
		}
	}
	return nil
}

func (s *InboundService) RestartInbounds(tx *gorm.DB, ids []uint) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	var inbounds []*model.Inbound
	err := tx.Model(model.Inbound{}).Preload("Tls").Where("id in ?", ids).Find(&inbounds).Error
	if err != nil {
		return err
	}
	for _, inbound := range inbounds {
		err = coreInstance.RemoveInbound(inbound.Tag)
		if err != nil && err != os.ErrInvalid {
			return err
		}
		// Close all existing connections. The core may have been stopped
		// concurrently (cron / user restart), so guard against a nil instance.
		if instance := coreInstance.GetInstance(); instance != nil {
			if tracker := instance.ConnTracker(); tracker != nil {
				tracker.CloseConnByInbound(inbound.Tag)
			}
		}

		inboundConfig, err := inbound.MarshalJSON()
		if err != nil {
			return err
		}
		inboundConfig, err = s.addUsers(tx, inboundConfig, inbound.Id, inbound.Type)
		if err != nil {
			return err
		}
		err = coreInstance.AddInbound(inboundConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
