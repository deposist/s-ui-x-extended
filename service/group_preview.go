package service

import (
	"encoding/json"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

// GroupPreviewMember is one resolved member of a group preview.
type GroupPreviewMember struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Source  string `json:"source"` // "static" or "provider:<tag>"
	Exists  bool   `json:"exists"`
	IsGroup bool   `json:"isGroup"`
}

// GroupPreview is the resolved member list for a group outbound, shown in the
// UI before saving. It lists static members with their existence and type
// (so the admin can see if a member is missing or is itself a group) and
// provider-backed members with their source provider tag.
type GroupPreview struct {
	Tag             string               `json:"tag"`
	GroupType       string               `json:"groupType"`
	StaticMembers   []GroupPreviewMember `json:"staticMembers"`
	ProviderMembers []GroupPreviewMember `json:"providerMembers"`
	UseAllProviders bool                 `json:"useAllProviders"`
}

// GroupPreview resolves the members of a group outbound from the database. It
// is read-only and never modifies the running core.
func (s *ConfigService) GroupPreview(tag string) (*GroupPreview, error) {
	db := database.GetDB()

	var row model.Outbound
	if err := db.Model(model.Outbound{}).Where("tag = ?", tag).First(&row).Error; err != nil {
		return nil, err
	}

	if !isGroupType(row.Type) {
		return nil, common.NewErrorf("outbound %q is not a group type", tag)
	}

	preview := &GroupPreview{
		Tag:       tag,
		GroupType: row.Type,
	}

	opts := optionsMapOf(row.Options)
	if opts == nil {
		return preview, nil
	}

	// Resolve static members.
	if members, _ := opts["outbounds"].([]any); len(members) > 0 {
		tagToType := loadOutboundTagTypes(db)
		for _, item := range members {
			mTag, _ := item.(string)
			if mTag == "" {
				continue
			}
			mType, exists := tagToType[mTag]
			preview.StaticMembers = append(preview.StaticMembers, GroupPreviewMember{
				Tag:     mTag,
				Type:    mType,
				Source:  "static",
				Exists:  exists,
				IsGroup: isGroupType(mType),
			})
		}
	}

	// Resolve provider references.
	if providers, _ := opts["providers"].([]any); len(providers) > 0 {
		providerTags := loadProviderTags(db)
		for _, item := range providers {
			pTag, _ := item.(string)
			if pTag == "" {
				continue
			}
			_, exists := providerTags[pTag]
			preview.ProviderMembers = append(preview.ProviderMembers, GroupPreviewMember{
				Tag:    pTag,
				Source: "provider:" + pTag,
				Exists: exists,
			})
		}
	}

	if useAll, _ := opts["use_all_providers"].(bool); useAll {
		preview.UseAllProviders = true
	}

	return preview, nil
}

func loadOutboundTagTypes(db *gorm.DB) map[string]string {
	var rows []model.Outbound
	if err := db.Model(model.Outbound{}).Select("tag", "type").Find(&rows).Error; err != nil {
		return nil
	}
	m := make(map[string]string, len(rows))
	for _, row := range rows {
		m[row.Tag] = row.Type
	}
	return m
}

func loadProviderTags(db *gorm.DB) map[string]struct{} {
	var rows []model.Provider
	if err := db.Model(model.Provider{}).Select("tag").Find(&rows).Error; err != nil {
		return nil
	}
	m := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		m[row.Tag] = struct{}{}
	}
	return m
}

// optionsMapOf is already defined in tagrefs_rows.go, but we keep the import
// for future use.
var _ = json.Unmarshal
