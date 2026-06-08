package importxui

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Dialect3XUIMHSanaei struct{}

func (Dialect3XUIMHSanaei) Name() string {
	return "dialect_3xui_mhsanaei"
}

func (Dialect3XUIMHSanaei) Detect(db *sql.DB) (bool, error) {
	for _, table := range []string{"inbounds", "client_traffics"} {
		var count int64
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&count); err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (Dialect3XUIMHSanaei) ReadInbounds(db *sql.DB) ([]xuiInboundRow, error) {
	present, err := tableColumns(db, "inbounds")
	if err != nil {
		return nil, err
	}
	// Project a fixed column order, defaulting any column a given 3x-ui fork
	// does not define. Vanilla mhsanaei lacks all_time/traffic_reset/
	// last_traffic_reset_time; normalized forks add node_id and keep these.
	projection := selectColumns(present, []columnSpec{
		{"id", "0"},
		{"user_id", "0"},
		{"up", "0"},
		{"down", "0"},
		{"total", "0"},
		{"all_time", "0"},
		{"remark", "''"},
		{"enable", "1"},
		{"expiry_time", "0"},
		{"traffic_reset", "''"},
		{"last_traffic_reset_time", "0"},
		{"listen", "''"},
		{"port", "0"},
		{"protocol", "''"},
		{"settings", "NULL"},
		{"stream_settings", "NULL"},
		{"tag", "''"},
		{"sniffing", "NULL"},
	})
	// #nosec G202 -- projection is assembled solely from the fixed column
	// allow-list above (quoted via quoteIdent) and literal defaults; no
	// external input reaches the SQL string.
	rows, err := db.Query("SELECT " + projection + " FROM inbounds ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []xuiInboundRow
	for rows.Next() {
		var row xuiInboundRow
		var remark, trafficReset, listen, protocol, settings, streamSettings, tag, sniffing sql.NullString
		var enable sql.NullInt64
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Up, &row.Down, &row.Total, &row.AllTime,
			&remark, &enable, &row.ExpiryTime, &trafficReset, &row.LastTrafficResetTime,
			&listen, &row.Port, &protocol, &settings, &streamSettings, &tag, &sniffing,
		); err != nil {
			return nil, err
		}
		row.Remark = nullString(remark)
		row.Enable = !enable.Valid || enable.Int64 != 0
		row.TrafficReset = nullString(trafficReset)
		row.Listen = nullString(listen)
		row.Protocol = strings.ToLower(strings.TrimSpace(nullString(protocol)))
		row.Settings = nullJSON(settings)
		row.StreamSettings = nullJSON(streamSettings)
		row.Tag = strings.TrimSpace(nullString(tag))
		row.Sniffing = nullJSON(sniffing)
		if row.Tag == "" {
			row.Tag = fmt.Sprintf("inbound-%d", row.Port)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (Dialect3XUIMHSanaei) ReadClients(db *sql.DB) ([]xuiClientTraffic, error) {
	present, err := tableColumns(db, "client_traffics")
	if err != nil {
		return nil, err
	}
	// Vanilla mhsanaei has neither all_time nor last_online; default both so
	// the positional Scan below stays valid on every fork.
	projection := selectColumns(present, []columnSpec{
		{"id", "0"},
		{"inbound_id", "0"},
		{"enable", "1"},
		{"email", "''"},
		{"up", "0"},
		{"down", "0"},
		{"all_time", "0"},
		{"expiry_time", "0"},
		{"total", "0"},
		{"reset", "0"},
		{"last_online", "0"},
	})
	// #nosec G202 -- projection is assembled solely from the fixed column
	// allow-list above (quoted via quoteIdent) and literal defaults; no
	// external input reaches the SQL string.
	rows, err := db.Query("SELECT " + projection + " FROM client_traffics ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []xuiClientTraffic
	for rows.Next() {
		var row xuiClientTraffic
		var enable sql.NullInt64
		var email sql.NullString
		if err := rows.Scan(
			&row.ID, &row.InboundID, &enable, &email, &row.Up, &row.Down,
			&row.AllTime, &row.ExpiryTime, &row.Total, &row.Reset, &row.LastOnline,
		); err != nil {
			return nil, err
		}
		row.Enable = !enable.Valid || enable.Int64 != 0
		row.Email = strings.TrimSpace(nullString(email))
		result = append(result, row)
	}
	return result, rows.Err()
}

func (Dialect3XUIMHSanaei) ReadSettings(db *sql.DB) ([]xuiSetting, error) {
	exists, err := tableExistsSQL(db, "settings")
	if err != nil || !exists {
		return nil, err
	}
	rows, err := db.Query("SELECT id, key, value FROM settings ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []xuiSetting
	for rows.Next() {
		var row xuiSetting
		var key, value sql.NullString
		if err := rows.Scan(&row.ID, &key, &value); err != nil {
			return nil, err
		}
		row.Key = strings.TrimSpace(nullString(key))
		row.Value = nullString(value)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (Dialect3XUIMHSanaei) ReadUsers(db *sql.DB) ([]xuiUser, error) {
	exists, err := tableExistsSQL(db, "users")
	if err != nil || !exists {
		return nil, err
	}
	rows, err := db.Query("SELECT id, username, password FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []xuiUser
	for rows.Next() {
		var row xuiUser
		var username, password sql.NullString
		if err := rows.Scan(&row.ID, &username, &password); err != nil {
			return nil, err
		}
		row.Username = strings.TrimSpace(nullString(username))
		row.Password = nullString(password)
		if row.Username != "" {
			result = append(result, row)
		}
	}
	return result, rows.Err()
}

func (Dialect3XUIMHSanaei) ReadOutboundTraffics(db *sql.DB) ([]xuiOutboundTraffic, error) {
	exists, err := tableExistsSQL(db, "outbound_traffics")
	if err != nil || !exists {
		return nil, err
	}
	rows, err := db.Query("SELECT id, tag, up, down FROM outbound_traffics ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []xuiOutboundTraffic
	for rows.Next() {
		var row xuiOutboundTraffic
		var tag sql.NullString
		if err := rows.Scan(&row.ID, &tag, &row.Up, &row.Down); err != nil {
			return nil, err
		}
		row.Tag = strings.TrimSpace(nullString(tag))
		result = append(result, row)
	}
	return result, rows.Err()
}

func (d Dialect3XUIMHSanaei) ReadXrayConfig(db *sql.DB) (string, error) {
	settings, err := d.ReadSettings(db)
	if err != nil {
		return "", err
	}
	for _, setting := range settings {
		if setting.Key == "xrayConfig" || setting.Key == "xrayTemplateConfig" {
			var parsed any
			if json.Unmarshal([]byte(setting.Value), &parsed) == nil {
				return setting.Value, nil
			}
		}
	}
	return "", nil
}

func tableExistsSQL(db *sql.DB, table string) (bool, error) {
	var count int64
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
