package realtime

import (
	"fmt"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkPublishToNClients(b *testing.B) {
	for _, subscribers := range []int{10, 100, 1000} {
		subscribers := subscribers
		b.Run(fmt.Sprintf("subscribers_%d", subscribers), func(b *testing.B) {
			h := newHub()
			channels := make([]chan Event, 0, subscribers)
			unregisters := make([]func(), 0, subscribers)
			var drops atomic.Int64
			for i := 0; i < subscribers; i++ {
				ch := make(chan Event, 1024)
				channels = append(channels, ch)
				go func(events <-chan Event) {
					for range events {
					}
				}(ch)
				unregisters = append(unregisters, h.Register(&ClientHandle{
					User:   fmt.Sprintf("user-%04d", i),
					IP:     fmt.Sprintf("127.0.0.%d", i%250+1),
					Scope:  ScopeAdmin,
					SendCh: ch,
					OnDrop: func(string) {
						drops.Add(1)
					},
				}))
			}
			defer func() {
				for _, unregister := range unregisters {
					unregister()
				}
				for _, ch := range channels {
					close(ch)
				}
			}()

			samples := make([]int64, 0, min(4096, b.N))
			b.ReportMetric(float64(subscribers), "subscribers")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()
				h.Publish(TopicNotification, map[string]any{"seq": i})
				elapsed := time.Since(start).Nanoseconds()
				if len(samples) < cap(samples) {
					samples = append(samples, elapsed)
				}
			}
			b.StopTimer()
			reportLatencyDistribution(b, samples)
			b.ReportMetric(float64(drops.Load()), "drops")
		})
	}
}

func reportLatencyDistribution(b *testing.B, samples []int64) {
	b.Helper()
	if len(samples) == 0 {
		return
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p50 := samples[len(samples)/2]
	p95 := samples[(len(samples)*95)/100]
	if len(samples) == 1 {
		p95 = samples[0]
	}
	b.ReportMetric(float64(p50), "p50_ns")
	b.ReportMetric(float64(p95), "p95_ns")
	b.ReportMetric(float64(samples[len(samples)-1]), "max_ns")
}
