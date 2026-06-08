package paidsub

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// maxQRInputBytes bounds the QR payload to keep rendering cheap and block CPU
// abuse. Subscription URLs and share links are well under this.
const maxQRInputBytes = 1500

// renderQR encodes content into a PNG QR code. The size is fixed; input length
// is capped.
func renderQR(content string) ([]byte, error) {
	if content == "" {
		return nil, fmt.Errorf("empty qr content")
	}
	if len(content) > maxQRInputBytes {
		return nil, fmt.Errorf("qr content too long")
	}
	return qrcode.Encode(content, qrcode.Medium, 320)
}
