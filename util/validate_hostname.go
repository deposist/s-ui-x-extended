package util

import (
	"strings"

	"github.com/deposist/s-ui-x/util/common"
	"golang.org/x/net/idna"
)

func ValidateHostname(host string) error {
	if host == "" {
		return nil
	}
	if strings.TrimSpace(host) != host {
		return common.NewError("invalid hostname")
	}
	if strings.ContainsAny(host, "/\\:@?#[]") {
		return common.NewError("invalid hostname")
	}

	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return common.NewError("invalid hostname")
	}
	if ascii == "" || len(ascii) > 253 {
		return common.NewError("invalid hostname")
	}

	labels := strings.Split(ascii, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return common.NewError("invalid hostname")
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return common.NewError("invalid hostname")
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return common.NewError("invalid hostname")
		}
	}
	return nil
}
