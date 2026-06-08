package importxui

import (
	"database/sql"
	"fmt"
	"strings"
)

// 3x-ui forks diverge in their `inbounds` / `client_traffics` columns: vanilla
// mhsanaei has neither `all_time` nor `last_online`, while normalized forks add
// `traffic_reset`, `last_traffic_reset_time`, `node_id` and friends. A fixed
// SELECT list breaks on every schema that lacks one of the named columns
// ("no such column: all_time"). The helpers below let a reader project exactly
// the columns it wants, substituting a literal default for any the source
// database does not define, so the importer works across forks.

// quoteIdent wraps a SQLite identifier in double quotes, escaping embedded
// quotes. Identifiers handled here come from a fixed allow-list of table and
// column names, but quoting keeps the generated SQL well-formed regardless.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// tableColumns returns the set of column names defined on table, lower-cased.
// `SELECT * ... LIMIT 0` exposes the live column list without reading any rows
// and works uniformly across SQLite drivers, unlike parameterised PRAGMA calls.
// Names are normalized to lower case because SQLite identifiers are
// case-insensitive: a fork that declares `ExpiryTime` must still match the
// lower-case spec name, otherwise selectColumns would wrongly substitute a
// default for a column that actually exists.
func tableColumns(db *sql.DB, table string) (map[string]struct{}, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", quoteIdent(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[strings.ToLower(name)] = struct{}{}
	}
	return set, nil
}

// columnSpec pairs a column the importer wants to read with the SQL literal to
// substitute when the source schema omits it.
type columnSpec struct {
	name        string
	missingExpr string
}

// selectColumns builds a SELECT projection that reads each wanted column when
// present and falls back to its missingExpr (aliased to the column name) when
// the source table does not define it. The projection order matches specs, so
// the caller's positional Scan stays valid regardless of which columns exist.
func selectColumns(present map[string]struct{}, specs []columnSpec) string {
	parts := make([]string, len(specs))
	for i, spec := range specs {
		if _, ok := present[strings.ToLower(spec.name)]; ok {
			parts[i] = quoteIdent(spec.name)
		} else {
			parts[i] = spec.missingExpr + " AS " + quoteIdent(spec.name)
		}
	}
	return strings.Join(parts, ", ")
}
