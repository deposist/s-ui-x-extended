package importxui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func BenchmarkPlan(b *testing.B) {
	for _, inbounds := range []int{100, 1000} {
		inbounds := inbounds
		b.Run(fmt.Sprintf("wireguard_inbounds_%d", inbounds), func(b *testing.B) {
			initImportXUIPerfMainDB(b)
			src := createImportXUIPerfSource(b, inbounds)
			b.ReportMetric(float64(inbounds), "inbounds")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
				if err != nil {
					b.Fatal(err)
				}
				if len(plan.Items) != inbounds {
					b.Fatalf("plan items=%d want %d", len(plan.Items), inbounds)
				}
			}
		})
	}
}

func BenchmarkApply(b *testing.B) {
	for _, inbounds := range []int{100, 1000} {
		inbounds := inbounds
		b.Run(fmt.Sprintf("wireguard_inbounds_%d/dry_run", inbounds), func(b *testing.B) {
			initImportXUIPerfMainDB(b)
			src := createImportXUIPerfSource(b, inbounds)
			plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(inbounds), "inbounds")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report, err := Apply(src, *plan, ApplyOptions{DryRun: true, SkipAudit: true})
				if err != nil {
					b.Fatal(err)
				}
				if report.Summary.Endpoints.Imported != inbounds {
					b.Fatalf("dry-run imported endpoints=%d want %d", report.Summary.Endpoints.Imported, inbounds)
				}
			}
		})
		b.Run(fmt.Sprintf("wireguard_inbounds_%d/real", inbounds), func(b *testing.B) {
			initImportXUIPerfMainDB(b)
			src := createImportXUIPerfSource(b, inbounds)
			plan, err := Plan(src, PlanOptions{Strategy: StrategyMerge})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(inbounds), "inbounds")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				initImportXUIPerfMainDB(b)
				b.StartTimer()
				report, err := Apply(src, *plan, ApplyOptions{
					SkipAudit: true,
					Now: func() int64 {
						return 1700000000 + int64(i)
					},
				})
				if err != nil {
					b.Fatal(err)
				}
				if report.Summary.Endpoints.Imported != inbounds {
					b.Fatalf("real imported endpoints=%d want %d", report.Summary.Endpoints.Imported, inbounds)
				}
			}
		})
	}
}

func initImportXUIPerfMainDB(tb testing.TB) {
	tb.Helper()
	if db := database.GetDB(); db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
			time.Sleep(25 * time.Millisecond)
		}
	}
	dir := makeImportXUITempDir(tb)
	tb.Setenv("SUI_DB_FOLDER", dir)
	if err := database.InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			tb.Skip(err)
		}
		tb.Fatal(err)
	}
	database.GetDB().Config.Logger = gormlogger.Discard
	tb.Cleanup(func() {
		if db := database.GetDB(); db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})
}

func createImportXUIPerfSource(tb testing.TB, inbounds int) string {
	tb.Helper()
	path := filepath.Join(makeImportXUITempDir(tb), "x-ui.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		tb.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	if err := db.Exec(`CREATE TABLE inbounds (
		id integer primary key,
		user_id integer,
		up integer,
		down integer,
		total integer,
		all_time integer,
		remark text,
		enable integer,
		expiry_time integer,
		traffic_reset text,
		last_traffic_reset_time integer,
		listen text,
		port integer,
		protocol text,
		settings text,
		stream_settings text,
		tag text,
		sniffing text
	)`).Error; err != nil {
		tb.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE client_traffics (
		id integer primary key,
		inbound_id integer,
		enable integer,
		email text,
		up integer,
		down integer,
		all_time integer,
		expiry_time integer,
		total integer,
		reset integer,
		last_online integer
	)`).Error; err != nil {
		tb.Fatal(err)
	}
	tx := db.Begin()
	for i := 0; i < inbounds; i++ {
		settings := fmt.Sprintf(`{"mtu":1280,"secretKey":"private-key-%d","peers":[{"publicKey":"public-key-%d","allowedIPs":["0.0.0.0/0"],"keepAlive":25}]}`, i, i)
		tag := fmt.Sprintf("wg-phase5-%04d", i)
		if err := tx.Exec(`INSERT INTO inbounds
			(id, user_id, up, down, total, all_time, remark, enable, expiry_time, traffic_reset,
			 last_traffic_reset_time, listen, port, protocol, settings, stream_settings, tag, sniffing)
			VALUES (?, 1, 0, 0, 0, 0, ?, 1, 0, '', 0, '', ?, 'wireguard', ?, '{}', ?, '{}')`,
			i+1, tag, 30000+i, settings, tag,
		).Error; err != nil {
			_ = tx.Rollback().Error
			tb.Fatal(err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		tb.Fatal(err)
	}
	return path
}
