package service

import (
	"time"
)

type PanelService struct {
	RestartScheduler RestartScheduler
	Runtime          *Runtime
}

func NewPanelService(restartScheduler RestartScheduler) *PanelService {
	return &PanelService{RestartScheduler: restartScheduler}
}

func (s *PanelService) RestartPanel(delay time.Duration) error {
	var restartScheduler RestartScheduler
	var runtime *Runtime
	if s != nil {
		restartScheduler = s.RestartScheduler
		runtime = s.Runtime
	}
	if restartScheduler == nil {
		restartScheduler = runtimeOrDefault(runtime).RestartScheduler()
	}
	return restartScheduler.ScheduleRestart(delay)
}
