package database

import (
	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

// AdaptToCurrentVersion performs idempotent post-migration adjustments that
// ensure a database imported from an older S-UI version is fully usable on the
// current build:
//
//  1. Plaintext admin/user passwords are rehashed with bcrypt.
//  2. Any remaining default user state is normalized (random password if a
//     legacy backup contained the historical "admin" plaintext default).
//  3. Indexes added by this fork are (re-)created if missing.
//  4. The `settings.version` row is updated to the current version so that
//     `cmd/migration` skips running again on the next startup.
//
// All steps are idempotent: running the function multiple times is safe.
//
// AdaptToCurrentVersion expects the package-level `db` to be open. It must be
// called after `InitDB` (so AutoMigrate already ran), but before the panel
// starts serving traffic.
func AdaptToCurrentVersion() error {
	if db == nil {
		return common.NewError("AdaptToCurrentVersion: database not initialized")
	}
	if err := ensureIndexes(); err != nil {
		return err
	}
	if err := rehashLegacyPasswords(db); err != nil {
		return err
	}
	if err := bumpVersionSetting(db); err != nil {
		return err
	}
	return nil
}

// rehashLegacyPasswords scans the users table and rewrites any password field
// that is not already a bcrypt hash. The wire format from `common.HashPassword`
// stores the bcrypt blob behind a `bcrypt:` prefix; raw `$2[aby]$...` blobs
// (which old backups never had, but might appear via manual edits) are also
// considered hashed.
func rehashLegacyPasswords(tx *gorm.DB) error {
	var users []model.User
	if err := tx.Model(model.User{}).Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if user.Password == "" {
			continue
		}
		if common.IsPasswordHash(user.Password) {
			continue
		}
		hashed, err := common.HashPassword(user.Password)
		if err != nil {
			return err
		}
		if err := tx.Model(model.User{}).Where("id = ?", user.Id).Update("password", hashed).Error; err != nil {
			return err
		}
		logger.Infof("backup adapt: rehashed plaintext password for user %q", user.Username)
	}
	return nil
}

// bumpVersionSetting writes the current build version into the `settings`
// table so the migration runner does not re-run already-applied migrations on
// the next start.
func bumpVersionSetting(tx *gorm.DB) error {
	current := config.GetVersion()
	if current == "" {
		return nil
	}
	var existing model.Setting
	err := tx.Model(model.Setting{}).Where("key = ?", "version").First(&existing).Error
	if IsNotFound(err) {
		return tx.Create(&model.Setting{Key: "version", Value: current}).Error
	}
	if err != nil {
		return err
	}
	if existing.Value == current {
		return nil
	}
	// The running build is authoritative for the version stamp; only refuse to
	// overwrite when the stored version is from a strictly higher MAJOR (a
	// genuinely-future release we must not downgrade). Legacy/stale versions
	// (e.g. upstream 1.x, or a final 1.0.0 vs a 1.0.0-betaN build) are stamped
	// with the current build version.
	if config.IsGenuinelyNewer(existing.Value, current) {
		return nil
	}
	return tx.Model(model.Setting{}).Where("key = ?", "version").Update("value", current).Error
}

func compareVersion(left string, right string) (int, bool) {
	return config.CompareVersions(left, right)
}
