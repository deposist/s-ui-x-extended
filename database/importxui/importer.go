package importxui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"gorm.io/gorm"
)

type importState struct {
	report           *Report
	realityByKey     map[string]*realitySpec
	realityBySource  map[int64]*realitySpec
	plainTLSByKey    map[string]*tlsCertSpec
	plainTLSBySource map[int64]*tlsCertSpec
	tlsIDByKey       map[string]uint
	inboundIDBySrc   map[int64]uint
	clientRefs       []ClientRef
	server           string
	hostname         string
}

func Import(srcPath string, opts Options) (*Report, error) {
	report := &Report{}
	opts, err := opts.normalized()
	if err != nil {
		return report, fmt.Errorf("xui-import: %w", err)
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if err := checkContext(opts.Context); err != nil {
		return report, fmt.Errorf("xui-import: %w", err)
	}
	if !opts.DryRun {
		if !applyMu.TryLock() {
			return report, fmt.Errorf("xui-import: %w", ErrBusy)
		}
		defer applyMu.Unlock()
	}
	src, err := openSource(srcPath)
	if err != nil {
		return report, fmt.Errorf("xui-import: %w", err)
	}
	defer src.close()

	db := database.GetDB()
	if db == nil {
		return report, fmt.Errorf("xui-import: destination database is not initialized")
	}
	tx := db.Begin()
	if tx.Error != nil {
		return report, fmt.Errorf("xui-import: %w", tx.Error)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback().Error
		}
	}()

	state := &importState{
		report:           report,
		realityByKey:     map[string]*realitySpec{},
		realityBySource:  map[int64]*realitySpec{},
		plainTLSByKey:    map[string]*tlsCertSpec{},
		plainTLSBySource: map[int64]*tlsCertSpec{},
		tlsIDByKey:       map[string]uint{},
		inboundIDBySrc:   map[int64]uint{},
		server:           destinationServer(tx),
		hostname:         resolveLinkHostname(tx, opts.Hostname),
	}
	if err := state.run(tx, src, opts); err != nil {
		return report, fmt.Errorf("xui-import: %w", err)
	}
	if opts.DryRun {
		return report, nil
	}
	if err := tx.Commit().Error; err != nil {
		return report, fmt.Errorf("xui-import: %w", err)
	}
	committed = true
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return report, fmt.Errorf("xui-import: %w", err)
	}
	return report, nil
}

func (s *importState) run(tx *gorm.DB, src *sourceDB, opts Options) error {
	total, err := src.inboundCount()
	if err != nil {
		return err
	}
	s.report.Summary.Inbounds.Total = total
	if err := s.importTLS(tx, src); err != nil {
		return err
	}
	if err := s.importInboundsAndEndpoints(tx, src, opts.Strategy); err != nil {
		return err
	}
	if err := s.importClients(tx, src, opts.Strategy); err != nil {
		return err
	}
	if err := s.importOptionalExtras(tx, src, opts); err != nil {
		return err
	}
	if !opts.DryRun && !opts.SkipAudit {
		if err := s.recordAudit(tx, opts); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importOptionalExtras(tx *gorm.DB, src *sourceDB, opts Options) error {
	if !opts.IncludeHistory && !opts.IncludeRouting {
		return nil
	}
	items := map[string]PlanItem{}
	if opts.IncludeHistory {
		items[planKey(KindHistory, "traffic")] = PlanItem{Kind: KindHistory, SrcID: "traffic", DstTag: "stats", Action: ActionCreate}
	}
	if opts.IncludeRouting {
		items[planKey(KindRouting, "xrayConfig")] = PlanItem{Kind: KindRouting, SrcID: "xrayConfig", DstTag: "config", Action: ActionCreate}
	}
	extra := &applyState{
		report:     s.report,
		plan:       items,
		onProgress: opts.OnProgress,
		total:      len(items),
	}
	if opts.IncludeHistory {
		if err := extra.applyHistorical(opts.Context, tx, src, ApplyOptions{Context: opts.Context, Now: opts.Now}); err != nil {
			return err
		}
	}
	if opts.IncludeRouting {
		if err := extra.applyRouting(opts.Context, tx, src, ApplyOptions{Context: opts.Context, Now: opts.Now}); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) importTLS(tx *gorm.DB, src *sourceDB) error {
	return src.eachInbound(func(row xuiInboundRow) error {
		spec, warnings, err := extractReality(row)
		if err != nil {
			return err
		}
		s.report.warnAll(warnings)
		if spec != nil {
			if existing, ok := s.realityByKey[spec.Key]; ok {
				s.realityBySource[row.ID] = existing
				s.report.Summary.TLS.Reused++
				return nil
			}
			s.realityByKey[spec.Key] = spec
			s.realityBySource[row.ID] = spec
			existing, found, err := findExistingRealityTLS(tx, *spec)
			if err != nil {
				return err
			}
			if found {
				s.tlsIDByKey[spec.Key] = existing.Id
				s.report.Summary.TLS.Reused++
				return nil
			}
			record, err := buildTLSRecord(*spec)
			if err != nil {
				return err
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			s.tlsIDByKey[spec.Key] = record.Id
			s.report.Summary.TLS.Created++
			return nil
		}
		return s.importPlainTLS(tx, row)
	})
}

// importPlainTLS migrates a non-reality (plain) TLS inbound's inline certificate
// into an s-ui TLS record, deduplicating by certificate content within the
// import and against existing records. Inbounds whose certificate is only a file
// path produce a warning (handled in extractPlainTLS) and no record.
func (s *importState) importPlainTLS(tx *gorm.DB, row xuiInboundRow) error {
	spec, warnings, err := extractPlainTLS(row)
	if err != nil {
		return err
	}
	s.report.warnAll(warnings)
	if spec == nil {
		return nil
	}
	if existing, ok := s.plainTLSByKey[spec.Key]; ok {
		s.plainTLSBySource[row.ID] = existing
		s.report.Summary.TLS.Reused++
		return nil
	}
	s.plainTLSByKey[spec.Key] = spec
	s.plainTLSBySource[row.ID] = spec
	existing, found, err := findExistingPlainTLS(tx, *spec)
	if err != nil {
		return err
	}
	if found {
		s.tlsIDByKey[spec.Key] = existing.Id
		s.report.Summary.TLS.Reused++
		return nil
	}
	record, err := buildPlainTLSRecord(*spec)
	if err != nil {
		return err
	}
	if err := tx.Create(&record).Error; err != nil {
		return err
	}
	s.tlsIDByKey[spec.Key] = record.Id
	s.report.Summary.TLS.Created++
	return nil
}

func (s *importState) importInboundsAndEndpoints(tx *gorm.DB, src *sourceDB, strategy Strategy) error {
	return src.eachInbound(func(row xuiInboundRow) error {
		if row.Protocol == "wireguard" {
			endpoint, warnings, err := mapWireguardEndpoint(row)
			if err != nil {
				return err
			}
			s.report.warnAll(warnings)
			if endpoint == nil {
				s.report.Summary.Endpoints.Skipped++
				return nil
			}
			imported, err := applyEndpoint(tx, endpoint, strategy, s.report)
			if err != nil {
				return err
			}
			if imported {
				s.report.Summary.Endpoints.Imported++
			}
			return nil
		}

		var tlsID uint
		var reality *realitySpec
		if spec, ok := s.realityBySource[row.ID]; ok {
			reality = spec
			tlsID = s.tlsIDByKey[spec.Key]
		} else if spec, ok := s.plainTLSBySource[row.ID]; ok {
			tlsID = s.tlsIDByKey[spec.Key]
		}
		mapped, err := mapInbound(row, tlsID, reality, s.server)
		if err != nil {
			return err
		}
		s.report.warnAll(mapped.Warnings)
		if mapped.Inbound.Type == "" {
			s.report.Summary.Inbounds.Skipped++
			return nil
		}
		dstID, imported, skipped, err := applyInbound(tx, &mapped.Inbound, strategy, s.report)
		if err != nil {
			return err
		}
		if skipped {
			s.report.Summary.Inbounds.Skipped++
			return nil
		}
		if imported {
			s.report.Summary.Inbounds.Imported++
		}
		s.inboundIDBySrc[row.ID] = dstID
		for i := range mapped.ClientRefs {
			mapped.ClientRefs[i].DstInboundID = dstID
		}
		s.clientRefs = append(s.clientRefs, mapped.ClientRefs...)
		s.report.ByInbound = append(s.report.ByInbound, InboundStat{
			SrcTag:  row.Tag,
			DstTag:  mapped.Inbound.Tag,
			Clients: len(mapped.ClientRefs),
		})
		return nil
	})
}

func applyInbound(tx *gorm.DB, inbound *model.Inbound, strategy Strategy, report *Report) (uint, bool, bool, error) {
	var existing model.Inbound
	err := tx.Where("tag = ?", inbound.Tag).First(&existing).Error
	if err != nil && !database.IsNotFound(err) {
		return 0, false, false, err
	}
	if database.IsNotFound(err) {
		if err := tx.Create(inbound).Error; err != nil {
			return 0, false, false, err
		}
		return inbound.Id, true, false, nil
	}
	report.Summary.Inbounds.Conflicts++
	switch strategy {
	case StrategySkip:
		report.warn(fmt.Sprintf("inbound %s: existing tag skipped by strategy", inbound.Tag))
		return existing.Id, false, true, nil
	case StrategyReplace:
		if err := tx.Delete(&existing).Error; err != nil {
			return 0, false, false, err
		}
		inbound.Id = 0
		if err := tx.Create(inbound).Error; err != nil {
			return 0, false, false, err
		}
		return inbound.Id, true, false, nil
	default:
		inbound.Id = existing.Id
		if err := tx.Save(inbound).Error; err != nil {
			return 0, false, false, err
		}
		return inbound.Id, true, false, nil
	}
}

func applyEndpoint(tx *gorm.DB, endpoint *model.Endpoint, strategy Strategy, report *Report) (bool, error) {
	var existing model.Endpoint
	err := tx.Where("tag = ?", endpoint.Tag).First(&existing).Error
	if err != nil && !database.IsNotFound(err) {
		return false, err
	}
	if database.IsNotFound(err) {
		return true, tx.Create(endpoint).Error
	}
	switch strategy {
	case StrategySkip:
		report.warn(fmt.Sprintf("endpoint %s: existing tag skipped by strategy", endpoint.Tag))
		return false, nil
	case StrategyReplace:
		if err := tx.Delete(&existing).Error; err != nil {
			return false, err
		}
		endpoint.Id = 0
		return true, tx.Create(endpoint).Error
	default:
		endpoint.Id = existing.Id
		return true, tx.Save(endpoint).Error
	}
}

func (s *importState) importClients(tx *gorm.DB, src *sourceDB, strategy Strategy) error {
	aggs, err := collectClientAggregates(src, s.clientRefs, s.inboundIDBySrc)
	if err != nil {
		return err
	}
	s.report.Summary.Clients.UniqueEmails = len(aggs)
	emails := make([]string, 0, len(aggs))
	for email := range aggs {
		emails = append(emails, email)
	}
	sortStrings(emails)
	for _, email := range emails {
		if err := applyClient(tx, aggs[email], strategy, s.report, s.hostname); err != nil {
			return err
		}
	}
	return nil
}

func applyClient(tx *gorm.DB, agg *clientAggregate, strategy Strategy, report *Report, hostname string) error {
	next, err := agg.toModel()
	if err != nil {
		return err
	}
	var existing model.Client
	err = tx.Where("name = ?", agg.Email).First(&existing).Error
	if err != nil && !database.IsNotFound(err) {
		return err
	}
	if database.IsNotFound(err) {
		if next.Links, err = buildClientLinks(tx, next.Config, next.Inbounds, hostname); err != nil {
			return err
		}
		report.Summary.Clients.Created++
		return tx.Create(&next).Error
	}
	switch strategy {
	case StrategySkip:
		report.warn(fmt.Sprintf("client %s: existing name skipped by strategy", agg.Email))
		return nil
	case StrategyReplace:
		next.Id = existing.Id
		next.SubSecret = existing.SubSecret
		if next.Links, err = buildClientLinks(tx, next.Config, next.Inbounds, hostname); err != nil {
			return err
		}
		report.Summary.Clients.Merged++
		return tx.Save(&next).Error
	default:
		mergedInbounds, err := mergeInboundJSON(existing.Inbounds, agg.Inbounds)
		if err != nil {
			return err
		}
		updates := map[string]any{"inbounds": mergedInbounds}
		// Regenerate local links over the merged inbound set so a newly
		// merged-in inbound also appears in the subscription, while preserving
		// the client's existing non-local (external/sub) links. Only overwrite
		// Links when we actually built them (hostname known), to avoid
		// clobbering an existing client's links with NULL on a host-less import.
		mergedLinks, err := buildMergedClientLinks(tx, existing.Config, mergedInbounds, hostname, existing.Links)
		if err != nil {
			return err
		}
		if mergedLinks != nil {
			updates["links"] = mergedLinks
		}
		report.Summary.Clients.Merged++
		return tx.Model(&existing).Updates(updates).Error
	}
}

func (s *importState) recordAudit(tx *gorm.DB, opts Options) error {
	now := time.Now().Unix()
	if opts.Now != nil {
		now = opts.Now()
	}
	details, err := auditDetails(s.report.Summary)
	if err != nil {
		return err
	}
	return tx.Create(&model.AuditEvent{
		DateTime: now,
		Actor:    "system",
		Event:    "xui_import",
		Resource: "database",
		Severity: "info",
		Details:  json.RawMessage(details),
	}).Error
}

func destinationServer(tx *gorm.DB) string {
	for _, key := range []string{"subDomain", "subListen", "webDomain", "webListen"} {
		var value string
		if err := tx.Model(model.Setting{}).Select("value").Where("key = ?", key).Scan(&value).Error; err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "127.0.0.1"
}

func WritePreImportBackup(now int64) (string, error) {
	if now == 0 {
		now = time.Now().Unix()
	}
	data, err := database.GetDb("")
	if err != nil {
		return "", fmt.Errorf("xui-import: %w", err)
	}
	dir := filepath.Dir(config.GetDBPath())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("xui-import: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("s-ui-pre-xui-import-%d.db", now))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("xui-import: %w", err)
	}
	logger.Info("xui-import: pre-import backup saved to ", path)
	prunePreImportBackups(dir, preImportBackupRetention)
	return path, nil
}

// preImportBackupRetention bounds how many s-ui-pre-xui-import-*.db files are
// kept. Every import writes one, and a slow import behind a client/proxy that
// resubmits can produce dozens, filling the db directory. Keeping the newest N
// preserves rollback while preventing unbounded growth.
const preImportBackupRetention = 10

// prunePreImportBackups removes all but the newest keep pre-import backups in
// dir. The filenames embed a fixed-width unix timestamp, so a lexical sort is
// chronological. Best-effort: failures are logged, not fatal to the import.
func prunePreImportBackups(dir string, keep int) {
	matches, err := filepath.Glob(filepath.Join(dir, "s-ui-pre-xui-import-*.db"))
	if err != nil || len(matches) <= keep {
		return
	}
	sort.Strings(matches)
	for _, old := range matches[:len(matches)-keep] {
		if err := os.Remove(old); err != nil {
			logger.Warning("xui-import: failed to prune old pre-import backup ", old, ": ", err)
		}
	}
}

func sortStrings(values []string) {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
}
