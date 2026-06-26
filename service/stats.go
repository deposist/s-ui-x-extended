package service

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"

	"gorm.io/gorm"
)

type onlines struct {
	Inbound        []string                          `json:"inbound,omitempty"`
	User           []string                          `json:"user,omitempty"`
	Outbound       []string                          `json:"outbound,omitempty"`
	Failover       map[string]FailoverStatusEntry    `json:"failover,omitempty"`
	OutboundHealth map[string]OutboundHealthSnapshot `json:"outboundHealth,omitempty"`
	ProviderHealth map[string]ProviderHealthSnapshot `json:"providerHealth,omitempty"`
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

type TrafficBucket struct {
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
	Download  int64 `json:"download"`
	Upload    int64 `json:"upload"`
}

type TrafficSummary struct {
	StartTime int64           `json:"startTime"`
	EndTime   int64           `json:"endTime"`
	Range     int             `json:"range"`
	Buckets   []TrafficBucket `json:"buckets"`
	Download  int64           `json:"download"`
	Upload    int64           `json:"upload"`
}

type trafficSummaryWindow struct {
	LimitHours  int
	BucketCount int
	StartTime   int64
	EndTime     int64
	BucketSpan  int64
}

type trafficAggregateRow struct {
	Bucket    int
	Direction bool
	Traffic   int64
}

const (
	defaultTrafficSummaryHours = 24
	maxTrafficSummaryHours     = 24 * 366
	defaultTrafficBucketCount  = 48
	maxTrafficBucketCount      = 720

	inboundTrafficBucketSelect = "CASE WHEN CAST((date_time - ?) / ? AS INTEGER) >= ? " +
		"THEN ? ELSE CAST((date_time - ?) / ? AS INTEGER) END AS bucket, " +
		"direction, SUM(traffic) AS traffic"
)

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
	// Failover groups have their own liveness, independent of traffic stats, so
	// publish their live status on every tick (including the no-traffic path).
	currentOnlines.Failover = FailoverLiveSnapshot()
	currentOnlines.OutboundHealth = AllOutboundHealthSnapshots()
	currentOnlines.ProviderHealth = AllProviderHealthSnapshots()

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
	if err := database.CreateInBatchesSafe(tx, stats); err != nil {
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

func (s *StatsService) GetInboundTrafficSummary(limitHours int, bucketCount int, endTime int64) (TrafficSummary, error) {
	window := newTrafficSummaryWindow(limitHours, bucketCount, endTime)
	buckets := newTrafficBuckets(window)
	rows, err := queryInboundTrafficBuckets(window)
	if err != nil {
		return TrafficSummary{}, err
	}
	download, upload := applyTrafficAggregateRows(buckets, rows)

	return TrafficSummary{
		StartTime: window.StartTime,
		EndTime:   window.EndTime,
		Range:     window.LimitHours,
		Buckets:   buckets,
		Download:  download,
		Upload:    upload,
	}, nil
}

func newTrafficSummaryWindow(limitHours int, bucketCount int, endTime int64) trafficSummaryWindow {
	if limitHours <= 0 {
		limitHours = defaultTrafficSummaryHours
	}
	if limitHours > maxTrafficSummaryHours {
		limitHours = maxTrafficSummaryHours
	}
	if bucketCount <= 0 {
		bucketCount = defaultTrafficBucketCount
	}
	if bucketCount > maxTrafficBucketCount {
		bucketCount = maxTrafficBucketCount
	}
	if endTime <= 0 {
		endTime = time.Now().Unix()
	}

	startTime := endTime - int64(limitHours)*3600
	if startTime < 0 {
		startTime = 0
	}
	span := endTime - startTime
	if span <= 0 {
		span = 1
	}
	bucketSpan := (span + int64(bucketCount) - 1) / int64(bucketCount)
	if bucketSpan <= 0 {
		bucketSpan = 1
	}

	return trafficSummaryWindow{
		LimitHours:  limitHours,
		BucketCount: bucketCount,
		StartTime:   startTime,
		EndTime:     endTime,
		BucketSpan:  bucketSpan,
	}
}

func newTrafficBuckets(window trafficSummaryWindow) []TrafficBucket {
	buckets := make([]TrafficBucket, window.BucketCount)
	for i := range buckets {
		bucketStart := window.StartTime + int64(i)*window.BucketSpan
		if bucketStart > window.EndTime {
			bucketStart = window.EndTime
		}
		bucketEnd := bucketStart + window.BucketSpan
		if bucketEnd > window.EndTime {
			bucketEnd = window.EndTime
		}
		buckets[i] = TrafficBucket{StartTime: bucketStart, EndTime: bucketEnd}
	}
	return buckets
}

func queryInboundTrafficBuckets(window trafficSummaryWindow) ([]trafficAggregateRow, error) {
	var rows []trafficAggregateRow
	err := database.GetDB().Model(model.Stats{}).
		Select(
			inboundTrafficBucketSelect,
			window.StartTime, window.BucketSpan, window.BucketCount, window.BucketCount-1, window.StartTime, window.BucketSpan,
		).
		Where("resource = ? AND date_time >= ? AND date_time <= ?", "inbound", window.StartTime, window.EndTime).
		Group("bucket, direction").
		Scan(&rows).Error
	return rows, err
}

func applyTrafficAggregateRows(buckets []TrafficBucket, rows []trafficAggregateRow) (int64, int64) {
	var download int64
	var upload int64
	for _, row := range rows {
		if row.Bucket < 0 || row.Bucket >= len(buckets) {
			continue
		}
		if row.Direction {
			buckets[row.Bucket].Upload += row.Traffic
			upload += row.Traffic
		} else {
			buckets[row.Bucket].Download += row.Traffic
			download += row.Traffic
		}
	}
	return download, upload
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
	type bucketTotals struct {
		sum   [2]int64
		count [2]int
	}
	buckets := make([]bucketTotals, numBuckets)
	for _, r := range stats {
		idx := int((r.DateTime - timeMin) / bucketSpan)
		if idx < 0 {
			idx = 0
		} else if idx >= numBuckets {
			idx = numBuckets - 1
		}
		dirIdx := 0
		if r.Direction {
			dirIdx = 1
		}
		buckets[idx].sum[dirIdx] += r.Traffic
		buckets[idx].count[dirIdx]++
	}

	downsampled := make([]model.Stats, 0, maxRows)
	for i := 0; i < numBuckets; i++ {
		bucketStart := timeMin + int64(i)*bucketSpan
		for dirIdx, dir := range []bool{false, true} {
			avg := int64(0)
			if buckets[i].count[dirIdx] > 0 {
				avg = buckets[i].sum[dirIdx] / int64(buckets[i].count[dirIdx])
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
		Inbound:        append([]string(nil), onlineResources.Inbound...),
		User:           append([]string(nil), onlineResources.User...),
		Outbound:       append([]string(nil), onlineResources.Outbound...),
		Failover:       FailoverLiveSnapshot(),
		OutboundHealth: AllOutboundHealthSnapshots(),
		ProviderHealth: AllProviderHealthSnapshots(),
	}, nil
}
func (s *StatsService) DelOldStats(days int) error {
	oldTime := time.Now().AddDate(0, 0, -(days)).Unix()
	db := database.GetDB()
	return db.Where("date_time < ?", oldTime).Delete(model.Stats{}).Error
}
