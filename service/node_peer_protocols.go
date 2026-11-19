package service

import "github.com/gofrs/uuid/v5"

func peerProtocolSupported(protocol string) bool {
	switch protocol {
	case "vless", "vmess", "trojan", "anytls", "hysteria", "hysteria2", "tuic":
		return true
	default:
		return false
	}
}

func validPeerProtocolCredential(user peerUser) bool {
	validPassword := user.Password != "" && len(user.Password) <= 256
	switch user.Type {
	case "vless", "vmess":
		return peerCredentialIdentity(user) != "" && user.Password == "" && user.AuthString == "" &&
			(user.Flow == "" || user.Type == "vless" && user.Flow == "xtls-rprx-vision")
	case "trojan", "anytls", "hysteria2":
		return validPassword && user.UUID == "" && user.Flow == "" && user.AuthString == ""
	case "tuic":
		return peerCredentialIdentity(user) != "" && validPassword && user.Flow == "" && user.AuthString == ""
	case "hysteria":
		return user.AuthString != "" && len(user.AuthString) <= 256 && user.UUID == "" && user.Flow == "" && user.Password == ""
	default:
		return false
	}
}

func peerCredentialIdentity(user peerUser) string {
	switch user.Type {
	case "vless", "vmess", "tuic":
		id, err := uuid.FromString(user.UUID)
		if err == nil && id != uuid.Nil {
			return id.String()
		}
	case "trojan", "anytls", "hysteria2":
		return user.Password
	case "hysteria":
		return user.AuthString
	}
	return ""
}

func peerInboundCredential(user peerUser, inbound map[string]any) map[string]string {
	entry := map[string]string{"name": user.Name}
	switch user.Type {
	case "trojan", "anytls", "hysteria2":
		entry["password"] = user.Password
	case "tuic":
		entry["uuid"], entry["password"] = peerCredentialIdentity(user), user.Password
	case "hysteria":
		entry["auth_str"] = user.AuthString
	case "vless", "vmess":
		entry["uuid"] = peerCredentialIdentity(user)
		transport, _ := inbound["transport"].(map[string]any)
		transportType, _ := transport["type"].(string)
		tls, _ := inbound["tls"].(map[string]any)
		if user.Flow != "" && tls["enabled"] == true && (transportType == "" || transportType == "tcp") {
			entry["flow"] = user.Flow
		}
	}
	return entry
}
