package importxui

import (
	"database/sql"
	"errors"
)

var ErrDialectUnknown = errors.New("xui_dialect_unknown")

type Dialect interface {
	Name() string
	Detect(db *sql.DB) (bool, error)
	ReadInbounds(db *sql.DB) ([]xuiInboundRow, error)
	ReadClients(db *sql.DB) ([]xuiClientTraffic, error)
	ReadSettings(db *sql.DB) ([]xuiSetting, error)
	ReadUsers(db *sql.DB) ([]xuiUser, error)
	ReadOutboundTraffics(db *sql.DB) ([]xuiOutboundTraffic, error)
	ReadXrayConfig(db *sql.DB) (string, error)
}

func RegisteredDialects() []Dialect {
	return []Dialect{Dialect3XUIMHSanaei{}}
}
