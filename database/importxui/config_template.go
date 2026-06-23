package importxui

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const deterministicSalt = "s-ui:xui-import:v1:"

func deterministicClientConfig(email string) map[string]map[string]any {
	mixedPassword := deterministicSeq(email+":mixed", 10)
	ssPassword16 := deterministicBytesBase64(email+":ss16", 16)
	ssPassword32 := deterministicBytesBase64(email+":ss32", 32)
	id := deterministicUUID(email + ":uuid")
	return map[string]map[string]any{
		"mixed": {
			"username": email,
			"password": mixedPassword,
		},
		"socks": {
			"username": email,
			"password": mixedPassword,
		},
		"http": {
			"username": email,
			"password": mixedPassword,
		},
		"shadowsocks": {
			"name":     email,
			"password": ssPassword32,
		},
		"shadowsocks16": {
			"name":     email,
			"password": ssPassword16,
		},
		"shadowtls": {
			"name":     email,
			"password": ssPassword32,
		},
		"vmess": {
			"name":    email,
			"uuid":    id,
			"alterId": 0,
		},
		"vless": {
			"name": email,
			"uuid": id,
			"flow": "xtls-rprx-vision",
		},
		"anytls": {
			"name":     email,
			"password": mixedPassword,
		},
		"trojan": {
			"name":     email,
			"password": mixedPassword,
		},
		"naive": {
			"username": email,
			"password": mixedPassword,
		},
		"hysteria": {
			"name":     email,
			"auth_str": mixedPassword,
		},
		"tuic": {
			"name":     email,
			"uuid":     id,
			"password": mixedPassword,
		},
		"hysteria2": {
			"name":     email,
			"password": mixedPassword,
		},
	}
}

func deterministicSeq(seed string, count int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buf := deterministicBytes(seed, count)
	out := make([]byte, count)
	for i := range out {
		out[i] = chars[int(buf[i])%len(chars)]
	}
	return string(out)
}

func deterministicBytesBase64(seed string, count int) string {
	return base64.StdEncoding.EncodeToString(deterministicBytes(seed, count))
}

func deterministicBytes(seed string, count int) []byte {
	out := make([]byte, 0, count)
	counter := 0
	for len(out) < count {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s%s:%d", deterministicSalt, seed, counter)))
		out = append(out, sum[:]...)
		counter++
	}
	return out[:count]
}

func deterministicUUID(seed string) string {
	b := deterministicBytes(seed, 16)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
}
