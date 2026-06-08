package service

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/deposist/s-ui-x/core"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	gormlogger "gorm.io/gorm/logger"
)

func BenchmarkStatsService_SaveStats(b *testing.B) {
	for _, clients := range []int{100, 1000} {
		clients := clients
		b.Run(fmt.Sprintf("clients_%d", clients), func(b *testing.B) {
			initServicePerfDB(b)
			seedStatsBenchClients(b, clients)
			tracker := core.NewStatsTracker()
			statsService := &StatsService{Runtime: NewRuntime(syntheticStatsCoreForBench(b, tracker))}
			b.ReportMetric(float64(clients), "clients")
			b.ReportMetric(float64(clients*2), "stats/input")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				seedSyntheticUserStatsForBench(b, tracker, clients)
				b.StartTimer()
				if err := statsService.SaveStats(true); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUpdateClientTrafficDeltas(b *testing.B) {
	for _, emptyPercent := range []int{0, 50, 90} {
		emptyPercent := emptyPercent
		b.Run(fmt.Sprintf("clients_1000_empty_%d_pct", emptyPercent), func(b *testing.B) {
			initServicePerfDB(b)
			const clients = 1000
			seedStatsBenchClients(b, clients)
			deltas := make(map[string]clientTrafficDelta, clients)
			for i := 0; i < clients; i++ {
				name := fmt.Sprintf("user-%04d", i)
				if i%100 < emptyPercent {
					deltas[name] = clientTrafficDelta{}
					continue
				}
				deltas[name] = clientTrafficDelta{up: int64(i + 1), down: int64(i + 2)}
			}
			b.ReportMetric(float64(emptyPercent), "empty_pct")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tx := database.GetDB().Begin()
				if tx.Error != nil {
					b.Fatal(tx.Error)
				}
				if err := updateClientTrafficDeltas(tx, deltas); err != nil {
					_ = tx.Rollback().Error
					b.Fatal(err)
				}
				if err := tx.Rollback().Error; err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func seedStatsBenchClients(tb testing.TB, n int) {
	tb.Helper()
	clients := make([]model.Client, n)
	for i := range clients {
		clients[i] = model.Client{
			Enable:      true,
			Name:        fmt.Sprintf("user-%04d", i),
			Config:      []byte(`{}`),
			Inbounds:    []byte(`[]`),
			Links:       []byte(`[]`),
			IPLimitMode: "monitor",
		}
	}
	if err := database.GetDB().CreateInBatches(&clients, database.SafeSQLiteBatchSize(database.GetDB(), &model.Client{})).Error; err != nil {
		tb.Fatal(err)
	}
}

func syntheticStatsCoreForBench(tb testing.TB, tracker *core.StatsTracker) *core.Core {
	tb.Helper()
	box := &core.Box{}
	setUnexportedFieldForBench(reflect.ValueOf(box).Elem().FieldByName("statsTracker"), reflect.ValueOf(tracker))
	coreInstance := core.NewCore()
	setUnexportedFieldForBench(reflect.ValueOf(coreInstance).Elem().FieldByName("isRunning"), reflect.ValueOf(true))
	setUnexportedFieldForBench(reflect.ValueOf(coreInstance).Elem().FieldByName("instance"), reflect.ValueOf(box))
	return coreInstance
}

func seedSyntheticUserStatsForBench(tb testing.TB, tracker *core.StatsTracker, n int) {
	tb.Helper()
	trackerValue := reflect.ValueOf(tracker).Elem()
	usersField := trackerValue.FieldByName("users")
	users := reflect.MakeMapWithSize(usersField.Type(), n)
	counterType := usersField.Type().Elem()
	for i := 0; i < n; i++ {
		counter := reflect.New(counterType).Elem()
		read := &atomic.Int64{}
		write := &atomic.Int64{}
		read.Store(int64(i + 1))
		write.Store(int64(i + 2))
		setUnexportedFieldForBench(counter.FieldByName("read"), reflect.ValueOf(read))
		setUnexportedFieldForBench(counter.FieldByName("write"), reflect.ValueOf(write))
		users.SetMapIndex(reflect.ValueOf(fmt.Sprintf("user-%04d", i)), counter)
	}
	setUnexportedFieldForBench(usersField, users)

	inboundsField := trackerValue.FieldByName("inbounds")
	setUnexportedFieldForBench(inboundsField, reflect.MakeMap(inboundsField.Type()))
	outboundsField := trackerValue.FieldByName("outbounds")
	setUnexportedFieldForBench(outboundsField, reflect.MakeMap(outboundsField.Type()))
}

func setUnexportedFieldForBench(field reflect.Value, value reflect.Value) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(value)
}

func initServicePerfDB(tb testing.TB) {
	tb.Helper()
	if db := database.GetDB(); db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
			time.Sleep(25 * time.Millisecond)
		}
	}
	dir := tb.TempDir()
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
