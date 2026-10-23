package model

// ClientLimits holds the per-client limits projected onto fleet nodes. The
// limit itself is enforced by the core; this row only records what the operator
// asked for, so the panel can rebuild the fleet projection after a restart.
//
// Speed is stored the way the UI asks for it: one shared number, or separate
// download and upload numbers. The core limiter takes one number per record, so
// a split limit becomes two records and a shared one becomes a single
// bidirectional record — that projection lives in the fleet service, not here.
type ClientLimits struct {
	ClientId uint `json:"clientId" gorm:"primaryKey;autoIncrement:false"`

	// SpeedShared means one number covers both directions.
	SpeedShared bool `json:"speedShared"`
	// Speed is used when SpeedShared is true, in bytes per second.
	Speed uint64 `json:"speed"`
	// DownloadSpeed and UploadSpeed are used when SpeedShared is false.
	DownloadSpeed uint64 `json:"downloadSpeed"`
	UploadSpeed   uint64 `json:"uploadSpeed"`

	// Connections is the connection limit; 0 means unlimited.
	Connections uint32 `json:"connections"`

	UpdatedAt int64 `json:"updatedAt"`
}
