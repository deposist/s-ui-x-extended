package model

// AWGDevice is the durable desired state for one managed AmneziaWG peer.
type AWGDevice struct {
	Id                uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId          uint   `json:"clientId" gorm:"column:client_id;index;not null"`
	Name              string `json:"name" gorm:"not null"`
	CreateRequestKey  string `json:"-" gorm:"column:create_request_key;not null;default:''"`
	RotateRequestKey  string `json:"-" gorm:"column:rotate_request_key;not null;default:''"`
	CryptoContext     []byte `json:"-" gorm:"column:crypto_context;not null"`
	PublicKey         string `json:"publicKey" gorm:"column:public_key;uniqueIndex;not null"`
	PreviousPublicKey string `json:"-" gorm:"column:previous_public_key;index"`
	PrivateKeyEnc     []byte `json:"-" gorm:"column:private_key_enc;not null"`
	PSKEnc            []byte `json:"-" gorm:"column:psk_enc;not null"`
	IPv4Address       string `json:"ipv4Address" gorm:"column:ipv4_address;not null"`
	DesiredEnabled    bool   `json:"desiredEnabled" gorm:"column:desired_enabled;index;not null;default:true"`
	SyncState         string `json:"syncState" gorm:"column:sync_state;index;not null"`
	Provisioned       bool   `json:"provisioned" gorm:"index;not null;default:false"`
	LastError         string `json:"-" gorm:"column:last_error"`
	RxBaseline        uint64 `json:"-" gorm:"column:rx_baseline;not null;default:0"`
	TxBaseline        uint64 `json:"-" gorm:"column:tx_baseline;not null;default:0"`
	TotalRx           uint64 `json:"totalRx" gorm:"column:total_rx;not null;default:0"`
	TotalTx           uint64 `json:"totalTx" gorm:"column:total_tx;not null;default:0"`
	LastHandshake     int64  `json:"lastHandshake" gorm:"column:last_handshake;not null;default:0"`
	CreatedAt         int64  `json:"createdAt" gorm:"column:created_at;not null"`
	UpdatedAt         int64  `json:"updatedAt" gorm:"column:updated_at;not null"`
	RevokedAt         int64  `json:"revokedAt" gorm:"column:revoked_at;not null;default:0"`
	IPReusableAfter   int64  `json:"-" gorm:"column:ip_reusable_after;index;not null;default:0"`
}

func (AWGDevice) TableName() string { return "awg_devices" }
