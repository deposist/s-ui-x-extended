package sub

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type SubService struct {
	service.SettingService
	LinkService
}

func (s *SubService) GetSubs(subId string) (*string, []string, error) {
	now := time.Now()
	cacheKey := "base:" + subId
	if body, headers, ok := subscriptionCacheGet(cacheKey, now); ok {
		return &body, headers, nil
	}
	client, err := s.getClientBySubId(subId)
	if err != nil {
		return nil, nil, err
	}

	cfg := cachedSubDisplaySettings(&s.SettingService, time.Now())

	clientInfo := ""
	if cfg.showInfo {
		clientInfo = s.getClientInfo(client)
	}
	if cfg.nameInRemark {
		clientInfo = " " + client.Name + clientInfo
	}

	linksArray := s.LinkService.GetLinks(&client.Links, "all", clientInfo)
	result := strings.Join(linksArray, "\n")

	headers := buildClientHeaders(client, cfg)

	if cfg.encode {
		result = base64.StdEncoding.EncodeToString([]byte(result))
	}

	subscriptionCacheSet(cacheKey, result, headers, now)
	return &result, headers, nil
}

func (j *SubService) getClientBySubId(subId string) (*model.Client, error) {
	db := database.GetDB()
	client := &model.Client{}
	err := db.Model(model.Client{}).Where("enable = true and sub_secret = ?", subId).First(client).Error
	if err == nil {
		return client, j.ensureClientSubSecret(db, client)
	}
	if !database.IsNotFound(err) {
		return nil, err
	}
	required, _ := j.SettingService.GetSubSecretRequired()
	if required {
		return nil, gorm.ErrRecordNotFound
	}
	// Legacy name-based lookup, active only when the admin has disabled required
	// sub-secrets. Client names are admin-chosen and often guessable, so this
	// fallback allows unauthenticated enumeration of other clients' configs by
	// name. Warn whenever it actually serves a config so the operator is aware
	// the insecure mode is on (enable required sub-secrets to close it).
	err = db.Model(model.Client{}).Where("enable = true and name = ?", subId).First(client).Error
	if err != nil {
		return nil, err
	}
	logger.Warning("sub: served config via legacy name lookup (subSecretRequired is OFF). Enable required sub-secrets to prevent name-based enumeration")
	return client, j.ensureClientSubSecret(db, client)
}

func buildClientHeaders(client *model.Client, cfg subDisplaySettings) []string {
	headers := util.GetHeaders(client, cfg.updates)
	if cfg.title != "" {
		headers[2] = cfg.title
	}
	headers = append(headers, cfg.supportURL, cfg.profileURL, cfg.announce)
	return headers
}

func (s *SubService) getClientInfo(c *model.Client) string {
	now := time.Now().Unix()

	var result []string
	if vol := c.Volume - (c.Up + c.Down); vol > 0 {
		result = append(result, fmt.Sprintf("%s left", s.formatTraffic(vol)))
	}
	if c.Expiry > 0 {
		days := (c.Expiry - now) / 86400
		label := "days"
		if days == 1 {
			label = "day"
		}
		result = append(result, fmt.Sprintf("%d %s left", days, label))
	}
	if len(result) > 0 {
		return " (" + strings.Join(result, ", ") + ")"
	} else {
		return " (unlimited)"
	}
}

func (s *SubService) formatTraffic(trafficBytes int64) string {
	if trafficBytes < 1024 {
		return fmt.Sprintf("%.2fB", float64(trafficBytes)/float64(1))
	} else if trafficBytes < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(trafficBytes)/float64(1024))
	} else if trafficBytes < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(trafficBytes)/float64(1024*1024))
	} else if trafficBytes < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(trafficBytes)/float64(1024*1024*1024))
	} else if trafficBytes < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(trafficBytes)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fEB", float64(trafficBytes)/float64(1024*1024*1024*1024*1024))
	}
}

func (s *SubService) ensureClientSubSecret(db *gorm.DB, client *model.Client) error {
	if client.SubSecret != "" {
		return nil
	}
	secret, err := uuid.NewV4()
	if err != nil {
		return err
	}
	client.SubSecret = secret.String()
	return db.Model(model.Client{}).Where("id = ?", client.Id).Update("sub_secret", client.SubSecret).Error
}
