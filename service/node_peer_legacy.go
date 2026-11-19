package service

import "slices"

func legacyPeerProtocol(kind string) bool {
	return kind == "vless" || kind == "vmess" || kind == "trojan"
}

func legacyPeerInventory(inbounds []PeerInbound) []PeerInbound {
	result := slices.Clone(inbounds)
	for i, inbound := range result {
		if inbound.Link == nil {
			continue
		}
		link := *inbound.Link
		if !legacyPeerProtocol(inbound.Type) || (link.Transport != nil && link.Transport.Type != "tcp") ||
			(link.TLS != nil && link.TLS.Reality != nil) {
			result[i].Link = nil
			continue
		}
		link.Transport, link.Options = nil, nil
		result[i].Link = &link
	}
	return result
}
