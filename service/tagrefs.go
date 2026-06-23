package service

import (
	"sort"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

// TagReference describes one place in the stored configuration that points at
// a tag. Lazy references are resolved by the core on every use, so replacing
// the target under the same tag is safe; eager ones are captured once at
// adapter construction and would keep pointing at a closed adapter.
type TagReference struct {
	Kind    string
	Locator string
	Lazy    bool
}

// eagerTagReferences filters references that pin the target adapter at
// construction time. Only those force a full core restart on hot replace.
func eagerTagReferences(refs []TagReference) []TagReference {
	var eager []TagReference
	for _, ref := range refs {
		if !ref.Lazy {
			eager = append(eager, ref)
		}
	}
	return eager
}

// formatTagReferenceError renders an actionable save error enumerating every
// configuration site that still points at the tag. Deleting or renaming a
// referenced tag is blocked because the next core start would fail on the
// dangling reference (a total outage), so the admin must rewire the
// references first.
func formatTagReferenceError(entityKind string, tag string, refs []TagReference) error {
	locators := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, dup := seen[ref.Locator]; dup {
			continue
		}
		seen[ref.Locator] = struct{}{}
		locators = append(locators, ref.Locator)
	}
	sort.Strings(locators)
	hint := "remove the reference or point it to another " + entityKind + " first"
	if entityKind == "outbound" || entityKind == "endpoint" {
		hint = "remove the reference or point it to another outbound (e.g. direct) first"
	}
	return common.NewErrorf("%s %q is still referenced by: %s. %s",
		entityKind, tag, strings.Join(locators, ", "), hint)
}

// inboundTagReferences lists configuration sites that capture the inbound
// adapter at construction time (currently ssm-api services binding managed
// shadowsocks servers).
func inboundTagReferences(tx *gorm.DB, tag string) ([]TagReference, error) {
	rows, err := ssmServiceRows(tx)
	if err != nil {
		return nil, err
	}
	return scanServiceRowsForInboundTag(rows, tag), nil
}

// ssmCascadeServiceIds lists ssm-api services that must be recreated after the
// inbound they manage was hot-reloaded, so they re-bind to the fresh adapter.
func ssmCascadeServiceIds(tx *gorm.DB, inboundTag string) ([]uint, error) {
	rows, err := ssmServiceRows(tx)
	if err != nil {
		return nil, err
	}
	return ssmServiceIdsReferencingInbound(rows, inboundTag), nil
}

func ssmServiceRows(tx *gorm.DB) ([]model.Service, error) {
	var rows []model.Service
	err := tx.Model(model.Service{}).Select("id", "type", "tag", "options").
		Where("type = ?", "ssm-api").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// outboundTagReferences lists every configuration site that points at an
// outbound-namespace tag. Outbounds and endpoints share one tag namespace in
// the core's lookup, so both tables are scanned, together with service dial
// detours and the config blob. excludeOutboundId/excludeEndpointId drop the
// row being edited so its own options never count as a self-reference.
func outboundTagReferences(tx *gorm.DB, tag string, excludeOutboundId uint, excludeEndpointId uint) ([]TagReference, error) {
	var outbounds []model.Outbound
	if err := tx.Model(model.Outbound{}).Select("id", "type", "tag", "options").Find(&outbounds).Error; err != nil {
		return nil, err
	}
	refs := scanOutboundRowsForTag(outbounds, tag, excludeOutboundId)

	var endpoints []model.Endpoint
	if err := tx.Model(model.Endpoint{}).Select("id", "type", "tag", "options").Find(&endpoints).Error; err != nil {
		return nil, err
	}
	refs = append(refs, scanEndpointRowsForTag(endpoints, tag, excludeEndpointId)...)

	var services []model.Service
	if err := tx.Model(model.Service{}).Select("id", "type", "tag", "options").Find(&services).Error; err != nil {
		return nil, err
	}
	refs = append(refs, scanServiceRowsForOutboundDetour(services, tag)...)

	blob, err := configBlobFrom(tx)
	if err != nil {
		return nil, err
	}
	blobRefs, err := scanConfigBlobForOutboundTag(blob, tag)
	if err != nil {
		return nil, err
	}
	return append(refs, blobRefs...), nil
}

func configBlobFrom(tx *gorm.DB) ([]byte, error) {
	var stored model.Setting
	result := tx.Model(model.Setting{}).Where("key = ?", "config").Limit(1).Find(&stored)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return []byte(stored.Value), nil
}
