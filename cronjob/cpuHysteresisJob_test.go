package cronjob

import "testing"

func TestCPUHysteresisJobAlertsOnTransitions(t *testing.T) {
	samples := []float64{91, 92, 93, 94, 95, 96, 50, 49, 48, 47, 46}
	var events []string
	job := &CPUHysteresisJob{
		settings: func() (bool, float64) {
			return true, 90
		},
		notify: func(event string, fields map[string]string) {
			events = append(events, event)
			if fields["threshold"] == "" || fields["cpu"] == "" {
				t.Fatalf("missing cpu fields: %#v", fields)
			}
		},
	}
	job.cpuPercent = func() float64 {
		if len(samples) == 0 {
			return 0
		}
		value := samples[0]
		samples = samples[1:]
		return value
	}

	for i := 0; i < 11; i++ {
		job.Run()
	}
	if len(events) != 2 || events[0] != "cpu_high" || events[1] != "cpu_normal" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestCPUHysteresisJobDoesNotRepeatHighAlert(t *testing.T) {
	var events []string
	job := &CPUHysteresisJob{
		cpuPercent: func() float64 {
			return 99
		},
		settings: func() (bool, float64) {
			return true, 90
		},
		notify: func(event string, fields map[string]string) {
			events = append(events, event)
		},
	}
	for i := 0; i < 12; i++ {
		job.Run()
	}
	if len(events) != 1 || events[0] != "cpu_high" {
		t.Fatalf("expected one high alert, got %#v", events)
	}
}

func TestCPUHysteresisJobDisabledResetsState(t *testing.T) {
	enabled := true
	var events []string
	job := &CPUHysteresisJob{
		cpuPercent: func() float64 {
			return 99
		},
		settings: func() (bool, float64) {
			return enabled, 90
		},
		notify: func(event string, fields map[string]string) {
			events = append(events, event)
		},
	}
	for i := 0; i < cpuHysteresisSamples; i++ {
		job.Run()
	}
	enabled = false
	job.Run()
	enabled = true
	for i := 0; i < cpuHysteresisSamples; i++ {
		job.Run()
	}
	if len(events) != 2 {
		t.Fatalf("expected alert before and after disabled reset, got %#v", events)
	}
}
