package sub

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x/service"
)

// subDisplaySettingsTTL bounds how long the per-process snapshot of the
// display/format subscription settings is reused before being re-read. It
// matches the existing sub rate-limit setting TTL; these are presentation
// settings, so a minute of staleness after an admin change is acceptable and it
// removes ~8 SELECTs from every subscription request.
const subDisplaySettingsTTL = time.Minute

// subDisplaySettings is the snapshot of the read-mostly display/format settings
// consulted on the subscription hot path. The security-relevant gates
// (subSecretRequired, subLinkEnable) are deliberately NOT cached here and stay
// read fresh on every request.
type subDisplaySettings struct {
	showInfo     bool
	nameInRemark bool
	updates      int
	title        string
	supportURL   string
	profileURL   string
	announce     string
	encode       bool
}

var subDisplaySettingsCache = struct {
	sync.Mutex
	value     subDisplaySettings
	expiresAt time.Time
}{}

// cachedSubDisplaySettings returns the display settings snapshot, reading it
// from the database at most once per subDisplaySettingsTTL. Each getter ignores
// its error exactly as the previous inline call sites did (zero value on error).
func cachedSubDisplaySettings(ss *service.SettingService, now time.Time) subDisplaySettings {
	subDisplaySettingsCache.Lock()
	defer subDisplaySettingsCache.Unlock()
	if now.Before(subDisplaySettingsCache.expiresAt) {
		return subDisplaySettingsCache.value
	}
	var v subDisplaySettings
	v.showInfo, _ = ss.GetSubShowInfo()
	v.nameInRemark, _ = ss.GetSubNameInRemark()
	v.updates, _ = ss.GetSubUpdates()
	v.title, _ = ss.GetSubTitle()
	v.supportURL, _ = ss.GetSubSupportUrl()
	v.profileURL, _ = ss.GetSubProfileUrl()
	v.announce, _ = ss.GetSubAnnounce()
	v.encode, _ = ss.GetSubEncode()
	subDisplaySettingsCache.value = v
	subDisplaySettingsCache.expiresAt = now.Add(subDisplaySettingsTTL)
	return v
}

func resetSubDisplaySettingsCacheForTest() {
	subDisplaySettingsCache.Lock()
	defer subDisplaySettingsCache.Unlock()
	subDisplaySettingsCache.value = subDisplaySettings{}
	subDisplaySettingsCache.expiresAt = time.Time{}
}
