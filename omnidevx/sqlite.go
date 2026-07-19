package omnidevx

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	core "github.com/plexusone/omnidevx-core"
	_ "modernc.org/sqlite" // pure-Go sqlite driver, registered as "sqlite"
)

// thread is one row of the Codex threads table that OmniDevX reads.
// Content-bearing columns (title, first_user_message, preview) are never
// selected.
type thread struct {
	ID              string
	CreatedAt       int64
	UpdatedAt       int64
	CWD             string
	GitBranch       sql.NullString
	GitOriginURL    sql.NullString
	Model           sql.NullString
	ReasoningEffort sql.NullString
	CLIVersion      string
	Source          string
	TokensUsed      int64
}

const threadsQuery = `SELECT id, created_at, updated_at, cwd, git_branch,
	git_origin_url, model, reasoning_effort, cli_version, source, tokens_used
	FROM threads`

// collectThreads emits session start/end events from the newest state
// database, returning the set of thread IDs seen so rollout parsing does not
// duplicate their session events. A missing database is not an error — the
// collector falls back to rollout files alone.
func (c *Collector) collectThreads(ctx context.Context, req core.CollectRequest, result *core.CollectionResult) (map[string]bool, error) {
	seen := map[string]bool{}

	dbPath, err := newestStateDB(c.dir)
	if err != nil {
		return nil, err
	}
	if dbPath == "" {
		return seen, nil
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("codex: open state db %s: %w", dbPath, err)
	}
	defer db.Close() //nolint:errcheck // read-only handle

	rows, err := db.QueryContext(ctx, threadsQuery)
	if err != nil {
		// Schema drift (missing columns) degrades to rollout-only collection
		// rather than failing the run.
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
			Severity: core.SeverityWarning,
			Message:  fmt.Sprintf("query threads (schema drift?): %v; falling back to rollout files", err),
			Path:     dbPath,
		})
		return seen, nil
	}
	defer rows.Close() //nolint:errcheck // rows fully consumed below

	for rows.Next() {
		var t thread
		if err := rows.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt, &t.CWD, &t.GitBranch,
			&t.GitOriginURL, &t.Model, &t.ReasoningEffort, &t.CLIVersion, &t.Source, &t.TokensUsed); err != nil {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
				Severity: core.SeverityWarning,
				Message:  fmt.Sprintf("scan thread row: %v", err),
				Path:     dbPath,
			})
			continue
		}
		seen[t.ID] = true
		result.Events = append(result.Events, threadEvents(t, req)...)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("codex: iterate threads: %w", err)
	}
	return seen, nil
}

// newestStateDB returns the state_N.sqlite path with the highest N, or ""
// when none exists.
func newestStateDB(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "state_*.sqlite"))
	if err != nil {
		return "", fmt.Errorf("codex: glob state db: %w", err)
	}
	if len(matches) == 0 {
		return "", nil
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func threadEvents(t thread, req core.CollectRequest) []core.Event {
	ctx := core.EventContext{
		SessionID:  t.ID,
		Workspace:  t.CWD,
		GitBranch:  t.GitBranch.String,
		Repository: normalizeRepoURL(t.GitOriginURL.String),
	}

	var events []core.Event
	created := time.Unix(t.CreatedAt, 0).UTC()
	if req.Period.Contains(created) {
		attrs := map[string]any{}
		if t.Model.String != "" {
			attrs[core.AttrModel] = t.Model.String
		}
		if t.ReasoningEffort.String != "" {
			attrs[core.AttrReasoningEffort] = t.ReasoningEffort.String
		}
		if t.CLIVersion != "" {
			attrs[core.AttrClientVersion] = t.CLIVersion
		}
		if t.Source != "" {
			attrs[core.AttrSessionSource] = t.Source
		}
		if len(attrs) == 0 {
			attrs = nil
		}
		events = append(events, core.Event{
			ID:         "codex-cli:" + t.ID + ":start",
			Type:       core.EventSessionStarted,
			Timestamp:  created,
			Source:     source,
			Context:    ctx,
			Attributes: attrs,
			Provenance: provenance(),
		})
	}

	updated := time.Unix(t.UpdatedAt, 0).UTC()
	if req.Period.Contains(updated) {
		events = append(events, core.Event{
			ID:        "codex-cli:" + t.ID + ":end",
			Type:      core.EventSessionEnded,
			Timestamp: updated,
			Source:    source,
			Context:   ctx,
			Attributes: map[string]any{
				core.AttrSessionTotalTokens: t.TokensUsed,
			},
			Provenance: provenance(),
		})
	}
	return events
}
