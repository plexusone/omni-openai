package omnidevx

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	core "github.com/plexusone/omnidevx-core"
)

// buildTestHome assembles a fake ~/.codex containing the testdata rollout
// and, optionally, a state DB with one indexed thread.
func buildTestHome(t *testing.T, withDB bool) string {
	t.Helper()
	home := t.TempDir()

	rolloutDir := filepath.Join(home, "sessions", "2026", "07", "01")
	if err := os.MkdirAll(rolloutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "rollout-2026-07-01T10-00-00-thread01.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // G703 false positive: rolloutDir is t.TempDir() + literal segments
	if err := os.WriteFile(filepath.Join(rolloutDir, "rollout-2026-07-01T10-00-00-thread01.jsonl"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	if !withDB {
		return home
	}

	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck // test fixture
	_, err = db.Exec(`CREATE TABLE threads (
		id TEXT PRIMARY KEY, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
		cwd TEXT NOT NULL, git_branch TEXT, git_origin_url TEXT, model TEXT,
		reasoning_effort TEXT, cli_version TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '', tokens_used INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		t.Fatal(err)
	}
	// 2026-07-02T09:00:00Z .. 2026-07-02T10:00:00Z
	_, err = db.Exec(`INSERT INTO threads VALUES
		('thread-db-01', 1782032400, 1782036000, '/Users/testuser/src/example',
		 'main', 'git@github.com:example/project.git', 'gpt-5.2-codex', 'high',
		 '0.9.0', 'cli', 123456)`)
	if err != nil {
		t.Fatal(err)
	}
	return home
}

func collect(t *testing.T, home string) *core.CollectionResult {
	t.Helper()
	c, err := New(Config{Dir: home})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), core.CollectRequest{
		Subject: core.SubjectRef{PersonID: "person:test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func countByType(events []core.Event) map[core.EventType]int {
	counts := map[core.EventType]int{}
	for _, e := range events {
		counts[e.Type]++
	}
	return counts
}

func TestCollectRolloutOnly(t *testing.T) {
	result := collect(t, buildTestHome(t, false))

	counts := countByType(result.Events)
	want := map[core.EventType]int{
		core.EventSessionStarted:   1, // from session_meta (no DB)
		core.EventPromptSubmitted:  1,
		core.EventTaskStarted:      1,
		core.EventTaskCompleted:    1,
		core.EventToolCompleted:    2, // function_call_output + custom_tool_call_output
		core.EventPatchApplied:     1,
		core.EventUsageRecorded:    1,
		core.EventMessageCompleted: 1,
	}
	for eventType, n := range want {
		if counts[eventType] != n {
			t.Errorf("event type %s: got %d, want %d", eventType, counts[eventType], n)
		}
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics: got %d, want 1 (unparseable line): %+v", len(result.Diagnostics), result.Diagnostics)
	}

	for _, e := range result.Events {
		if e.Context.SessionID != "thread01" {
			t.Errorf("event %s: session: got %q", e.ID, e.Context.SessionID)
		}
		if e.Context.Repository != "github.com/example/project" {
			t.Errorf("event %s: repository: got %q", e.ID, e.Context.Repository)
		}
		if e.Subject.PersonID != "person:test" {
			t.Errorf("event %s: subject not stamped", e.ID)
		}
	}
}

func TestCollectWithStateDB(t *testing.T) {
	result := collect(t, buildTestHome(t, true))

	counts := countByType(result.Events)
	// thread-db-01 adds session start + end from SQLite; thread01's start
	// still comes from its rollout since it is not in the DB.
	if counts[core.EventSessionStarted] != 2 {
		t.Errorf("session starts: got %d, want 2", counts[core.EventSessionStarted])
	}
	if counts[core.EventSessionEnded] != 1 {
		t.Errorf("session ends: got %d, want 1", counts[core.EventSessionEnded])
	}

	var dbStart, dbEnd *core.Event
	for i, e := range result.Events {
		if e.Context.SessionID != "thread-db-01" {
			continue
		}
		switch e.Type {
		case core.EventSessionStarted:
			dbStart = &result.Events[i]
		case core.EventSessionEnded:
			dbEnd = &result.Events[i]
		}
	}
	if dbStart == nil || dbEnd == nil {
		t.Fatal("missing session events for thread-db-01")
	}
	if got := dbStart.Attributes[core.AttrModel]; got != "gpt-5.2-codex" {
		t.Errorf("model: got %v", got)
	}
	if got := dbStart.Attributes[core.AttrReasoningEffort]; got != "high" {
		t.Errorf("reasoning effort: got %v", got)
	}
	if dbStart.Context.Repository != "github.com/example/project" {
		t.Errorf("ssh remote not normalized: %q", dbStart.Context.Repository)
	}
	if got := dbEnd.Attributes[core.AttrSessionTotalTokens]; got != int64(123456) {
		t.Errorf("session tokens: got %v (%T)", got, got)
	}
}

func TestUsageAndToolAttribution(t *testing.T) {
	result := collect(t, buildTestHome(t, false))

	toolNames := map[string]bool{}
	for _, e := range result.Events {
		switch e.Type {
		case core.EventUsageRecorded:
			// last_token_usage (per-turn delta), not the cumulative total.
			if got := e.Attributes[core.AttrInputTokens]; got != int64(1000) {
				t.Errorf("usage input tokens: got %v, want 1000 (last, not total)", got)
			}
			if got := e.Attributes[core.AttrCacheReadTokens]; got != int64(800) {
				t.Errorf("usage cached tokens: got %v, want 800", got)
			}
		case core.EventToolCompleted:
			if name, ok := e.Attributes[core.AttrTool].(string); ok {
				toolNames[name] = true
			}
		case core.EventTaskCompleted:
			if got := e.Attributes[core.AttrDurationMS]; got != int64(8000) {
				t.Errorf("task duration: got %v, want 8000", got)
			}
			if got := e.Attributes[core.AttrTimeToFirstTokenMS]; got != int64(1200) {
				t.Errorf("time to first token: got %v, want 1200", got)
			}
		}
	}
	if !toolNames["exec_command"] || !toolNames["apply_patch"] {
		t.Errorf("tool attribution incomplete: %v", toolNames)
	}

	// Privacy: no content-like attributes may leak.
	for _, e := range result.Events {
		for key := range e.Attributes {
			switch key {
			case "message", "arguments", "output", "stdout", "stderr", "input", "last_agent_message":
				t.Errorf("event %s leaks content attribute %q", e.ID, key)
			}
		}
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/x/y.git":  "github.com/x/y",
		"git@github.com:x/y.git":      "github.com/x/y",
		"ssh://git@github.com/x/y":    "github.com/x/y",
		"http://gitlab.com/a/b/c.git": "gitlab.com/a/b/c",
		"":                            "",
	}
	for in, want := range cases {
		if got := normalizeRepoURL(in); got != want {
			t.Errorf("normalizeRepoURL(%q): got %q, want %q", in, got, want)
		}
	}
}
