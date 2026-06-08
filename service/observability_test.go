package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"gorm.io/gorm"
)

func TestObservabilityBucketsAreBoundedByDefault(t *testing.T) {
	initSettingTestDB(t)
	resetObservabilityHistoryForTest(t)
	svc := &ObservabilityService{}

	for i := 0; i < 350; i++ {
		if err := svc.RecordObservabilitySample(ObservabilityBucket2s, testObservabilitySample(i)); err != nil {
			t.Fatal(err)
		}
		if err := svc.RecordCoreSample(ObservabilityBucket5m, testCoreSample(i)); err != nil {
			t.Fatal(err)
		}
	}

	samples, err := svc.HistoryForBucket(ObservabilityBucket2s)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != observabilityDefaultBucketCaps[ObservabilityBucket2s] {
		t.Fatalf("unexpected 2s bucket length: %d", len(samples))
	}
	if samples[0].DateTime != 50 {
		t.Fatalf("2s bucket did not retain newest samples first=%d", samples[0].DateTime)
	}

	coreSamples, err := svc.CoreHistoryForBucket(ObservabilityBucket5m)
	if err != nil {
		t.Fatal(err)
	}
	if len(coreSamples) != observabilityDefaultBucketCaps[ObservabilityBucket5m] {
		t.Fatalf("unexpected 5m bucket length: %d", len(coreSamples))
	}
	if coreSamples[0].DateTime != 206 {
		t.Fatalf("5m bucket did not retain newest samples first=%d", coreSamples[0].DateTime)
	}
	if _, err := svc.HistoryForBucket(ObservabilityBucket("10s")); err == nil {
		t.Fatal("invalid bucket should be rejected")
	}
}

func TestObservabilityMemoryCapShrinksBuckets(t *testing.T) {
	settingService := initSettingTestDB(t)
	resetObservabilityHistoryForTest(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "observabilityMemoryCapMB").Update("value", "1").Error; err != nil {
		t.Fatal(err)
	}
	svc := &ObservabilityService{}
	expectedCap := capsForObservabilityMemory(1)[ObservabilityBucket2s]
	if expectedCap >= observabilityDefaultBucketCaps[ObservabilityBucket2s] {
		t.Fatalf("test setup did not shrink capacity: %d", expectedCap)
	}

	for i := 0; i < 350; i++ {
		if err := svc.RecordObservabilitySample(ObservabilityBucket2s, testObservabilitySample(i)); err != nil {
			t.Fatal(err)
		}
	}

	samples, err := svc.HistoryForBucket(ObservabilityBucket2s)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != expectedCap {
		t.Fatalf("expected cap %d, got %d", expectedCap, len(samples))
	}
	if samples[0].DateTime != int64(350-expectedCap) {
		t.Fatalf("memory-capped bucket did not retain newest samples first=%d", samples[0].DateTime)
	}
}

func TestObservabilityMemoryCapCacheRefreshesAfterTTL(t *testing.T) {
	settingService := initSettingTestDB(t)
	resetObservabilityHistoryForTest(t)
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	observabilityMemoryCapCache = newObservabilityMemoryCapCache(func() time.Time {
		return now
	})

	if err := setObservabilityMemoryCapForTest("1"); err != nil {
		t.Fatal(err)
	}
	svc := &ObservabilityService{}
	if err := svc.RecordObservabilitySample(ObservabilityBucket2s, testObservabilitySample(1)); err != nil {
		t.Fatal(err)
	}
	if applied := appliedObservabilityMemoryCapForTest(); applied != 1 {
		t.Fatalf("expected initial cap 1, got %d", applied)
	}

	if err := setObservabilityMemoryCapForTest("32"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(observabilityMemoryCapCacheTTL - time.Second)
	if err := svc.RecordObservabilitySample(ObservabilityBucket2s, testObservabilitySample(2)); err != nil {
		t.Fatal(err)
	}
	if applied := appliedObservabilityMemoryCapForTest(); applied != 1 {
		t.Fatalf("cap refreshed before TTL expired: %d", applied)
	}

	now = now.Add(2 * time.Second)
	if err := svc.RecordObservabilitySample(ObservabilityBucket2s, testObservabilitySample(3)); err != nil {
		t.Fatal(err)
	}
	if applied := appliedObservabilityMemoryCapForTest(); applied != observabilityDefaultMemoryCapMB {
		t.Fatalf("cap did not refresh after TTL: %d", applied)
	}
}

func TestObservabilityMemoryCapSettingValidation(t *testing.T) {
	settingService := initSettingTestDB(t)
	payload, err := json.Marshal(map[string]string{
		"observabilityMemoryCapMB": "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		return settingService.Save(tx, payload)
	})
	if err == nil {
		t.Fatal("invalid observability memory cap should be rejected")
	}
}

func resetObservabilityHistoryForTest(t *testing.T) {
	t.Helper()
	oldHistory := observabilityHistory
	oldMemoryCapCache := observabilityMemoryCapCache
	observabilityHistory = newObservabilityStore()
	observabilityMemoryCapCache = newObservabilityMemoryCapCache(time.Now)
	t.Cleanup(func() {
		observabilityHistory = oldHistory
		observabilityMemoryCapCache = oldMemoryCapCache
	})
}

func setObservabilityMemoryCapForTest(value string) error {
	return database.GetDB().Model(model.Setting{}).
		Where("key = ?", "observabilityMemoryCapMB").
		Update("value", value).Error
}

func appliedObservabilityMemoryCapForTest() int {
	observabilityHistory.mu.RLock()
	defer observabilityHistory.mu.RUnlock()
	return observabilityHistory.lastAppliedMemoryCap
}

func BenchmarkObservabilityHistoryForBucketRead(b *testing.B) {
	oldHistory := observabilityHistory
	observabilityHistory = newObservabilityStore()
	b.Cleanup(func() { observabilityHistory = oldHistory })

	observabilityHistory.mu.Lock()
	for i := 0; i < observabilityDefaultBucketCaps[ObservabilityBucket2s]; i++ {
		observabilityHistory.samples[ObservabilityBucket2s].append(testObservabilitySample(i))
	}
	observabilityHistory.mu.Unlock()

	svc := &ObservabilityService{}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := svc.HistoryForBucket(ObservabilityBucket2s); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func testObservabilitySample(i int) ObservabilitySample {
	return ObservabilitySample{
		DateTime: int64(i),
		CPU:      float64(i),
		Memory: map[string]interface{}{
			// #nosec G115 -- synthetic test loop index, always small and non-negative.
			"current": uint64(i),
		},
		Network: map[string]interface{}{
			// #nosec G115 -- synthetic test loop index, always small and non-negative.
			"sent": uint64(i),
		},
	}
}

func testCoreSample(i int) CoreSample {
	return CoreSample{
		DateTime: int64(i),
		Core: map[string]interface{}{
			"running": i%2 == 0,
		},
	}
}
