package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClientListQuery struct {
	Page, Size                 int
	Search, Group, State, Sort string
	Descending, Fold           bool
	Online                     []string
}

type ClientListPage struct {
	Clients  []model.Client `json:"clients"`
	Groups   []string       `json:"groups"`
	Total    int64          `json:"total"`
	Filtered int64          `json:"filtered"`
	Online   int64          `json:"online"`
	Page     int            `json:"page"`
}

func (s *ClientService) ListPage(q ClientListQuery) (ClientListPage, error) {
	result := ClientListPage{Clients: []model.Client{}, Groups: []string{}}
	size := min(100, max(1, q.Size))
	if q.Size <= 0 {
		size = 10
	}
	names, err := json.Marshal(q.Online)
	if err != nil {
		return result, err
	}
	online := "name IN (SELECT value FROM json_each(?))"
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		base := tx.Model(&model.Client{})
		if err := base.Count(&result.Total).Error; err != nil {
			return err
		}
		if err := base.Session(&gorm.Session{}).Select("DISTINCT COALESCE([group], '')").Order("[group]").Scan(&result.Groups).Error; err != nil {
			return err
		}
		if err := base.Session(&gorm.Session{}).Where(online, string(names)).Count(&result.Online).Error; err != nil {
			return err
		}
		filter := tx.Model(&model.Client{})

		if q.Group != "-" {
			filter = filter.Where("[group] = ?", q.Group)
		}
		switch q.State {
		case "disable":
			filter = filter.Where("NOT enable")
		case "expired":
			filter = filter.Where("expiry > 0 AND expiry <= ?", time.Now().Unix())
		case "online":
			filter = filter.Where(online, string(names))
		}
		if q.Search != "" && q.Fold {
			// SQLite lower() folds ASCII only. Match Unicode client names and
			// descriptions before applying sort and pagination to the matches.
			var candidates []struct {
				Id         uint
				Name, Desc string
			}
			if err := filter.Session(&gorm.Session{}).Select("id, name, desc").Scan(&candidates).Error; err != nil {
				return err
			}
			needle := strings.ToLower(q.Search)
			matches := []uint{}
			for _, candidate := range candidates {
				if strings.Contains(strings.ToLower(candidate.Name), needle) || strings.Contains(strings.ToLower(candidate.Desc), needle) {
					matches = append(matches, candidate.Id)
				}
			}
			ids, err := json.Marshal(matches)
			if err != nil {
				return err
			}
			filter = filter.Where("id IN (SELECT value FROM json_each(?))", string(ids))
		} else if q.Search != "" {
			filter = filter.Where("instr(name, ?) > 0 OR instr(desc, ?) > 0", q.Search, q.Search)
		}

		if err := filter.Count(&result.Filtered).Error; err != nil {
			return err
		}
		result.Page = max(1, min(q.Page, int((result.Filtered+int64(size)-1)/int64(size))))
		sortColumn := map[string]string{
			"name": "name COLLATE NOCASE", "group": "[group] COLLATE NOCASE",
			"desc": "desc COLLATE NOCASE", "expiry": "expiry", "volume": "volume",
			"lastIpCount": "last_ip_count", "inbounds": "inbounds",
		}[q.Sort]
		direction := " ASC"
		if q.Descending {
			direction = " DESC"
		}
		switch {
		case q.Sort == "online":
			filter = filter.Order(clause.OrderBy{Expression: clause.Expr{
				SQL: "(" + online + ")" + direction + ", id", Vars: []any{string(names)},
			}})
		case sortColumn != "":
			filter = filter.Order(sortColumn + direction).Order("id")
		default:
			filter = filter.Order("id")
		}
		return filter.Select(clientListColumns).Limit(size).
			Offset((result.Page - 1) * size).Scan(&result.Clients).Error
	})
	return result, err
}

// GetOverviewClients bounds the dashboard payload to the most active clients.
func (s *ClientService) GetOverviewClients() ([]model.Client, error) {
	var result []model.Client
	err := database.GetDB().Model(&model.Client{}).Select("id, name, up, down").
		Order("MAX(0, COALESCE(up, 0)) + MAX(0, COALESCE(down, 0)) DESC").
		Order("name COLLATE NOCASE, id").Limit(10).Scan(&result).Error
	return result, err
}
