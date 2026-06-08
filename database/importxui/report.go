package importxui

type Report struct {
	Summary         Summary          `json:"summary"`
	Warnings        []string         `json:"warnings"`
	ByInbound       []InboundStat    `json:"by_inbound"`
	BackupPath      string           `json:"backup_path,omitempty"`
	GeneratedAdmins []GeneratedAdmin `json:"generated_admins,omitempty"`
}

type Summary struct {
	Inbounds   CountSummary    `json:"inbounds"`
	Endpoints  EndpointSummary `json:"endpoints"`
	Outbounds  EndpointSummary `json:"outbounds"`
	TLS        TLSSummary      `json:"tls"`
	Clients    ClientSummary   `json:"clients"`
	Historical CountSummary    `json:"historical,omitempty"`
	Routing    CountSummary    `json:"routing,omitempty"`
}

type CountSummary struct {
	Total     int `json:"total"`
	Imported  int `json:"imported"`
	Skipped   int `json:"skipped"`
	Conflicts int `json:"conflicts"`
}

type EndpointSummary struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type TLSSummary struct {
	Created int `json:"created"`
	Reused  int `json:"reused"`
}

type ClientSummary struct {
	UniqueEmails int `json:"unique_emails"`
	Merged       int `json:"merged"`
	Created      int `json:"created"`
}

type InboundStat struct {
	SrcTag  string `json:"src_tag"`
	DstTag  string `json:"dst_tag"`
	Clients int    `json:"clients"`
}

type GeneratedAdmin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *Report) warn(message string) {
	if message == "" {
		return
	}
	r.Warnings = append(r.Warnings, message)
}

func (r *Report) warnAll(messages []string) {
	for _, message := range messages {
		r.warn(message)
	}
}
