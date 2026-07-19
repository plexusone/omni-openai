package omnidevx

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	core "github.com/plexusone/omnidevx-core"
)

func TestSourceDescriptor(t *testing.T) {
	c, err := New(Config{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	want := core.Source{Provider: "openai", Product: "codex-cli"}
	if got := c.Source(); got != want {
		t.Errorf("Source(): got %+v, want %+v", got, want)
	}
}

func TestCollectEmptyHome(t *testing.T) {
	// No state DB and no sessions dir: an empty result, not an error.
	c, err := New(Config{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), core.CollectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 0 || len(result.Diagnostics) != 0 {
		t.Errorf("empty home: got %d events, %d diagnostics; want 0/0",
			len(result.Events), len(result.Diagnostics))
	}
}

func TestSchemaDriftFallsBackToRollouts(t *testing.T) {
	// A threads table missing expected columns must degrade to rollout-only
	// collection with a diagnostic, not fail the run.
	home := buildTestHome(t, false)
	db, err := sql.Open("sqlite", filepath.Join(home, "state_9.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE threads (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	c, err := New(Config{Dir: home})
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.Collect(context.Background(), core.CollectRequest{})
	if err != nil {
		t.Fatal(err)
	}

	drift := false
	for _, d := range result.Diagnostics {
		if d.Severity == core.SeverityWarning && d.Path != "" && filepath.Base(d.Path) == "state_9.sqlite" {
			drift = true
		}
	}
	if !drift {
		t.Errorf("expected schema-drift diagnostic, got %+v", result.Diagnostics)
	}
	// Rollout events must still be collected.
	if counts := countByType(result.Events); counts[core.EventPromptSubmitted] != 1 {
		t.Errorf("rollout fallback: prompts got %d, want 1", counts[core.EventPromptSubmitted])
	}
}

func TestNewestStateDBPicksHighest(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"state_3.sqlite", "state_5.sqlite", "state_4.sqlite"} {
		db, err := sql.Open("sqlite", filepath.Join(home, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`CREATE TABLE t (x INTEGER)`); err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}
	got, err := newestStateDB(home)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "state_5.sqlite" {
		t.Errorf("newestStateDB: got %s, want state_5.sqlite", got)
	}
}
