package cronjob

import (
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
)

type WALCheckpointJob struct{}

func NewWALCheckpointJob() *WALCheckpointJob {
	return &WALCheckpointJob{}
}

func (s *WALCheckpointJob) Run() {
	db := database.GetDB()
	if db == nil {
		return
	}
	if err := db.Exec("PRAGMA wal_checkpoint(FULL)").Error; err != nil {
		logger.Error("Error checkpointing WAL: ", err.Error())
	}
}
