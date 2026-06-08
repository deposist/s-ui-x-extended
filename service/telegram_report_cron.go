package service

import (
	"strings"
	"time"

	"github.com/deposist/s-ui-x/util/common"
	"github.com/robfig/cron/v3"
)

var telegramReportCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func ParseTelegramReportCron(spec string) (cron.Schedule, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	schedule, err := telegramReportCronParser.Parse(spec)
	if err != nil {
		return nil, err
	}
	first := schedule.Next(time.Unix(0, 0))
	second := schedule.Next(first)
	if !second.IsZero() && second.Sub(first) < time.Minute {
		return nil, common.NewError("telegramReportCron step must be at least 1 minute")
	}
	return schedule, nil
}
