package omnidevx

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	core "github.com/plexusone/omnidevx-core"
	"github.com/plexusone/omnidevx-core/sessions"
)

const (
	// rolloutTailBytes is the initial tail read of a rollout file. It grows
	// up to maxRolloutTailBytes while no human prompt is found, because a
	// long agent run is mostly tool calls.
	rolloutTailBytes    = 256 << 10
	maxRolloutTailBytes = 8 << 20
	recentPrompts       = 3
	promptWidth         = 300
)

// SessionReader lists Codex CLI sessions from the local state database and
// rollout files. It reads titles and prompt text, so it is separate from
// the metadata-only Collector; see the sessions package for the
// content-access contract.
type SessionReader struct {
	dir string
	// processes returns the command lines of running processes. Replaceable
	// in tests.
	processes func(ctx context.Context) ([]procInfo, error)
}

var _ sessions.Reader = (*SessionReader)(nil)

type procInfo struct {
	PID     int
	Command string
}

// NewSessionReader returns a SessionReader for the given config.
func NewSessionReader(cfg Config) (*SessionReader, error) {
	dir := cfg.Dir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("codex: resolve home directory: %w", err)
		}
		dir = filepath.Join(home, ".codex")
	}
	return &SessionReader{dir: dir, processes: listProcesses}, nil
}

// Harness implements sessions.Reader.
func (r *SessionReader) Harness() sessions.Harness { return sessions.HarnessCodex }

// threadColumns are the threads-table columns the reader can use. The
// first group is required; the rest degrade to NULL when a Codex version
// does not have them.
var (
	requiredThreadColumns = []string{"id", "cwd", "created_at", "updated_at"}
	optionalThreadColumns = []string{
		"rollout_path", "name", "title", "first_user_message", "archived",
		"git_branch", "git_origin_url", "created_at_ms", "updated_at_ms",
	}
)

// List implements sessions.Reader.
func (r *SessionReader) List(ctx context.Context, opts sessions.ListOptions) ([]sessions.Session, []core.Diagnostic, error) {
	dbPath, err := newestStateDBNumeric(r.dir)
	if err != nil {
		return nil, nil, err
	}
	if dbPath == "" {
		return nil, nil, fmt.Errorf("codex: no state database under %s", r.dir)
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, nil, fmt.Errorf("codex: open state db %s: %w", dbPath, err)
	}
	defer db.Close() //nolint:errcheck // read-only handle

	query, err := buildThreadsQuery(ctx, db)
	if err != nil {
		return nil, nil, fmt.Errorf("codex: %w (schema drift?)", err)
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("codex: query threads: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows fully consumed below

	running := r.runningSessions(ctx)

	var (
		out   []sessions.Session
		diags []core.Diagnostic
	)
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return out, diags, err
		}
		var (
			id, cwd                             string
			createdAt, updatedAt                int64
			rolloutPath, name, title, firstUser sql.NullString
			archived                            sql.NullInt64
			gitBranch, gitOrigin                sql.NullString
			createdAtMS, updatedAtMS            sql.NullInt64
		)
		if err := rows.Scan(&id, &cwd, &createdAt, &updatedAt, &rolloutPath, &name, &title,
			&firstUser, &archived, &gitBranch, &gitOrigin, &createdAtMS, &updatedAtMS); err != nil {
			diags = append(diags, core.Diagnostic{
				Severity: core.SeverityWarning,
				Message:  fmt.Sprintf("scan thread row: %v", err),
				Path:     dbPath,
			})
			continue
		}
		if archived.Int64 != 0 && !opts.IncludeArchived {
			continue
		}

		s := sessions.Session{
			Harness:        sessions.HarnessCodex,
			ID:             id,
			CWD:            cwd,
			GitBranch:      gitBranch.String,
			GitOrigin:      normalizeRepoURL(gitOrigin.String),
			CreatedAt:      epoch(createdAt, createdAtMS),
			LastActivityAt: epoch(updatedAt, updatedAtMS),
			Archived:       archived.Int64 != 0,
			State:          sessions.StateUnknown,
			Resume:         sessions.ResumeSpec{Argv: []string{"codex", "resume", id}, Dir: cwd},
		}
		if !opts.Since.IsZero() && s.LastActivityAt.Before(opts.Since) {
			continue
		}
		if pid, ok := running[id]; ok {
			s.State, s.Runtime = sessions.StateRunning, &sessions.Runtime{PID: pid}
		}

		var prompts []sessions.Prompt
		var lastHuman time.Time
		if rolloutPath.String != "" {
			lastHuman, prompts, err = readRolloutTail(rolloutPath.String)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				diags = append(diags, core.Diagnostic{
					Severity: core.SeverityWarning,
					Message:  fmt.Sprintf("read rollout: %v", err),
					Path:     rolloutPath.String,
				})
			}
		}
		s.LastHumanActivityAt = lastHuman

		if !opts.NoContent {
			s.RecentPrompts = prompts
			switch {
			case strings.TrimSpace(name.String) != "":
				s.Title, s.TitleSource = sessions.Truncate(name.String, 80), sessions.TitleHarness
			case strings.TrimSpace(title.String) != "":
				s.Title, s.TitleSource = sessions.Truncate(title.String, 80), sessions.TitleHarness
			case strings.TrimSpace(firstUser.String) != "":
				s.Title, s.TitleSource = sessions.Truncate(firstUser.String, 80), sessions.TitleFirstPrompt
			}
		}
		if s.Title == "" {
			s.Title, s.TitleSource = filepath.Base(cwd), sessions.TitleCWD
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return out, diags, fmt.Errorf("codex: iterate threads: %w", err)
	}
	return out, diags, nil
}

// buildThreadsQuery selects the columns this Codex version has, using NULL
// for optional columns it lacks. A missing required column is an error.
func buildThreadsQuery(ctx context.Context, db *sql.DB) (string, error) {
	have, err := tableColumns(ctx, db, "threads")
	if err != nil {
		return "", err
	}
	for _, c := range requiredThreadColumns {
		if !have[c] {
			return "", fmt.Errorf("threads table lacks required column %q", c)
		}
	}
	cols := append([]string{}, requiredThreadColumns...)
	for _, c := range optionalThreadColumns {
		if have[c] {
			cols = append(cols, c)
		} else {
			cols = append(cols, "NULL AS "+c)
		}
	}
	// Column order matches the Scan in List: required, then optional in the
	// order rollout_path, name, title, first_user_message, archived,
	// git_branch, git_origin_url, created_at_ms, updated_at_ms.
	return "SELECT " + strings.Join(cols, ", ") + " FROM threads", nil
}

func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", table, err)
	}
	defer rows.Close() //nolint:errcheck // rows fully consumed below
	have := map[string]bool{}
	for rows.Next() {
		var (
			cid        int
			name, typ  string
			notNull    int
			dflt       sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &primaryKey); err != nil {
			return nil, fmt.Errorf("inspect %s: %w", table, err)
		}
		have[name] = true
	}
	return have, rows.Err()
}

// epoch prefers the millisecond column and falls back to seconds.
func epoch(sec int64, ms sql.NullInt64) time.Time {
	if ms.Valid && ms.Int64 > 0 {
		return time.UnixMilli(ms.Int64).UTC()
	}
	return time.Unix(sec, 0).UTC()
}

var stateDBPattern = regexp.MustCompile(`^state_(\d+)\.sqlite$`)

// newestStateDBNumeric returns the state_N.sqlite with the highest numeric
// N (so state_10 sorts after state_9), or "" when none exists.
func newestStateDBNumeric(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("codex: read %s: %w", dir, err)
	}
	best, bestN := "", -1
	for _, e := range entries {
		m := stateDBPattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		if n, _ := strconv.Atoi(m[1]); n > bestN { //nolint:errcheck // \d+ always parses
			best, bestN = filepath.Join(dir, e.Name()), n
		}
	}
	return best, nil
}

// sessionRollout is the subset of a rollout line the reader uses.
type sessionRollout struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"payload"`
}

// humanText returns the text of a human-authored message, or "". Codex
// records injected context (environment, aborted-turn notices) as user
// messages wrapped in XML-like tags; those are not human activity.
func (rec sessionRollout) humanText() string {
	if rec.Type != "response_item" || rec.Payload.Type != "message" || rec.Payload.Role != "user" {
		return ""
	}
	var parts []string
	for _, c := range rec.Payload.Content {
		if c.Type != "input_text" {
			continue
		}
		text := strings.TrimSpace(c.Text)
		if text == "" || strings.HasPrefix(text, "<") {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

// readRolloutTail returns the last human-prompt time and the most recent
// human prompts from the end of a rollout file, growing the window until a
// prompt is found.
func readRolloutTail(path string) (time.Time, []sessions.Prompt, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, nil, err
	}
	defer f.Close() //nolint:errcheck // read-only file
	info, err := f.Stat()
	if err != nil {
		return time.Time{}, nil, err
	}
	size := info.Size()

	var (
		last    time.Time
		prompts []sessions.Prompt
	)
	for window := int64(rolloutTailBytes); ; window *= 4 {
		window = min(window, maxRolloutTailBytes)
		start := max(size-window, 0)
		last, prompts = time.Time{}, nil
		err := scanRollout(io.NewSectionReader(f, start, size-start), start > 0, func(rec sessionRollout) {
			if text := rec.humanText(); text != "" {
				last = rec.Timestamp
				prompts = append(prompts, sessions.Prompt{At: rec.Timestamp, Text: sessions.Truncate(text, promptWidth)})
			}
		})
		if err != nil {
			return time.Time{}, nil, err
		}
		if !last.IsZero() || start == 0 || window >= maxRolloutTailBytes {
			break
		}
	}
	if len(prompts) > recentPrompts {
		prompts = prompts[len(prompts)-recentPrompts:]
	}
	return last, prompts, nil
}

func scanRollout(r io.Reader, skipPartial bool, fn func(sessionRollout)) error {
	br := bufio.NewReaderSize(r, 1<<20)
	first := true
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 && !(first && skipPartial) {
			var rec sessionRollout
			if json.Unmarshal(bytes.TrimSpace(line), &rec) == nil {
				fn(rec)
			}
		}
		first = false
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// uuidPattern matches a Codex thread ID on a command line.
var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// runningSessions maps thread ID to PID for Codex processes that name their
// thread on the command line (codex resume <id>). Codex records no
// per-process liveness file, so a session started fresh is reported as
// unknown rather than guessed. Failure to list processes yields no entries.
func (r *SessionReader) runningSessions(ctx context.Context) map[string]int {
	running := map[string]int{}
	procs, err := r.processes(ctx)
	if err != nil {
		return running
	}
	for _, p := range procs {
		if !isCodexCommand(p.Command) {
			continue
		}
		if id := uuidPattern.FindString(p.Command); id != "" {
			running[id] = p.PID
		}
	}
	return running
}

// isCodexCommand reports whether a command line runs the codex CLI.
func isCodexCommand(cmd string) bool {
	exe, _, _ := strings.Cut(strings.TrimSpace(cmd), " ")
	return filepath.Base(exe) == "codex"
}

func listProcesses(ctx context.Context) ([]procInfo, error) {
	out, err := exec.CommandContext(ctx, "ps", "-Ao", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	var procs []procInfo
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		pidStr, cmd, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		procs = append(procs, procInfo{PID: pid, Command: strings.TrimSpace(cmd)})
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].PID < procs[j].PID })
	return procs, nil
}
