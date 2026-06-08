package sub

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util"
	"github.com/deposist/s-ui-x/util/common"
)

const defaultJson = `
{
  "inbounds": [
    {
      "type": "tun",
      "address": [
				"172.19.0.1/30",
				"fdfe:dcba:9876::1/126"
			],
      "mtu": 9000,
      "auto_route": true,
      "strict_route": false,
      "endpoint_independent_nat": false,
      "stack": "system",
      "platform": {
        "http_proxy": {
          "enabled": true,
          "server": "127.0.0.1",
          "server_port": 2080
        }
      }
    },
    {
      "type": "mixed",
      "listen": "127.0.0.1",
      "listen_port": 2080,
      "users": []
    }
  ]
}
`

type JsonService struct {
	service.SettingService
	LinkService
}

func (j *JsonService) GetJson(subId string, format string) (*string, []string, error) {
	var jsonConfig map[string]interface{}

	enabled, err := j.SettingService.GetSubJsonEnable()
	if err == nil && !enabled {
		return nil, nil, common.NewError("json subscription disabled")
	}

	client, inDatas, err := j.getData(subId)
	if err != nil {
		return nil, nil, err
	}

	outbounds, outTags, err := j.getOutbounds(client.Config, inDatas)
	if err != nil {
		return nil, nil, err
	}

	links := j.LinkService.GetLinks(&client.Links, "external", "")
	tagNumEnable := 0
	if len(links) > 1 {
		tagNumEnable = 1
	}
	for index, link := range links {
		json, tag, err := util.GetOutbound(link, (index+1)*tagNumEnable)
		if err == nil && len(tag) > 0 {
			*outbounds = append(*outbounds, *json)
			*outTags = append(*outTags, tag)
		}
	}

	j.addDefaultOutbounds(outbounds, outTags)

	err = json.Unmarshal([]byte(defaultJson), &jsonConfig)
	if err != nil {
		return nil, nil, err
	}

	jsonConfig["outbounds"] = outbounds

	// Add other objects from settings
	if err := j.addOthers(&jsonConfig); err != nil {
		return nil, nil, err
	}

	result, _ := json.MarshalIndent(jsonConfig, "", "  ")
	resultStr := string(result)

	headers := safeSubscriptionHeaders(buildClientHeaders(client, cachedSubDisplaySettings(&j.SettingService, time.Now())))

	return &resultStr, headers, nil
}

func (j *JsonService) getData(subId string) (*model.Client, []*model.Inbound, error) {
	db := database.GetDB()
	client, err := (&SubService{}).getClientBySubId(subId)
	if err != nil {
		return nil, nil, err
	}
	var clientInbounds []uint
	err = json.Unmarshal(client.Inbounds, &clientInbounds)
	if err != nil {
		return nil, nil, err
	}
	var inbounds []*model.Inbound
	err = db.Model(model.Inbound{}).Preload("Tls").Where("id in ?", clientInbounds).Find(&inbounds).Error
	if err != nil {
		return nil, nil, err
	}
	return client, inbounds, nil
}

func (j *JsonService) getOutbounds(clientConfig json.RawMessage, inbounds []*model.Inbound) (*[]map[string]interface{}, *[]string, error) {
	var outbounds []map[string]interface{}
	var configs map[string]interface{}
	var outTags []string

	err := json.Unmarshal(clientConfig, &configs)
	if err != nil {
		return nil, nil, err
	}
	for _, inData := range inbounds {
		if len(inData.OutJson) < 5 {
			continue
		}
		var outbound map[string]interface{}
		err = json.Unmarshal(inData.OutJson, &outbound)
		if err != nil {
			return nil, nil, err
		}
		protocol, _ := outbound["type"].(string)

		// Shadowsocks
		if protocol == "shadowsocks" {
			var userPass []string
			var inbOptions map[string]interface{}
			err = json.Unmarshal(inData.Options, &inbOptions)
			if err != nil {
				return nil, nil, err
			}
			method, _ := inbOptions["method"].(string)
			if strings.HasPrefix(method, "2022") {
				inbPass, _ := inbOptions["password"].(string)
				userPass = append(userPass, inbPass)
			}
			var pass string
			if method == "2022-blake3-aes-128-gcm" {
				if m, ok := configs["shadowsocks16"].(map[string]interface{}); ok {
					pass, _ = m["password"].(string)
				}
			} else {
				if m, ok := configs["shadowsocks"].(map[string]interface{}); ok {
					pass, _ = m["password"].(string)
				}
			}
			userPass = append(userPass, pass)
			outbound["password"] = strings.Join(userPass, ":")
		} else { // Other protocols
			// Drop user-level `flow` when the inbound transport is not
			// plain TCP. Xray-core only accepts `xtls-rprx-vision` over
			// TCP and rejects the connection on any other transport
			// (issue #1127). Detect transport type from the merged
			// outbound JSON so the same UUID can be reused across vless
			// inbounds with different transports.
			stripFlow := false
			if protocol == "vless" {
				if inData.TlsId == 0 {
					stripFlow = true
				} else if tr, ok := outbound["transport"].(map[string]interface{}); ok {
					if tt, _ := tr["type"].(string); tt != "" && tt != "tcp" {
						stripFlow = true
					}
				}
			}
			config, _ := configs[protocol].(map[string]interface{})
			for key, value := range config {
				if key == "name" || key == "alterId" || (key == "flow" && (inData.TlsId == 0 || stripFlow)) {
					continue
				}
				outbound[key] = value
			}
		}

		var addrs []map[string]interface{}
		err = json.Unmarshal(inData.Addrs, &addrs)
		if err != nil {
			return nil, nil, err
		}
		tag, _ := outbound["tag"].(string)
		if len(addrs) == 0 {
			// For mixed protocol, use separated socks and http
			if protocol == "mixed" {
				outbound["tag"] = tag
				j.pushMixed(&outbounds, &outTags, outbound)
			} else {
				outTags = append(outTags, tag)
				outbounds = append(outbounds, outbound)
			}
		} else {
			for index, addr := range addrs {
				// Copy original config
				newOut := make(map[string]interface{}, len(outbound))
				for key, value := range outbound {
					newOut[key] = value
				}
				// Change and push copied config
				newOut["server"], _ = addr["server"].(string)
				port, _ := addr["server_port"].(float64)
				newOut["server_port"] = int(port)

				// Override TLS
				if addrTls, ok := addr["tls"].(map[string]interface{}); ok {
					outTls, _ := newOut["tls"].(map[string]interface{})
					if outTls == nil {
						outTls = make(map[string]interface{})
					}
					for key, value := range addrTls {
						outTls[key] = value
					}
					newOut["tls"] = outTls
				}

				remark, _ := addr["remark"].(string)
				newTag := fmt.Sprintf("%d.%s%s", index+1, tag, remark)
				newOut["tag"] = newTag
				// For mixed protocol, use separated socks and http
				if protocol == "mixed" {
					j.pushMixed(&outbounds, &outTags, newOut)
				} else {
					outTags = append(outTags, newTag)
					outbounds = append(outbounds, newOut)
				}
			}
		}
	}
	return &outbounds, &outTags, nil
}

func (j *JsonService) addDefaultOutbounds(outbounds *[]map[string]interface{}, outTags *[]string) {
	outbound := []map[string]interface{}{
		{
			"outbounds": append([]string{"auto", "direct"}, *outTags...),
			"tag":       "proxy",
			"type":      "selector",
		},
		{
			"tag":       "auto",
			"type":      "urltest",
			"outbounds": outTags,
			"url":       "http://www.gstatic.com/generate_204",
			"interval":  "10m",
			"tolerance": 50,
		},
		{
			"type": "direct",
			"tag":  "direct",
		},
	}
	*outbounds = append(outbound, *outbounds...)
}

func (j *JsonService) addOthers(jsonConfig *map[string]interface{}) error {
	if err := j.addFragment(jsonConfig); err != nil {
		return err
	}
	if err := j.addNoises(jsonConfig); err != nil {
		return err
	}
	if err := j.addMux(jsonConfig); err != nil {
		return err
	}

	rules_start := []interface{}{
		map[string]interface{}{
			"action": "sniff",
		},
		map[string]interface{}{
			"clash_mode": "Direct",
			"action":     "route",
			"outbound":   "direct",
		},
	}
	rules_end := []interface{}{
		map[string]interface{}{
			"clash_mode": "Global",
			"action":     "route",
			"outbound":   "proxy",
		},
	}
	route := map[string]interface{}{
		"auto_detect_interface": true,
		"final":                 "proxy",
		"rules":                 rules_start,
	}

	othersStr, err := j.SettingService.GetSubJsonExt()
	if err != nil {
		return err
	}
	if len(othersStr) == 0 {
		if err := j.addDirectRules(route); err != nil {
			return err
		}
		(*jsonConfig)["route"] = route
		return nil
	}
	var othersJson map[string]interface{}
	err = json.Unmarshal([]byte(othersStr), &othersJson)
	if err != nil {
		return err
	}
	if _, ok := othersJson["log"]; ok {
		(*jsonConfig)["log"] = othersJson["log"]
	}
	if _, ok := othersJson["dns"]; ok {
		(*jsonConfig)["dns"] = othersJson["dns"]
	}
	if _, ok := othersJson["inbounds"]; ok {
		(*jsonConfig)["inbounds"] = othersJson["inbounds"]
	}
	if _, ok := othersJson["experimental"]; ok {
		(*jsonConfig)["experimental"] = othersJson["experimental"]
	}
	if _, ok := othersJson["rule_set"]; ok {
		route["rule_set"] = othersJson["rule_set"]
	}
	if settingRules, ok := othersJson["rules"].([]interface{}); ok {
		rules := append(rules_start, settingRules...)
		route["rules"] = append(rules, rules_end...)
	}
	if defaultDomainResolver, ok := othersJson["default_domain_resolver"].(string); ok {
		route["default_domain_resolver"] = defaultDomainResolver
	}
	if err := j.addDirectRules(route); err != nil {
		return err
	}
	(*jsonConfig)["route"] = route

	return nil
}

func (j *JsonService) addDirectRules(route map[string]interface{}) error {
	enabled, err := j.SettingService.GetSubJsonDirectRules()
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	route["rule_set"] = mergeDirectRuleSets(route["rule_set"])
	rules, _ := route["rules"].([]interface{})
	route["rules"] = insertDirectRouteRules(rules)
	return nil
}

func insertDirectRouteRules(rules []interface{}) []interface{} {
	directRule := map[string]interface{}{
		"rule_set": []string{"geosite-private", "geoip-private"},
		"action":   "route",
		"outbound": "direct",
	}
	if len(rules) == 0 {
		return []interface{}{directRule}
	}
	result := make([]interface{}, 0, len(rules)+1)
	result = append(result, rules[0], directRule)
	result = append(result, rules[1:]...)
	return result
}

func mergeDirectRuleSets(existing interface{}) []interface{} {
	result := make([]interface{}, 0)
	seen := map[string]bool{}
	if ruleSets, ok := existing.([]interface{}); ok {
		for _, ruleSet := range ruleSets {
			if tag, ok := ruleSetTag(ruleSet); ok {
				seen[tag] = true
			}
			result = append(result, ruleSet)
		}
	}
	for _, ruleSet := range directRuleSets() {
		tag, _ := ruleSetTag(ruleSet)
		if seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, ruleSet)
	}
	return result
}

func ruleSetTag(ruleSet interface{}) (string, bool) {
	ruleSetMap, ok := ruleSet.(map[string]interface{})
	if !ok {
		return "", false
	}
	tag, ok := ruleSetMap["tag"].(string)
	return tag, ok && tag != ""
}

func directRuleSets() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"tag":             "geosite-private",
			"type":            "remote",
			"format":          "binary",
			"url":             "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/private.srs",
			"download_detour": "direct",
		},
		map[string]interface{}{
			"tag":             "geoip-private",
			"type":            "remote",
			"format":          "binary",
			"url":             "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/private.srs",
			"download_detour": "direct",
		},
	}
}

func (j *JsonService) addMux(jsonConfig *map[string]interface{}) error {
	enabled, err := j.SettingService.GetSubJsonMux()
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	outbounds, ok := jsonConfigOutbounds(jsonConfig)
	if !ok {
		return nil
	}
	for _, outbound := range *outbounds {
		protocol, _ := outbound["type"].(string)
		if supportsJSONMux(protocol) {
			outbound["multiplex"] = map[string]interface{}{
				"enabled":  true,
				"protocol": "smux",
			}
		}
	}
	return nil
}

func (j *JsonService) addNoises(jsonConfig *map[string]interface{}) error {
	noisesStr, err := j.SettingService.GetSubJsonNoises()
	if err != nil {
		return err
	}
	if strings.TrimSpace(noisesStr) == "" {
		return nil
	}
	var noises []interface{}
	if err := json.Unmarshal([]byte(noisesStr), &noises); err != nil {
		return err
	}
	outbounds, ok := jsonConfigOutbounds(jsonConfig)
	if !ok {
		return nil
	}
	for _, outbound := range *outbounds {
		protocol, _ := outbound["type"].(string)
		if supportsJSONNoises(protocol) {
			outbound["noises"] = noises
		}
	}
	return nil
}

func (j *JsonService) addFragment(jsonConfig *map[string]interface{}) error {
	fragmentStr, err := j.SettingService.GetSubJsonFragment()
	if err != nil {
		return err
	}
	if strings.TrimSpace(fragmentStr) == "" {
		return nil
	}
	var fragment map[string]interface{}
	if err := json.Unmarshal([]byte(fragmentStr), &fragment); err != nil {
		return err
	}
	outbounds, ok := jsonConfigOutbounds(jsonConfig)
	if !ok {
		return nil
	}
	for _, outbound := range *outbounds {
		protocol, _ := outbound["type"].(string)
		if supportsJSONFragment(protocol) {
			outbound["fragment"] = fragment
		}
	}
	return nil
}

func jsonConfigOutbounds(jsonConfig *map[string]interface{}) (*[]map[string]interface{}, bool) {
	switch outbounds := (*jsonConfig)["outbounds"].(type) {
	case *[]map[string]interface{}:
		return outbounds, true
	case []map[string]interface{}:
		return &outbounds, true
	default:
		return nil, false
	}
}

func supportsJSONMux(protocol string) bool {
	switch protocol {
	case "vless", "vmess", "trojan", "shadowsocks":
		return true
	default:
		return false
	}
}

func supportsJSONNoises(protocol string) bool {
	switch protocol {
	case "vless", "vmess", "trojan":
		return true
	default:
		return false
	}
}

func supportsJSONFragment(protocol string) bool {
	switch protocol {
	case "vless", "vmess", "trojan":
		return true
	default:
		return false
	}
}

func (j *JsonService) pushMixed(outbounds *[]map[string]interface{}, outTags *[]string, out map[string]interface{}) {
	socksOut := make(map[string]interface{}, 1)
	httpOut := make(map[string]interface{}, 1)
	for key, value := range out {
		socksOut[key] = value
		httpOut[key] = value
	}
	socksTag := fmt.Sprintf("%s-socks", out["tag"])
	httpTag := fmt.Sprintf("%s-http", out["tag"])
	socksOut["type"] = "socks"
	httpOut["type"] = "http"
	socksOut["tag"] = socksTag
	httpOut["tag"] = httpTag
	*outbounds = append(*outbounds, socksOut, httpOut)
	*outTags = append(*outTags, socksTag, httpTag)
}
