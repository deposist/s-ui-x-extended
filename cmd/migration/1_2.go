package migration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

type InboundData struct {
	Id      uint
	Tag     string
	Addrs   json.RawMessage
	OutJson json.RawMessage
}

func moveJsonToDb(db *gorm.DB) error {
	binFolderPath := os.Getenv("SUI_BIN_FOLDER")
	if binFolderPath == "" {
		binFolderPath = "bin"
	}
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	configPath := dir + "/" + binFolderPath + "/config.json"
	// #nosec G703 -- configPath derives from the operator-set SUI_BIN_FOLDER under the app dir.
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// #nosec G304 G703 -- configPath derives from the operator-set SUI_BIN_FOLDER under the app dir.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var oldConfig map[string]interface{}
	err = json.Unmarshal(data, &oldConfig)
	if err != nil {
		return err
	}

	oldInbounds := oldConfig["inbounds"].([]interface{})
	legacyInboundRules := make([]interface{}, 0)
	if err := db.Migrator().DropTable(&model.Inbound{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.Inbound{}); err != nil {
		return err
	}
	for _, inbound := range oldInbounds {
		inbObj, _ := inbound.(map[string]interface{})
		tag, _ := inbObj["tag"].(string)
		if tlsObj, ok := inbObj["tls"]; ok {
			var tls_id uint
			err = db.Raw("SELECT id FROM tls WHERE inbounds like ?", `%"`+tag+`"%`).Find(&tls_id).Error
			if err != nil {
				return err
			}

			// Bind or Create tls_id
			if tls_id > 0 {
				inbObj["tls_id"] = tls_id
			} else {
				tls_server, _ := json.MarshalIndent(tlsObj, "", "  ")
				if len(tls_server) > 5 {
					newTls := &model.Tls{
						Name:   tag,
						Server: tls_server,
						Client: json.RawMessage("{}"),
					}
					err = db.Create(newTls).Error
					if err != nil {
						return err
					}
					inbObj["tls_id"] = newTls.Id
				}
			}
		}

		var inbData InboundData
		db.Raw("select id,addrs,out_json from inbound_data where tag = ?", tag).Find(&inbData)
		if inbData.Id > 0 {
			inbObj["out_json"] = inbData.OutJson
			var addrs []map[string]interface{}
			if len(inbData.Addrs) > 0 {
				if err := json.Unmarshal(inbData.Addrs, &addrs); err != nil {
					return err
				}
			}
			for index, addr := range addrs {
				if tlsEnable, ok := addr["tls"].(bool); ok {
					newTls := map[string]interface{}{
						"enabled": tlsEnable,
					}
					if insecure, ok := addr["insecure"].(bool); ok {
						newTls["insecure"] = insecure
						delete(addrs[index], "insecure")
					}
					if sni, ok := addr["server_name"].(string); ok {
						newTls["server_name"] = sni
						delete(addrs[index], "server_name")
					}
					addrs[index]["tls"] = newTls
				}
			}
			inbObj["addrs"] = addrs
		} else {
			inbObj["out_json"] = json.RawMessage("{}")
			inbObj["addrs"] = json.RawMessage("[]")
		}
		legacyInboundRules = appendLegacyInboundRuleActions(legacyInboundRules, tag, inbObj)
		// Delete deprecated fields after their rule-action equivalents were
		// collected above. sing-box 1.11+ rejects these fields on inbounds.
		delete(inbObj, "sniff")
		delete(inbObj, "sniff_override_destination")
		delete(inbObj, "sniff_timeout")
		delete(inbObj, "domain_strategy")
		delete(inbObj, "udp_disable_domain_unmapping")
		inbJson, _ := json.Marshal(inbObj)

		var newInbound model.Inbound
		err = newInbound.UnmarshalJSON(inbJson)
		if err != nil {
			return err
		}
		err = db.Create(&newInbound).Error
		if err != nil {
			return err
		}
	}
	delete(oldConfig, "inbounds")

	blockOutboundTags := []string{}
	dnsOutboundTags := []string{}

	oldOutbounds := oldConfig["outbounds"].([]interface{})
	if err := db.Migrator().DropTable(&model.Outbound{}, &model.Endpoint{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.Outbound{}, &model.Endpoint{}); err != nil {
		return err
	}
	for _, outbound := range oldOutbounds {
		outType, _ := outbound.(map[string]interface{})["type"].(string)
		outboundRaw, _ := json.MarshalIndent(outbound, "", "  ")
		if outType == "wireguard" { // Check if it is Entrypoint
			var newEntrypoint model.Endpoint
			err = newEntrypoint.UnmarshalJSON(outboundRaw)
			if err != nil {
				return err
			}
			err = db.Create(&newEntrypoint).Error
			if err != nil {
				return err
			}
		} else { // It is Outbound
			var newOutbound model.Outbound
			err = newOutbound.UnmarshalJSON(outboundRaw)
			if err != nil {
				return err
			}
			// Delete deprecated fields
			if newOutbound.Type == "direct" {
				var options map[string]interface{}
				if err := json.Unmarshal(newOutbound.Options, &options); err == nil {
					delete(options, "override_address")
					delete(options, "override_port")
					newOutbound.Options, _ = json.Marshal(options)
				}
			}

			switch newOutbound.Type {
			case "dns":
				dnsOutboundTags = append(dnsOutboundTags, newOutbound.Tag)
			case "block":
				blockOutboundTags = append(blockOutboundTags, newOutbound.Tag)
			default:
				err = db.Create(&newOutbound).Error
				if err != nil {
					return err
				}
			}
		}
	}
	delete(oldConfig, "outbounds")

	// Check routing rules
	if routingRules, ok := oldConfig["route"].(map[string]interface{}); ok {
		if rules, hasRules := routingRules["rules"].([]interface{}); hasRules {
			hasDns := false
			for index, rule := range rules {
				ruleObj, _ := rule.(map[string]interface{})
				isBlock := false
				isDns := false
				outboundTag, _ := ruleObj["outbound"].(string)
				for _, tag := range blockOutboundTags {
					if tag == outboundTag {
						isBlock = true
						delete(ruleObj, "outbound")
						ruleObj["action"] = "reject"
						break
					}
				}
				for _, tag := range dnsOutboundTags {
					if tag == outboundTag {
						isDns = true
						hasDns = true
						delete(ruleObj, "outbound")
						ruleObj["action"] = "hijack-dns"
						break
					}
				}
				if !isBlock && !isDns {
					if _, hasAction := ruleObj["action"]; !hasAction {
						ruleObj["action"] = "route"
					}
				}
				rules[index] = ruleObj
			}
			if hasDns {
				rules = append(rules, map[string]interface{}{"action": "sniff"})
			}
			rules = appendMissingLegacyInboundRules(rules, legacyInboundRules)
			routingRules["rules"] = rules
		} else if len(legacyInboundRules) > 0 {
			routingRules["rules"] = legacyInboundRules
		}
		oldConfig["route"] = routingRules
	} else if len(legacyInboundRules) > 0 {
		oldConfig["route"] = map[string]interface{}{"rules": legacyInboundRules}
	}

	// Remove v2rayapi and clashapi from experimental config
	experimental := oldConfig["experimental"].(map[string]interface{})
	delete(experimental, "v2ray_api")
	delete(experimental, "clash_api")
	oldConfig["experimental"] = experimental

	// Save the other configs
	var otherConfigs json.RawMessage
	otherConfigs, err = json.MarshalIndent(oldConfig, "", "  ")
	if err != nil {
		return err
	}

	return db.Save(&model.Setting{
		Key:   "config",
		Value: string(otherConfigs),
	}).Error
}

func appendLegacyInboundRuleActions(rules []interface{}, inboundTag string, inbound map[string]interface{}) []interface{} {
	if inboundTag == "" {
		return rules
	}
	if strategy, _ := inbound["domain_strategy"].(string); strategy != "" {
		rules = append(rules, map[string]interface{}{
			"inbound":  []interface{}{inboundTag},
			"action":   "resolve",
			"strategy": strategy,
		})
	}
	if sniff, _ := inbound["sniff"].(bool); sniff {
		rule := map[string]interface{}{
			"inbound": []interface{}{inboundTag},
			"action":  "sniff",
		}
		if timeout, _ := inbound["sniff_timeout"].(string); timeout != "" {
			rule["timeout"] = timeout
		}
		rules = append(rules, rule)
	}
	if disableUnmapping, _ := inbound["udp_disable_domain_unmapping"].(bool); disableUnmapping {
		rules = append(rules, map[string]interface{}{
			"inbound":                      []interface{}{inboundTag},
			"action":                       "route-options",
			"udp_disable_domain_unmapping": true,
		})
	}
	return rules
}

func appendMissingLegacyInboundRules(rules []interface{}, legacyRules []interface{}) []interface{} {
	prependRules := make([]interface{}, 0)
	for _, legacyRule := range legacyRules {
		legacyMap, ok := legacyRule.(map[string]interface{})
		if !ok {
			continue
		}
		duplicate := false
		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}
			if legacyRouteRulesEquivalent(ruleMap, legacyMap) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			for _, rule := range prependRules {
				ruleMap, ok := rule.(map[string]interface{})
				if ok && legacyRouteRulesEquivalent(ruleMap, legacyMap) {
					duplicate = true
					break
				}
			}
		}
		if !duplicate {
			prependRules = append(prependRules, legacyRule)
		}
	}
	if len(prependRules) == 0 {
		return rules
	}
	return append(prependRules, rules...)
}

func legacyRouteRulesEquivalent(left map[string]interface{}, right map[string]interface{}) bool {
	if left["action"] != right["action"] {
		return false
	}
	if !legacySameInbound(left["inbound"], right["inbound"]) {
		return false
	}
	if !legacySameRulePayload(left, right) {
		return false
	}
	return true
}

func legacySameRulePayload(left map[string]interface{}, right map[string]interface{}) bool {
	for key, value := range right {
		if key == "action" || key == "inbound" {
			continue
		}
		if left[key] != value {
			return false
		}
	}
	for key, value := range left {
		if key == "action" || key == "inbound" {
			continue
		}
		if rightValue, ok := right[key]; !ok || rightValue != value {
			return false
		}
	}
	return true
}

func legacySameInbound(left interface{}, right interface{}) bool {
	return legacyStringSetEqual(legacyStringSet(left), legacyStringSet(right))
}

func legacyStringSet(value interface{}) map[string]bool {
	set := map[string]bool{}
	switch typed := value.(type) {
	case string:
		if typed != "" {
			set[typed] = true
		}
	case []interface{}:
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				set[s] = true
			}
		}
	}
	return set
}

func legacyStringSetEqual(left map[string]bool, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if !right[value] {
			return false
		}
	}
	return true
}

func migrateTls(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&model.Tls{}, "inbounds") {
		return nil
	}
	err := db.Migrator().DropColumn(&model.Tls{}, "inbounds")
	if err != nil {
		return err
	}
	var tlsConfig []model.Tls
	err = db.Model(model.Tls{}).Scan(&tlsConfig).Error
	if err != nil {
		return err
	}

	if len(tlsConfig) == 0 {
		return nil
	}

	for index, tls := range tlsConfig {
		var tlsClient map[string]interface{}
		err = json.Unmarshal(tls.Client, &tlsClient)
		if err != nil {
			continue
		}
		for key := range tlsClient {
			switch key {
			case "insecure", "disable_sni", "utls", "ech", "reality":
				continue
			default:
				delete(tlsClient, key)
			}
		}
		tlsConfig[index].Client, _ = json.MarshalIndent(tlsClient, "", "  ")
	}

	return db.Save(&tlsConfig).Error
}

func dropInboundData(db *gorm.DB) error {
	if !db.Migrator().HasTable(&InboundData{}) {
		return nil
	}
	return db.Migrator().DropTable(&InboundData{})
}

func migrateClients(db *gorm.DB) error {
	var oldClients []model.Client
	err := db.Model(model.Client{}).Scan(&oldClients).Error
	if err != nil {
		return err
	}

	if len(oldClients) == 0 {
		return nil
	}

	for index, oldClient := range oldClients {
		var old_inbounds []string
		err = json.Unmarshal(oldClient.Inbounds, &old_inbounds)
		if err != nil {
			return err
		}
		var inbound_ids []uint
		err = db.Raw("SELECT id FROM inbounds WHERE tag in ?", old_inbounds).Find(&inbound_ids).Error
		if err != nil {
			return err
		}
		oldClients[index].Inbounds, _ = json.Marshal(inbound_ids)
	}
	return db.Save(oldClients).Error
}

func migrateChanges(db *gorm.DB) error {
	return db.Migrator().DropColumn(&model.Changes{}, "index")
}

func to1_2(db *gorm.DB) error {
	err := moveJsonToDb(db)
	if err != nil {
		return err
	}
	err = migrateTls(db)
	if err != nil {
		return err
	}
	err = dropInboundData(db)
	if err != nil {
		return err
	}
	err = migrateClients(db)
	if err != nil {
		return err
	}
	return migrateChanges(db)
}
