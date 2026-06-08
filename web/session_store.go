package web

import (
	"encoding/base32"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/service"
	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gorilla/securecookie"
	gsessions "github.com/gorilla/sessions"
	"gorm.io/gorm"
)

type SQLiteSessionStore struct {
	db        *gorm.DB
	codecs    []securecookie.Codec
	optionsMu sync.RWMutex
	options   *gsessions.Options
	now       func() time.Time
}

type sqliteSessionRow struct {
	ID        string `gorm:"column:id"`
	Data      []byte `gorm:"column:data"`
	ExpiresAt int64  `gorm:"column:expires_at"`
}

var sessionIDEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func NewSQLiteSessionStore(db *gorm.DB, keyPairs ...[]byte) (*SQLiteSessionStore, error) {
	if db == nil {
		return nil, errors.New("sqlite session store requires an initialized database")
	}
	codecs := codecsFromHashKeys(keyPairs...)
	if len(codecs) == 0 {
		return nil, errors.New("sqlite session store requires at least one non-empty cookie key")
	}
	store := &SQLiteSessionStore{
		db:     db,
		codecs: codecs,
		options: &gsessions.Options{
			Path:     "/",
			MaxAge:   86400 * 30,
			SameSite: http.SameSiteLaxMode,
			HttpOnly: true,
			// Secure-by-default so any future code path that creates a session
			// without calling Options() still gets a Secure cookie. Production
			// login/CSRF flows override this via resolveCookieSecure().
			Secure: true,
		},
		now: time.Now,
	}
	if err := store.ensureSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteSessionStore) Options(options ginsessions.Options) {
	next := cloneSessionOptions(options.ToGorillaOptions())
	s.optionsMu.Lock()
	s.options = next
	s.optionsMu.Unlock()
}

func (s *SQLiteSessionStore) Get(r *http.Request, name string) (*gsessions.Session, error) {
	return gsessions.GetRegistry(r).Get(s, name)
}

func (s *SQLiteSessionStore) New(r *http.Request, name string) (*gsessions.Session, error) {
	session := gsessions.NewSession(s, name)
	session.Options = s.currentOptions()
	session.IsNew = true

	cookie, err := r.Cookie(name)
	if err != nil {
		return session, nil
	}
	if err := securecookie.DecodeMulti(name, cookie.Value, &session.ID, s.codecs...); err != nil {
		return session, err
	}
	loaded, err := s.load(session)
	if err != nil {
		return session, err
	}
	session.IsNew = !loaded
	if !loaded {
		session.ID = ""
	}
	return session, nil
}

func (s *SQLiteSessionStore) Save(_ *http.Request, w http.ResponseWriter, session *gsessions.Session) error {
	if session.Options.MaxAge < 0 {
		if session.ID != "" {
			if err := s.erase(session.ID); err != nil {
				return err
			}
		}
		http.SetCookie(w, gsessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	// Regenerate the session ID when the login flow requests it: erase the old
	// row and clear the ID so a fresh one is minted below. This defeats session
	// fixation — a pre-auth (CSRF) session cookie cannot survive authentication
	// under the same ID. The marker must not persist into the stored session data.
	if _, regenerate := session.Values[service.SessionRegenerateKey]; regenerate {
		delete(session.Values, service.SessionRegenerateKey)
		if session.ID != "" {
			if err := s.erase(session.ID); err != nil {
				return err
			}
			session.ID = ""
		}
	}

	if session.ID == "" {
		session.ID = sessionIDEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
	}
	if err := s.save(session); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, gsessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

func (s *SQLiteSessionStore) ensureSchema() error {
	if err := s.liveDB().Exec(`
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	data BLOB NOT NULL,
	expires_at INTEGER NOT NULL DEFAULT 0
)`).Error; err != nil {
		return err
	}
	return s.liveDB().Exec("CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)").Error
}

func (s *SQLiteSessionStore) save(session *gsessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.codecs...)
	if err != nil {
		return err
	}
	expiresAt := int64(0)
	if session.Options.MaxAge > 0 {
		expiresAt = s.now().Add(time.Duration(session.Options.MaxAge) * time.Second).Unix()
	}
	return s.liveDB().Exec(`
INSERT INTO sessions(id, data, expires_at)
VALUES(?, ?, ?)
ON CONFLICT(id) DO UPDATE SET data = excluded.data, expires_at = excluded.expires_at
`, session.ID, []byte(encoded), expiresAt).Error
}

func (s *SQLiteSessionStore) load(session *gsessions.Session) (bool, error) {
	var row sqliteSessionRow
	err := s.liveDB().Table("sessions").Where("id = ?", session.ID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if row.ExpiresAt > 0 && row.ExpiresAt <= s.now().Unix() {
		if err := s.erase(row.ID); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := securecookie.DecodeMulti(session.Name(), string(row.Data), &session.Values, s.codecs...); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLiteSessionStore) erase(id string) error {
	return s.liveDB().Exec("DELETE FROM sessions WHERE id = ?", id).Error
}

func (s *SQLiteSessionStore) liveDB() *gorm.DB {
	if live := database.GetDB(); live != nil {
		return live
	}
	return s.db
}

func (s *SQLiteSessionStore) currentOptions() *gsessions.Options {
	s.optionsMu.RLock()
	defer s.optionsMu.RUnlock()
	return cloneSessionOptions(s.options)
}

func cloneSessionOptions(options *gsessions.Options) *gsessions.Options {
	if options == nil {
		return nil
	}
	clone := *options
	return &clone
}

func codecsFromHashKeys(keys ...[]byte) []securecookie.Codec {
	codecs := make([]securecookie.Codec, 0, len(keys))
	for _, key := range keys {
		if len(key) == 0 {
			continue
		}
		codecs = append(codecs, securecookie.New(key, nil))
	}
	return codecs
}
