package ipmonitor

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"

	gormlogger "gorm.io/gorm/logger"
)

func BenchmarkLoadCacheEntry(b *testing.B) {
	for _, cfg := range []struct {
		clients      int
		ipsPerClient int
	}{
		{clients: 100, ipsPerClient: 5},
		{clients: 1000, ipsPerClient: 5},
	} {
		cfg := cfg
		b.Run(fmt.Sprintf("clients_%d_ips_%d", cfg.clients, cfg.ipsPerClient), func(b *testing.B) {
			initIPMonitorPerfDB(b)
			names := seedIPMonitorPerfClients(b, cfg.clients, cfg.ipsPerClient)
			now := time.Now()
			b.ReportMetric(float64(cfg.clients), "clients")
			b.ReportMetric(float64(cfg.ipsPerClient), "ips/client")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				entry, ok := loadCacheEntry(names[i%len(names)], now)
				if !ok {
					b.Fatal("loadCacheEntry returned no cache entry")
				}
				if len(entry.ips) != cfg.ipsPerClient {
					b.Fatalf("loaded ips=%d want %d", len(entry.ips), cfg.ipsPerClient)
				}
			}
		})
	}
}

func BenchmarkAllow(b *testing.B) {
	for _, cfg := range []struct {
		clients      int
		ipsPerClient int
	}{
		{clients: 100, ipsPerClient: 5},
		{clients: 1000, ipsPerClient: 5},
	} {
		cfg := cfg
		b.Run(fmt.Sprintf("clients_%d_ips_%d/known", cfg.clients, cfg.ipsPerClient), func(b *testing.B) {
			initIPMonitorPerfDB(b)
			names := seedIPMonitorPerfClients(b, cfg.clients, cfg.ipsPerClient)
			if err := WarmUp(); err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(cfg.clients), "clients")
			b.ReportMetric(float64(cfg.ipsPerClient), "ips/client")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name := names[i%len(names)]
				if !Allow(name, ipMonitorPerfIP(i%len(names), 0)) {
					b.Fatal("known IP should be allowed")
				}
			}
		})
		b.Run(fmt.Sprintf("clients_%d_ips_%d/reject_over_limit", cfg.clients, cfg.ipsPerClient), func(b *testing.B) {
			initIPMonitorPerfDB(b)
			names := seedIPMonitorPerfClients(b, cfg.clients, cfg.ipsPerClient)
			if err := WarmUp(); err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(cfg.clients), "clients")
			b.ReportMetric(float64(cfg.ipsPerClient), "ips/client")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name := names[i%len(names)]
				if Allow(name, ipMonitorPerfNewIP(i%len(names))) {
					b.Fatal("new IP over enforce limit should be rejected")
				}
			}
		})
	}
}

func TestIPMonitorAllowAnchorIssue18Phase5(t *testing.T) {
	initIPMonitorPerfDB(t)
	names := seedIPMonitorPerfClients(t, 1000, 5)
	start := time.Now()
	if err := WarmUp(); err != nil {
		t.Fatal(err)
	}
	warmup := time.Since(start)
	if !Allow(names[0], ipMonitorPerfIP(0, 0)) {
		t.Fatal("known IP should be allowed")
	}
	if Allow(names[0], ipMonitorPerfNewIP(0)) {
		t.Fatal("new IP over enforce limit should be rejected")
	}
	t.Logf("phase5 issue18 anchor: warmup=%s clients=%d ips_per_client=%d", warmup, len(names), 5)
}

func initIPMonitorPerfDB(tb testing.TB) {
	tb.Helper()
	ResetCaches()
	realtime.CloseAll("phase5_reset")
	closeIPMonitorTestDB(database.GetDB())
	dir := makeIPMonitorTempDir(tb, "s-ui-ipmonitor-perf-")
	tb.Setenv("SUI_DB_FOLDER", dir)
	if err := database.InitDB(filepath.Join(dir, "s-ui.db")); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			tb.Skip(err)
		}
		tb.Fatal(err)
	}
	database.GetDB().Config.Logger = gormlogger.Discard
	tb.Cleanup(func() {
		closeIPMonitorTestDB(database.GetDB())
		ResetCaches()
		realtime.CloseAll("phase5_done")
	})
}

func seedIPMonitorPerfClients(tb testing.TB, clients int, ipsPerClient int) []string {
	tb.Helper()
	names := make([]string, clients)
	rows := make([]model.Client, clients)
	for i := 0; i < clients; i++ {
		name := fmt.Sprintf("phase5-user-%04d", i)
		names[i] = name
		rows[i] = model.Client{
			Enable:      true,
			Name:        name,
			LimitIP:     ipsPerClient,
			IPLimitMode: ModeEnforce,
			Config:      []byte(`{}`),
			Inbounds:    []byte(`[]`),
			Links:       []byte(`[]`),
		}
	}
	if err := database.GetDB().CreateInBatches(&rows, database.SafeSQLiteBatchSize(database.GetDB(), &model.Client{})).Error; err != nil {
		tb.Fatal(err)
	}
	now := time.Now().Unix()
	ipRows := make([]model.ClientIP, 0, clients*ipsPerClient)
	for i, name := range names {
		for j := 0; j < ipsPerClient; j++ {
			ip := ipMonitorPerfIP(i, j)
			hash, err := hashIP(ip)
			if err != nil {
				tb.Fatal(err)
			}
			ipRows = append(ipRows, model.ClientIP{
				ClientName: name,
				IPHash:     hash,
				FirstSeen:  now,
				LastSeen:   now,
			})
		}
	}
	if err := database.GetDB().CreateInBatches(&ipRows, database.SafeSQLiteBatchSize(database.GetDB(), &model.ClientIP{})).Error; err != nil {
		tb.Fatal(err)
	}
	return names
}

func ipMonitorPerfIP(clientIndex int, ipIndex int) string {
	return fmt.Sprintf("198.51.%d.%d", clientIndex/256, (clientIndex%256+ipIndex)%250+1)
}

func ipMonitorPerfNewIP(clientIndex int) string {
	return fmt.Sprintf("203.0.%d.%d", clientIndex/256, clientIndex%250+1)
}
