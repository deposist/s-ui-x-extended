package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkTelegramNotifier_Enqueue(b *testing.B) {
	b.Run("fake_send_success", func(b *testing.B) {
		var sent atomic.Int64
		notifier := newTelegramNotifier(0, func(string) TelegramResult {
			sent.Add(1)
			return TelegramResult{Success: true}
		}, nil)
		job := telegramNotification{event: "phase5", text: "phase5 telegram bench"}
		b.ReportMetric(float64(telegramQueueCapacity), "capacity")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			notifier.Enqueue(job)
		}
		b.StopTimer()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = notifier.Stop(ctx)
		b.ReportMetric(float64(sent.Load()), "sent")
	})

	b.Run("overflow_blocked_sender", func(b *testing.B) {
		release := make(chan struct{})
		var sent atomic.Int64
		var overflows atomic.Int64
		notifier := newTelegramNotifier(0, func(string) TelegramResult {
			sent.Add(1)
			<-release
			return TelegramResult{Success: true}
		}, func(string, map[string]any) {
			overflows.Add(1)
		})
		job := telegramNotification{event: "phase5", text: "phase5 telegram bench"}
		b.ReportMetric(float64(telegramQueueCapacity), "capacity")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			notifier.Enqueue(job)
		}
		b.StopTimer()
		close(release)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = notifier.Stop(ctx)
		b.ReportMetric(float64(overflows.Load()), "overflows")
		b.ReportMetric(float64(sent.Load()), "sent")
	})
}

func TestTelegramNotifierOverflowAnchorPhase5(t *testing.T) {
	release := make(chan struct{})
	var overflows atomic.Int64
	notifier := newTelegramNotifier(telegramQueueCapacity, func(string) TelegramResult {
		<-release
		return TelegramResult{Success: true}
	}, func(string, map[string]any) {
		overflows.Add(1)
	})
	for i := 0; i < telegramQueueCapacity+100; i++ {
		notifier.Enqueue(telegramNotification{event: "phase5", text: "phase5"})
	}
	if got := overflows.Load(); got == 0 {
		close(release)
		t.Fatal("expected telegram notifier overflow under blocked sender")
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = notifier.Stop(ctx)
	t.Logf("phase5 telegram overflow anchor: overflows=%d capacity=%d", overflows.Load(), telegramQueueCapacity)
}
