package database

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const backupBenchInsertChunk = 10000

func TestSafeSQLiteBatchSizeClientVariableBudgetPhase5(t *testing.T) {
	initBackupBenchLiveDB(t, filepath.Join(t.TempDir(), "s-ui.db"))

	batch := SafeSQLiteBatchSize(GetDB(), &model.Client{})
	columns := countModelColumns(GetDB(), &model.Client{})
	if columns <= 0 {
		t.Fatal("model.Client column count must be positive")
	}
	if batch < 1 {
		t.Fatalf("SafeSQLiteBatchSize returned %d", batch)
	}
	if got := batch * columns; got > sqliteVariableBudget {
		t.Fatalf("batch exceeds SQLite variable budget: batch=%d columns=%d placeholders=%d budget=%d", batch, columns, got, sqliteVariableBudget)
	}

	rows := make([]model.Client, batch)
	for i := range rows {
		rows[i] = maxBackupBenchClient(i)
	}
	if err := GetDB().CreateInBatches(&rows, batch).Error; err != nil {
		t.Fatalf("CreateInBatches with safe batch failed: %v", err)
	}
}

func BenchmarkBackupGetDb(b *testing.B) {
	runBackupGetDbBench(b, 100_000)
}

func BenchmarkBackupLargeGetDb(b *testing.B) {
	runBackupGetDbBench(b, 1_000_000)
}

func runBackupGetDbBench(b *testing.B, rowsPerTable int) {
	fixture := prepareBackupBenchFixture(b, rowsPerTable)
	for _, exclude := range []string{"", "stats,client_ips,audit_events,changes"} {
		exclude := exclude
		name := "full"
		if exclude != "" {
			name = "exclude_heavy"
		}
		b.Run(fmt.Sprintf("rows_%d/%s", rowsPerTable, name), func(b *testing.B) {
			livePath := copyBackupBenchFixtureToLiveDB(b, fixture)
			initBackupBenchLiveDB(b, livePath)
			b.ReportMetric(float64(rowsPerTable), "rows/table")
			b.ReportMetric(3*float64(rowsPerTable), "heavy_rows")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, err := GetDb(exclude)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(out)), "bytes_out")
			}
		})
	}
}

func prepareBackupBenchFixture(b *testing.B, rowsPerTable int) string {
	b.Helper()
	path := filepath.Join(b.TempDir(), fmt.Sprintf("backup-fixture-%d.db", rowsPerTable))
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			b.Skip(err)
		}
		b.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	if err := db.Exec("PRAGMA journal_mode=OFF").Error; err != nil {
		b.Fatal(err)
	}
	if err := db.Exec("PRAGMA synchronous=OFF").Error; err != nil {
		b.Fatal(err)
	}
	tables := backupTables()
	models := make([]any, 0, len(tables))
	for _, table := range tables {
		models = append(models, table.model)
	}
	if err := db.AutoMigrate(models...); err != nil {
		b.Fatal(err)
	}
	if err := ensureNoTLSRowOn(db); err != nil {
		b.Fatal(err)
	}
	if err := seedBackupBenchSmallTables(db); err != nil {
		b.Fatal(err)
	}
	if err := seedBackupBenchHeavyTables(db, rowsPerTable); err != nil {
		b.Fatal(err)
	}
	if err := db.Exec("VACUUM").Error; err != nil {
		b.Fatal(err)
	}
	return path
}

func seedBackupBenchSmallTables(db *gorm.DB) error {
	now := time.Now().Unix()
	if err := db.Create(&[]model.Setting{
		{Key: "version", Value: "phase5-perf"},
		{Key: "config", Value: `{"dns":{},"route":{}}`},
		{Key: "installSalt", Value: "phase5-install-salt"},
	}).Error; err != nil {
		return err
	}
	if err := db.Create(&model.User{Username: "admin", Password: "hash"}).Error; err != nil {
		return err
	}
	clients := make([]model.Client, 1000)
	for i := range clients {
		clients[i] = maxBackupBenchClient(i)
		clients[i].LastOnline = now
	}
	return db.CreateInBatches(&clients, SafeSQLiteBatchSize(db, &model.Client{})).Error
}

func seedBackupBenchHeavyTables(db *gorm.DB, rows int) error {
	for start := 0; start < rows; start += backupBenchInsertChunk {
		limit := backupBenchInsertChunk
		if rows-start < limit {
			limit = rows - start
		}
		if err := db.Exec(`
			WITH RECURSIVE n(x) AS (
				SELECT 0
				UNION ALL
				SELECT x + 1 FROM n WHERE x + 1 < ?
			)
			INSERT INTO stats(date_time, resource, tag, direction, traffic)
			SELECT 1700000000 + ? + x, 'user', 'client-' || ((? + x) % 1000), ((? + x) % 2), ? + x
			FROM n
		`, limit, start, start, start, start).Error; err != nil {
			return err
		}
		if err := db.Exec(`
			WITH RECURSIVE n(x) AS (
				SELECT 0
				UNION ALL
				SELECT x + 1 FROM n WHERE x + 1 < ?
			)
			INSERT INTO client_ips(client_name, ip, ip_hash, first_seen, last_seen)
			SELECT 'client-' || ((? + x) % 1000), '', printf('%064x', ? + x), 1700000000 + ? + x, 1700000000 + ? + x
			FROM n
		`, limit, start, start, start, start).Error; err != nil {
			return err
		}
		if err := db.Exec(`
			WITH RECURSIVE n(x) AS (
				SELECT 0
				UNION ALL
				SELECT x + 1 FROM n WHERE x + 1 < ?
			)
			INSERT INTO changes(date_time, actor, key, action, obj)
			SELECT 1700000000 + ? + x, 'phase5', 'settings', 'set', CAST('{"i":' || (? + x) || '}' AS BLOB)
			FROM n
		`, limit, start, start).Error; err != nil {
			return err
		}
	}
	return nil
}

func maxBackupBenchClient(i int) model.Client {
	return model.Client{
		Enable:      true,
		Name:        fmt.Sprintf("client-%06d", i),
		SubSecret:   fmt.Sprintf("sub-secret-%06d", i),
		Config:      []byte(`{"vmess":{"uuid":"00000000-0000-4000-8000-000000000000","alterId":0},"vless":{"flow":"xtls-rprx-vision"},"trojan":{"password":"phase5"},"shadowsocks":{"password":"phase5"}}`),
		Inbounds:    []byte(`[1,2,3,4]`),
		Links:       []byte(`["vmess://phase5","vless://phase5","trojan://phase5"]`),
		Volume:      1 << 40,
		Expiry:      1893456000,
		Down:        int64(i * 2),
		Up:          int64(i),
		Desc:        strings.Repeat("phase5 backup bench ", 4),
		Group:       "phase5",
		LimitIP:     4,
		IPLimitMode: "enforce",
		LastIPCount: 4,
		DelayStart:  true,
		AutoReset:   true,
		ResetDays:   30,
		NextReset:   1893456000,
		TotalUp:     int64(i * 3),
		TotalDown:   int64(i * 4),
	}
}

func copyBackupBenchFixtureToLiveDB(b *testing.B, fixture string) string {
	b.Helper()
	livePath := filepath.Join(b.TempDir(), "s-ui.db")
	src, err := os.Open(fixture)
	if err != nil {
		b.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(livePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		b.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		b.Fatal(err)
	}
	return livePath
}

func initBackupBenchLiveDB(tb testing.TB, livePath string) {
	tb.Helper()
	closeLiveDB()
	tb.Setenv("SUI_DB_FOLDER", filepath.Dir(livePath))
	if err := InitDB(livePath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			tb.Skip(err)
		}
		tb.Fatal(err)
	}
	GetDB().Config.Logger = gormlogger.Discard
	tb.Cleanup(closeLiveDB)
}
