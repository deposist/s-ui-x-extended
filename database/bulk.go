package database

import (
	"reflect"
	"sync"

	"gorm.io/gorm"
)

// SQLite (mattn/go-sqlite3) ships with the historical compile-time limit
// SQLITE_MAX_VARIABLE_NUMBER = 999. Newer SQLite (3.32+) raised the default
// to 32766, but the wired-in mattn build still uses 999, so we have to
// honour the lower bound. We pick a conservative 800-placeholder budget per
// statement to leave room for GORM's bookkeeping placeholders (ON CONFLICT
// targets, returning clauses, optional WHERE in upserts).
const sqliteVariableBudget = 800

// minBatchSize avoids degenerate batch=0 when a model exposes more than the
// budget worth of columns (which should not happen for any current model,
// but the guard keeps the helper robust).
const minBatchSize = 1

var (
	schemaCacheMu sync.Mutex
	schemaCache   = map[reflect.Type]int{}
)

// SafeSQLiteBatchSize returns the largest batch size that keeps a multi-row
// INSERT within SQLite's variable budget for the given model. Pass either a
// pointer to a model value (`&model.Client{}`) or a slice of models.
func SafeSQLiteBatchSize(db *gorm.DB, modelValue any) int {
	cols := countModelColumns(db, modelValue)
	if cols <= 0 {
		return minBatchSize
	}
	batch := sqliteVariableBudget / cols
	if batch < minBatchSize {
		return minBatchSize
	}
	return batch
}

func countModelColumns(db *gorm.DB, modelValue any) int {
	t := reflect.TypeOf(modelValue)
	if t == nil {
		return 0
	}
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}

	schemaCacheMu.Lock()
	if cached, ok := schemaCache[t]; ok {
		schemaCacheMu.Unlock()
		return cached
	}
	schemaCacheMu.Unlock()

	cols := parseSchemaColumns(db, modelValue)
	if cols <= 0 {
		// Fall back to a reflective field count so a model without a
		// configured GORM session still gets a sane batch size.
		cols = reflectColumns(t)
	}

	schemaCacheMu.Lock()
	schemaCache[t] = cols
	schemaCacheMu.Unlock()
	return cols
}

func parseSchemaColumns(db *gorm.DB, modelValue any) int {
	if db == nil {
		return 0
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(modelValue); err != nil || stmt.Schema == nil {
		return 0
	}
	count := 0
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		// Skip association fields (foreign keys backed by relations are
		// already counted via their own DBName entries).
		if field.IgnoreMigration {
			continue
		}
		switch field.FieldType.Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
			// Allow embedded JSON/RawMessage and time-like columns: GORM
			// lists them as Fields with DBName, which is what we want.
		}
		count++
	}
	return count
}

func reflectColumns(t reflect.Type) int {
	if t.Kind() != reflect.Struct {
		return 0
	}
	count := 0
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if tag := f.Tag.Get("gorm"); tag == "-" {
			continue
		}
		count++
	}
	return count
}

// CreateInBatchesSafe inserts slice into the given table while keeping each
// generated INSERT below SQLite's variable budget. Pass a pointer to a slice
// (`*[]model.X`) or a non-nil slice value. Empty slices are a no-op.
func CreateInBatchesSafe(tx *gorm.DB, slice any) error {
	if tx == nil {
		return nil
	}
	v := reflect.ValueOf(slice)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}
	elem := reflect.New(v.Type().Elem()).Interface()
	batch := SafeSQLiteBatchSize(tx, elem)
	return tx.CreateInBatches(slice, batch).Error
}

// SaveInBatchesSafe runs gorm.Save in chunks small enough for SQLite. Save
// is upsert-style (INSERT OR REPLACE), so the same variable budget applies
// as for CreateInBatches. Use for slices of pointers or values.
func SaveInBatchesSafe(tx *gorm.DB, slice any) error {
	if tx == nil {
		return nil
	}
	v := reflect.ValueOf(slice)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		// Not a slice: forward to the regular Save (single row, no risk).
		return tx.Save(slice).Error
	}
	if v.Len() == 0 {
		return nil
	}
	elem := reflect.New(v.Type().Elem()).Interface()
	batch := SafeSQLiteBatchSize(tx, elem)
	for start := 0; start < v.Len(); start += batch {
		end := start + batch
		if end > v.Len() {
			end = v.Len()
		}
		chunk := v.Slice(start, end)
		if err := tx.Save(chunk.Interface()).Error; err != nil {
			return err
		}
	}
	return nil
}
