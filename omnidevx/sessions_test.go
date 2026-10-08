package omnidevx

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/plexusone/omnidevx-core/sessions"
)

var base = time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)

const fullThreadsSchema = `CREATE TABLE threads (
	id TEXT PRIMARY KEY, rollout_path TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
	cwd TEXT NOT NULL, title TEXT, name TEXT, first_user_message TEXT, archived INTEGER DEFAULT 0,
	git_branch TEXT, git_origin_url TEXT, created_at_ms INTEGER, updated_at_ms INTEGER)`

func openDB(t *testing.T, dir, schema string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return db
}

type thread0 struct {
	id, cwd, title, name, first string
	archived                    int
	updated                     time.Time
	rollout                     string
}

func insertThread(t *testing.T, db *sql.DB, th thread0) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO threads (id, rollout_path, created_at, updated_at, cwd, title, name,
		first_user_message, archived, git_branch, git_origin_url, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		th.id, th.rollout, base.Unix(), th.updated.Unix(), th.cwd, th.title, th.name, th.first, th.archived,
		"main", "git@github.com:example/app.git", base.UnixMilli(), th.updated.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
}

func rolloutLine(t *testing.T, ts time.Time, role, text string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"type": "response_item", "timestamp": ts.Format(time.RFC3339Nano),
		"payload": map[string]any{"type": "message", "role": role,
			"content": []any{map[string]any{"type": "input_text", "text": text}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeRollout(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func newSessionReader(t *testing.T, dir string) *SessionReader {
	t.Helper()
	r, err := NewSessionReader(Config{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r.processes = func(context.Context) ([]procInfo, error) { return nil, nil }
	return r
}

func TestSessionReaderListsThreads(t *testing.T) {
	home := t.TempDir()
	db := openDB(t, home, fullThreadsSchema)
	rollout := writeRollout(t, home, "rollout-a.jsonl",
		rolloutLine(t, base, "user", "<environment_context><cwd>/x</cwd></environment_context>"),
		rolloutLine(t, base.Add(time.Minute), "user", "wire up the GitHub app"),
		rolloutLine(t, base.Add(2*time.Minute), "assistant", "on it"),
		rolloutLine(t, base.Add(3*time.Minute), "user", "<turn_aborted>user interrupted</turn_aborted>"),
		rolloutLine(t, base.Add(5*time.Minute), "user", "continue with auth"),
		rolloutLine(t, base.Add(90*time.Minute), "assistant", "done"),
	)
	updated := base.Add(2 * time.Hour)
	insertThread(t, db, thread0{id: "0199aaaa-1111-7222-8333-444455556666", cwd: "/Users/example/src/app",
		title: "title col", name: "GitHub App auth", first: "wire up the GitHub app", updated: updated, rollout: rollout})

	got, diags, err := newSessionReader(t, home).List(context.Background(), sessions.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 || len(got) != 1 {
		t.Fatalf("sessions=%d diags=%+v", len(got), diags)
	}
	s := got[0]
	if s.Harness != sessions.HarnessCodex || s.CWD != "/Users/example/src/app" {
		t.Errorf("identity/cwd = %s %q", s.Harness, s.CWD)
	}
	if !s.LastActivityAt.Equal(updated) {
		t.Errorf("LastActivityAt = %v, want %v (database updated time)", s.LastActivityAt, updated)
	}
	if want := base.Add(5 * time.Minute); !s.LastHumanActivityAt.Equal(want) {
		t.Errorf("LastHumanActivityAt = %v, want %v (injected context and aborted-turn notices are not human)", s.LastHumanActivityAt, want)
	}
	if s.Title != "GitHub App auth" || s.TitleSource != sessions.TitleHarness {
		t.Errorf("title = %q (%s), want name column first", s.Title, s.TitleSource)
	}
	if len(s.RecentPrompts) != 2 || s.RecentPrompts[1].Text != "continue with auth" {
		t.Errorf("RecentPrompts = %+v", s.RecentPrompts)
	}
	if s.GitBranch != "main" || s.GitOrigin == "" {
		t.Errorf("git = %q %q", s.GitBranch, s.GitOrigin)
	}
	if s.State != sessions.StateUnknown {
		t.Errorf("State = %s, want unknown (no liveness signal)", s.State)
	}
	wantArgv := "codex resume 0199aaaa-1111-7222-8333-444455556666"
	if strings.Join(s.Resume.Argv, " ") != wantArgv || s.Resume.Dir != s.CWD {
		t.Errorf("Resume = %+v", s.Resume)
	}
}

func TestSessionReaderTitlePrecedenceAndNoContent(t *testing.T) {
	home := t.TempDir()
	db := openDB(t, home, fullThreadsSchema)
	insertThread(t, db, thread0{id: "0199bbbb-1111-7222-8333-444455556666", cwd: "/Users/example/src/app",
		title: "title col", updated: base})
	insertThread(t, db, thread0{id: "0199cccc-1111-7222-8333-444455556666", cwd: "/Users/example/src/app",
		first: "only the first message", updated: base})
	insertThread(t, db, thread0{id: "0199dddd-1111-7222-8333-444455556666", cwd: "/Users/example/src/app", updated: base})

	r := newSessionReader(t, home)
	got, _, err := r.List(context.Background(), sessions.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	titles := map[string]string{}
	for _, s := range got {
		titles[s.ID[:8]] = s.Title + "|" + s.TitleSource
	}
	want := map[string]string{
		"0199bbbb": "title col|harness",
		"0199cccc": "only the first message|first-prompt",
		"0199dddd": "app|cwd",
	}
	for k, v := range want {
		if titles[k] != v {
			t.Errorf("title[%s] = %q, want %q", k, titles[k], v)
		}
	}

	got, _, err = r.List(context.Background(), sessions.ListOptions{NoContent: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range got {
		if s.Title != "app" || len(s.RecentPrompts) != 0 {
			t.Errorf("NoContent leaked %q / %+v", s.Title, s.RecentPrompts)
		}
	}
}

func TestSessionReaderArchivedAndSince(t *testing.T) {
	home := t.TempDir()
	db := openDB(t, home, fullThreadsSchema)
	insertThread(t, db, thread0{id: "0199eeee-1111-7222-8333-444455556666", cwd: "/x", archived: 1, updated: base})
	insertThread(t, db, thread0{id: "0199ffff-1111-7222-8333-444455556666", cwd: "/x", updated: base.Add(-48 * time.Hour)})

	r := newSessionReader(t, home)
	got, _, err := r.List(context.Background(), sessions.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID[:8] != "0199ffff" {
		t.Fatalf("default list = %+v, want only the unarchived thread", got)
	}
	got, _, err = r.List(context.Background(), sessions.ListOptions{IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("IncludeArchived list has %d sessions, want 2", len(got))
	}
	got, _, err = r.List(context.Background(), sessions.ListOptions{Since: base.Add(-time.Hour), IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Archived {
		t.Fatalf("Since list = %+v", got)
	}
}

func TestSessionReaderSchemaDrift(t *testing.T) {
	// A Codex version without the optional columns still lists sessions.
	home := t.TempDir()
	db := openDB(t, home,
		`CREATE TABLE threads (id TEXT PRIMARY KEY, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, cwd TEXT NOT NULL)`)
	if _, err := db.Exec(`INSERT INTO threads VALUES ('0199abab-1111-7222-8333-444455556666', ?, ?, '/Users/example/src/app')`,
		base.Unix(), base.Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}

	got, _, err := newSessionReader(t, home).List(context.Background(), sessions.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("sessions = %+v", got)
	}
	if !got[0].LastActivityAt.Equal(base.Add(time.Hour)) {
		t.Errorf("LastActivityAt = %v, want seconds fallback", got[0].LastActivityAt)
	}

	// A missing required column is an error, not a silent empty list.
	home2 := t.TempDir()
	openDB(t, home2, `CREATE TABLE threads (id TEXT PRIMARY KEY)`)
	if _, _, err := newSessionReader(t, home2).List(context.Background(), sessions.ListOptions{}); err == nil {
		t.Fatal("expected error when required columns are missing")
	}
}

func TestNewestStateDBIsNumeric(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"state_9.sqlite", "state_10.sqlite", "state_2.sqlite", "state_10.sqlite-wal", "logs_2.sqlite"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := newestStateDBNumeric(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "state_10.sqlite" {
		t.Fatalf("got %s, want state_10.sqlite", got)
	}
}

func TestSessionReaderMissingDatabase(t *testing.T) {
	if _, _, err := newSessionReader(t, t.TempDir()).List(context.Background(), sessions.ListOptions{}); err == nil {
		t.Fatal("expected error when no state database exists")
	}
}

func TestSessionReaderRunningFromProcessList(t *testing.T) {
	home := t.TempDir()
	db := openDB(t, home, fullThreadsSchema)
	const live = "0199aaaa-1111-7222-8333-444455556666"
	insertThread(t, db, thread0{id: live, cwd: "/x", updated: base})
	insertThread(t, db, thread0{id: "0199bbbb-1111-7222-8333-444455556666", cwd: "/x", updated: base})

	r := newSessionReader(t, home)
	r.processes = func(context.Context) ([]procInfo, error) {
		return []procInfo{
			{PID: 4242, Command: "/usr/local/bin/codex resume " + live},
			// Another tool mentioning the id is not a codex process.
			{PID: 4343, Command: "vim notes-0199bbbb-1111-7222-8333-444455556666.md"},
		}, nil
	}
	got, _, err := r.List(context.Background(), sessions.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range got {
		switch s.ID {
		case live:
			if s.State != sessions.StateRunning || s.Runtime == nil || s.Runtime.PID != 4242 {
				t.Errorf("live session = %s %+v", s.State, s.Runtime)
			}
		default:
			if s.State != sessions.StateUnknown {
				t.Errorf("other session state = %s, want unknown", s.State)
			}
		}
	}
}

func TestSessionReaderMissingRolloutIsNotFatal(t *testing.T) {
	home := t.TempDir()
	db := openDB(t, home, fullThreadsSchema)
	insertThread(t, db, thread0{id: "0199cdcd-1111-7222-8333-444455556666", cwd: "/x", updated: base,
		rollout: filepath.Join(home, "gone.jsonl")})

	got, diags, err := newSessionReader(t, home).List(context.Background(), sessions.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(diags) != 0 {
		t.Fatalf("sessions=%d diags=%+v, want the session without a diagnostic", len(got), diags)
	}
	if !got[0].LastHumanActivityAt.IsZero() {
		t.Errorf("LastHumanActivityAt = %v, want zero without a rollout", got[0].LastHumanActivityAt)
	}
}

func TestRolloutTailGrowsPastAgentOutput(t *testing.T) {
	home := t.TempDir()
	pad := strings.Repeat("x", 1<<10)
	lines := []string{rolloutLine(t, base, "user", "kick off a long run")}
	for i := 0; i < 1200; i++ {
		lines = append(lines, rolloutLine(t, base.Add(time.Minute), "assistant", pad))
	}
	path := writeRollout(t, home, "long.jsonl", lines...)

	last, prompts, err := readRolloutTail(path)
	if err != nil {
		t.Fatal(err)
	}
	if !last.Equal(base) || len(prompts) != 1 {
		t.Fatalf("last=%v prompts=%+v, want the prompt before ~1.2MB of agent output", last, prompts)
	}
}

func TestIsCodexCommand(t *testing.T) {
	cases := map[string]bool{
		"/usr/local/bin/codex resume abc": true,
		"codex":                           true,
		"/opt/homebrew/bin/codex -m x":    true,
		"vim codex-notes.md":              false,
		"/usr/bin/codexd --flag":          false,
	}
	for cmd, want := range cases {
		if got := isCodexCommand(cmd); got != want {
			t.Errorf("isCodexCommand(%q) = %v, want %v", cmd, got, want)
		}
	}
}
