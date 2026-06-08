package cronjob

import (
	"fmt"
	"sync"

	"github.com/deposist/s-ui-x/service"
)

const cpuHysteresisSamples = 5

type CPUHysteresisJob struct {
	service.ServerService
	service.SettingService
	service.TelegramService

	mu          sync.Mutex
	window      []bool
	alertActive bool

	cpuPercent func() float64
	settings   func() (bool, float64)
	notify     func(string, map[string]string)
}

func NewCPUHysteresisJob() *CPUHysteresisJob {
	job := &CPUHysteresisJob{}
	job.cpuPercent = job.ServerService.GetCpuPercent
	job.settings = func() (bool, float64) {
		enabled, err := job.SettingService.GetTelegramNotifyCpu()
		if err != nil || !enabled {
			return false, 0
		}
		threshold, err := job.SettingService.GetTelegramCpuThreshold()
		if err != nil || threshold <= 0 {
			threshold = 90
		}
		return true, float64(threshold)
	}
	job.notify = job.TelegramService.NotifyTelegramEvent
	return job
}

func (j *CPUHysteresisJob) Run() {
	enabled, threshold := j.settings()
	if !enabled {
		j.reset()
		return
	}
	cpu := j.cpuPercent()
	above := cpu >= threshold

	j.mu.Lock()
	j.window = append(j.window, above)
	if len(j.window) > cpuHysteresisSamples {
		copy(j.window, j.window[1:])
		j.window = j.window[:cpuHysteresisSamples]
	}
	if len(j.window) < cpuHysteresisSamples {
		j.mu.Unlock()
		return
	}
	allAbove := true
	allBelow := true
	for _, sampleAbove := range j.window {
		allAbove = allAbove && sampleAbove
		allBelow = allBelow && !sampleAbove
	}
	var event string
	if allAbove && !j.alertActive {
		j.alertActive = true
		event = "cpu_high"
	} else if allBelow && j.alertActive {
		j.alertActive = false
		event = "cpu_normal"
	}
	j.mu.Unlock()

	if event != "" {
		j.notify(event, map[string]string{
			"cpu":       fmt.Sprintf("%.1f", cpu),
			"threshold": fmt.Sprintf("%.1f", threshold),
		})
	}
}

func (j *CPUHysteresisJob) reset() {
	j.mu.Lock()
	j.window = nil
	j.alertActive = false
	j.mu.Unlock()
}
