package cronjob

import (
	"sync"

	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"
)

type CheckCoreJob struct {
	service.ConfigService
	mu          sync.Mutex
	lastRunning *bool
}

func NewCheckCoreJob() *CheckCoreJob {
	return &CheckCoreJob{}
}

func (s *CheckCoreJob) Run() {
	before := s.ConfigService.IsCoreRunning()
	err := s.ConfigService.StartCore()
	after := s.ConfigService.IsCoreRunning()

	shouldPublish := before != after
	s.mu.Lock()
	if s.lastRunning != nil && *s.lastRunning != after {
		shouldPublish = true
	}
	afterSnapshot := after
	s.lastRunning = &afterSnapshot
	s.mu.Unlock()

	if shouldPublish {
		payload := map[string]any{
			"running": after,
		}
		if err != nil {
			payload["warning"] = "start_failed"
		}
		realtime.Publish(realtime.TopicCoreState, payload)
	}
}
