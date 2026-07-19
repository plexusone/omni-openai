package omnidevx

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	core "github.com/plexusone/omnidevx-core"
)

// DefaultConfidence is the provenance confidence for events reconstructed
// from Codex local stores.
const DefaultConfidence = 0.9

var source = core.Source{
	Provider: "openai",
	Product:  "codex-cli",
}

// Config holds configuration for the Codex CLI collector.
type Config struct {
	// Dir is the Codex home directory. Defaults to ~/.codex.
	Dir string
}

// Collector reads Codex CLI local history (SQLite thread index plus rollout
// JSONL event streams).
type Collector struct {
	dir string
}

var _ core.Collector = (*Collector)(nil)

// New returns a Collector for the given config.
func New(cfg Config) (*Collector, error) {
	dir := cfg.Dir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("codex: resolve home directory: %w", err)
		}
		dir = filepath.Join(home, ".codex")
	}
	return &Collector{dir: dir}, nil
}

// Source implements omnidevx.Collector.
func (c *Collector) Source() core.Source { return source }

// Collect implements omnidevx.Collector. Session metadata comes from the
// SQLite threads table when present; per-session events come from rollout
// JSONL files. Threads seen in SQLite are not double-counted when their
// rollout also carries session metadata.
func (c *Collector) Collect(ctx context.Context, req core.CollectRequest) (*core.CollectionResult, error) {
	result := &core.CollectionResult{
		Source:      source,
		Subject:     req.Subject,
		Period:      req.Period,
		Events:      []core.Event{},
		CollectedAt: time.Now().UTC(),
	}

	threadIDs, err := c.collectThreads(ctx, req, result)
	if err != nil {
		return nil, err
	}

	if err := c.collectRollouts(ctx, req, result, threadIDs); err != nil {
		return nil, err
	}

	for i := range result.Events {
		result.Events[i].Subject = req.Subject
	}
	return result, nil
}

// collectRollouts parses every rollout file under sessions/.
func (c *Collector) collectRollouts(ctx context.Context, req core.CollectRequest, result *core.CollectionResult, seenThreads map[string]bool) error {
	sessionsDir := filepath.Join(c.dir, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
				Severity: core.SeverityWarning,
				Message:  fmt.Sprintf("walk sessions: %v", err),
				Path:     path,
			})
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		events, diags := parseRolloutFile(path, req, seenThreads)
		result.Events = append(result.Events, events...)
		result.Diagnostics = append(result.Diagnostics, diags...)
		return nil
	})
}

// normalizeRepoURL converts a git remote URL to a canonical repository
// identifier, e.g. "https://github.com/x/y.git" or "git@github.com:x/y.git"
// become "github.com/x/y".
func normalizeRepoURL(remote string) string {
	repo := strings.TrimSpace(remote)
	if repo == "" {
		return ""
	}
	repo = strings.TrimSuffix(repo, ".git")
	for _, prefix := range []string{"https://", "http://", "ssh://", "git://"} {
		repo = strings.TrimPrefix(repo, prefix)
	}
	if at := strings.Index(repo, "@"); at >= 0 {
		repo = repo[at+1:]
		repo = strings.Replace(repo, ":", "/", 1)
	}
	return repo
}

func provenance() core.Provenance {
	return core.Provenance{
		CollectionMode: core.ModeHistory,
		Confidence:     DefaultConfidence,
	}
}
