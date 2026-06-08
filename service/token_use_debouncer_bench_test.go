package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func BenchmarkTokenUseDebouncer_Record(b *testing.B) {
	for _, parallelism := range []int{1, 16, 64} {
		parallelism := parallelism
		b.Run(fmt.Sprintf("parallel_%d", parallelism), func(b *testing.B) {
			var flushed atomic.Int64
			debouncer := newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
				flushed.Add(int64(len(updates)))
				return nil
			})
			var seq atomic.Uint64
			b.SetParallelism(parallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					n := seq.Add(1)
					// #nosec G115 -- bench sequence counter is well within int64 range.
					debouncer.Record(uint(n%10_000)+1, "198.51.100.1", int64(n))
				}
			})
			b.StopTimer()
			if err := debouncer.Flush(context.Background()); err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(flushed.Load()), "flushed_unique")
		})
	}
}

func BenchmarkTokenUseDebouncer_BatchFlush(b *testing.B) {
	initServicePerfDB(b)
	const tokens = 10_000
	rows := make([]model.Tokens, tokens)
	updates := make(map[uint]tokenUseUpdate, tokens)
	for i := 0; i < tokens; i++ {
		rows[i] = model.Tokens{Desc: fmt.Sprintf("phase5-token-%05d", i), TokenHash: fmt.Sprintf("hash-%05d", i), Enabled: true, UserId: 1}
		// #nosec G115 -- bench loop index, always small and non-negative.
		updates[uint(i+1)] = tokenUseUpdate{ip: "198.51.100.10", ts: int64(1700000000 + i)}
	}
	if err := database.GetDB().CreateInBatches(&rows, tokenUseBatchSize).Error; err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(tokens), "tokens")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := flushTokenUseUpdates(updates); err != nil {
			b.Fatal(err)
		}
	}
}
