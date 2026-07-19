// Package omnidevx collects developer-experience events from Codex CLI's
// local stores for the OmniDevX domain (github.com/plexusone/omnidevx-core).
//
// Codex persists session metadata in a SQLite database (~/.codex/state_N.sqlite,
// threads table) and per-session event streams in rollout JSONL files
// (~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl). Both are internal Codex
// formats, not stable public contracts: parsing is defensive, unrecognized
// records are skipped, and all events carry history-mode provenance.
//
// Only metadata is extracted — models, token usage, tool names, durations,
// timestamps, workspace and repository identifiers. Prompt text, agent
// output, and command arguments are never read into events.
//
// This collector lives here rather than in omnidevx-core/providers because
// it requires a SQLite driver (modernc.org/sqlite), which is too heavy a
// dependency for the core module.
package omnidevx
