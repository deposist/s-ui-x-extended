package service

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/ipmonitor"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/realtime"

	"gorm.io/gorm"
)

type onlines struct {
	Inbound  []string `json:"inbound,omitempty"`
	User     []string `json:"user,omitempty"`
	Outbound []string `json:"outbound,omitempty"`
}

var (
	onlineResources   = &onlines{}
	onlineResourcesMu sync.RWMutex
)

var commitStatsTransaction = func(tx *gorm.DB) error {
	return tx.Commit().Error
}

type StatsService struct {
	Runtime *Runtime
}

func (s *StatsService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

type trafficDelta struct {
	Resource string `json:"resource"`
	Tag      string `json:"tag"`
	Up       int64  `json:"up,omitempty"`
	Down     int64  `json:"down,omitempty"`
}

type clientTrafficDelta struct {
	up   int64
	down int64
}

func (s *StatsService) SaveStats(enableTraffic bool) (err error) {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	box := coreInstance.GetInstance()
	if box == nil {
		return nil
	}
	st := box.StatsTracker()
	if st == nil {
		return nil
	}
	stats := st.GetStats()

	currentOnlines := onlines{}

	if len(*stats) == 0 {
		onlineResourcesMu.Lock()
		onlineResources = &currentOnlines
		onlineResourcesMu.Unlock()
		if err := ipmonitor.Flush(); err != nil {
			return err
		}
		publishStatsRealtime(currentOnlines, nil)
		return nil
	}

	db := database.GetDB()
	tx := db.Begin()
	publishOnCommit := false
	publishOnlines := onlines{}
	var publishStats []model.Stats
	clientDeltas := map[string]clientTrafficDelta{}
	defer func() {
		if err == nil {
			if commitErr := commitStatsTransaction(tx); commitErr != nil {
				err = commitErr
				if auditErr := (&AuditService{Runtime: s.runtime()}).Record(AuditEvent{
					Actor:    "system",
					Event:    "stats_commit_failed",
					Resource: "stats",
					Severity: AuditSeverityWarn,
					Details: map[string]any{
						"error": commitErr.Error(),
					},
				}); auditErr != nil {
					logger.Warning("stats commit failure audit failed:", auditErr)
				}
				realtime.Publish(realtime.TopicCoreState, map[string]any{
					"warning": "stats_commit_failed",
				})
				return
			}
			if publishOnCommit {
				publishStatsRealtime(publishOnlines, publishStats)
			}
		} else {
			tx.Rollback()
		}
	}()

	for _, stat := range *stats {
		if stat.Resource == "user" {
			if stat.Direction {
				delta := clientDeltas[stat.Tag]
				delta.up += stat.Traffic
				clientDeltas[stat.Tag] = delta
			} else {
				delta := clientDeltas[stat.Tag]
				delta.down += stat.Traffic
				clientDeltas[stat.Tag] = delta
			}
		}
		if stat.Direction {
			switch stat.Resource {
			case "inbound":
				currentOnlines.Inbound = append(currentOnlines.Inbound, stat.Tag)
			case "outbound":
				currentOnlines.Outbound = append(currentOnlines.Outbound, stat.Tag)
			case "user":
				currentOnlines.User = append(currentOnlines.User, stat.Tag)
			}
		}
	}
	if err := updateClientTrafficDeltas(tx, clientDeltas); err != nil {
		return err
	}
	onlineResourcesMu.Lock()
	onlineResources = &currentOnlines
	onlineResourcesMu.Unlock()
	publishOnCommit = true
	publishOnlines = currentOnlines
	publishStats = append([]model.Stats(nil), (*stats)...)

	if !enableTraffic {
		return ipmonitor.FlushTo(tx)
	}
	if err := tx.Create(&stats).Error; err != nil {
		return err
	}
	return ipmonitor.FlushTo(tx)
}

func updateClientTrafficDeltas(tx *gorm.DB, deltas map[string]clientTrafficDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	names := make([]string, 0, len(deltas))
	for name, delta := range deltas {
		if delta.up == 0 && delta.down == 0 {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for start := 0; start < len(names); start += 100 {
		end := start + 100
		if end > len(names) {
			end = len(names)
		}
		if err := updateClientTrafficDeltaBatch(tx, names[start:end], deltas); err != nil {
			return err
		}
	}
	return nil
}

func updateClientTrafficDeltaBatch(tx *gorm.DB, names []string, deltas map[string]clientTrafficDelta) error {
	if len(names) == 0 {
		return nil
	}
	var query strings.Builder
	args := make([]any, 0, len(names)*5)
	query.WriteString("UPDATE clients SET up = up + CASE name")
	for _, name := range names {
		query.WriteString(" WHEN ? THEN ?")
		args = append(args, name, deltas[name].up)
	}
	query.WriteString(" ELSE 0 END, down = down + CASE name")
	for _, name := range names {
		query.WriteString(" WHEN ? THEN ?")
		args = append(args, name, deltas[name].down)
	}
	query.WriteString(" ELSE 0 END WHERE name IN (")
	for i, name := range names {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteByte('?')
		args = append(args, name)
	}
	query.WriteByte(')')
	return tx.Exec(query.String(), args...).Error
}

func publishStatsRealtime(currentOnlines onlines, stats []model.Stats) {
	realtime.Publish(realtime.TopicOnlines, currentOnlines)
	realtime.Publish(realtime.TopicTrafficDelta, trafficDeltas(stats))
}

func trafficDeltas(stats []model.Stats) []trafficDelta {
	type key struct {
		resource string
		tag      string
	}
	byKey := map[key]*trafficDelta{}
	order := make([]key, 0)
	for _, stat := range stats {
		k := key{resource: stat.Resource, tag: stat.Tag}
		delta := byKey[k]
		if delta == nil {
			delta = &trafficDelta{Resource: stat.Resource, Tag: stat.Tag}
			byKey[k] = delta
			order = append(order, k)
		}
		if stat.Direction {
			delta.Up += stat.Traffic
		} else {
			delta.Down += stat.Traffic
		}
	}
	result := make([]trafficDelta, 0, len(order))
	for _, k := range order {
		result = append(result, *byKey[k])
	}
	return result
}

func (s *StatsService) GetStats(resource string, tag string, limit int) ([]model.Stats, error) {
	var err error
	var result []model.Stats

	currentTime := time.Now().Unix()
	timeDiff := currentTime - (int64(limit) * 3600)

	db := database.GetDB()
	resources := []string{resource}
	if resource == "endpoint" {
		resources = []string{"inbound", "outbound"}
	}
	err = db.Model(model.Stats{}).Where("resource in ? AND tag = ? AND date_time > ?", resources, tag, timeDiff).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	result = s.downsampleStats(result, 60) // 60 rows for 30 buckets
	return result, nil
}

// downsampleStats reduces stats to maxRows rows.
// Each bucket outputs two rows (direction false and true) with average Traffic.
func (s *StatsService) downsampleStats(stats []model.Stats, maxRows int) []model.Stats {
	if len(stats) <= maxRows {
		return stats
	}
	numBuckets := int(maxRows / 2)
	sort.Slice(stats, func(i, j int) bool { return stats[i].DateTime < stats[j].DateTime })
	timeMin, timeMax := stats[0].DateTime, stats[len(stats)-1].DateTime
	bucketSpan := (timeMax - timeMin) / int64(numBuckets)
	if bucketSpan == 0 {
		bucketSpan = 1
	}
	downsampled := make([]model.Stats, 0, maxRows)
	for i := 0; i < numBuckets; i++ {
		bucketStart := timeMin + int64(i)*bucketSpan
		bucketEnd := timeMin + int64(i+1)*bucketSpan
		if i == numBuckets-1 {
			bucketEnd = timeMax + 1
		}
		for _, dir := range []bool{false, true} {
			var sum int64
			var count int
			for _, r := range stats {
				if r.DateTime >= bucketStart && r.DateTime < bucketEnd && r.Direction == dir {
					sum += r.Traffic
					count++
				}
			}
			avg := int64(0)
			if count > 0 {
				avg = sum / int64(count)
			}
			downsampled = append(downsampled, model.Stats{
				DateTime:  bucketStart,
				Resource:  stats[0].Resource,
				Tag:       stats[0].Tag,
				Direction: dir,
				Traffic:   avg,
			})
		}
	}
	return downsampled
}

func (s *StatsService) GetOnlines() (onlines, error) {
	onlineResourcesMu.RLock()
	defer onlineResourcesMu.RUnlock()
	return onlines{
		Inbound:  append([]string(nil), onlineResources.Inbound...),
		User:     append([]string(nil), onlineResources.User...),
		Outbound: append([]string(nil), onlineResources.Outbound...),
	}, nil
}
func (s *StatsService) DelOldStats(days int) error {
	oldTime := time.Now().AddDate(0, 0, -(days)).Unix()
	db := database.GetDB()
	return db.Where("date_time < ?", oldTime).Delete(model.Stats{}).Error
}
