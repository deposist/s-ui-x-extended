package cronjob

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
)

const (
	observability30sTicks = 15
	observability1mTicks  = 30
	observability5mTicks  = 150
)

type ObservabilitySamplerJob struct {
	service.ObservabilityService

	mu                   sync.Mutex
	ticks                int
	currentObservability func() service.ObservabilitySample
	currentCore          func() service.CoreSample
	now                  func() time.Time
}

func NewObservabilitySamplerJob() *ObservabilitySamplerJob {
	job := &ObservabilitySamplerJob{}
	job.currentObservability = job.ObservabilityService.CurrentObservabilitySample
	job.currentCore = job.ObservabilityService.CurrentCoreSample
	job.now = time.Now
	return job
}

func (j *ObservabilitySamplerJob) Run() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if err := j.RecordObservabilitySample(service.ObservabilityBucket2s, j.currentObservability()); err != nil {
		logger.Warning("record observability sample failed:", err)
		return
	}
	if err := j.RecordCoreSample(service.ObservabilityBucket2s, j.currentCore()); err != nil {
		logger.Warning("record core observability sample failed:", err)
		return
	}
	j.ticks++

	j.aggregateEvery(service.ObservabilityBucket30s, observability30sTicks)
	j.aggregateEvery(service.ObservabilityBucket1m, observability1mTicks)
	j.aggregateEvery(service.ObservabilityBucket5m, observability5mTicks)
}

func (j *ObservabilitySamplerJob) aggregateEvery(bucket service.ObservabilityBucket, interval int) {
	if interval <= 0 || j.ticks%interval != 0 {
		return
	}
	samples, err := j.HistoryForBucket(service.ObservabilityBucket2s)
	if err != nil {
		logger.Warning("read observability samples for aggregation failed:", err)
		return
	}
	if len(samples) == 0 {
		return
	}
	if len(samples) > interval {
		samples = samples[len(samples)-interval:]
	}
	ts := j.now().Unix()
	if err := j.RecordObservabilitySample(bucket, service.AggregateObservabilitySamples(samples, ts)); err != nil {
		logger.Warning("record aggregated observability sample failed:", err)
	}

	coreSamples, err := j.CoreHistoryForBucket(service.ObservabilityBucket2s)
	if err != nil {
		logger.Warning("read core samples for aggregation failed:", err)
		return
	}
	if len(coreSamples) == 0 {
		return
	}
	if len(coreSamples) > interval {
		coreSamples = coreSamples[len(coreSamples)-interval:]
	}
	if err := j.RecordCoreSample(bucket, service.AggregateCoreSamples(coreSamples, ts)); err != nil {
		logger.Warning("record aggregated core sample failed:", err)
	}
}
