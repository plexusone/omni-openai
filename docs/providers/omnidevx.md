# OmniDevX: Codex CLI Collector

The `omnidevx` package collects developer-experience telemetry from
**Codex CLI's local stores** for the
[OmniDevX](https://github.com/plexusone/omnidevx-core) domain — the
PlexusOne developer-telemetry ecosystem behind SPACE/AI SPACE analytics.

It lives in this repository (rather than `omnidevx-core/providers`)
because reading Codex's state database requires a SQLite driver
(`modernc.org/sqlite`) — too heavy a dependency for the core module.
This mirrors the provider split used across the Omni ecosystem: lightweight,
stdlib-only collectors can live in `omnidevx-core`, while vendor-specific
collectors with heavier dependencies stay in the vendor repository.

## What it reads

Codex CLI persists history in two internal formats, both collected:

- **SQLite threads index** (`~/.codex/state_N.sqlite`) — session metadata:
  model, reasoning effort, git branch and origin, CLI version, session
  token totals. Content-bearing columns (`title`, `first_user_message`,
  `preview`) are **never selected**.
- **Rollout JSONL** (`~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`) —
  per-session event streams: prompts, tool calls, patch applies, task
  durations, per-turn token usage.

Both are internal Codex formats: parsing is defensive and all events carry
history-mode provenance.

## Session reader

`SessionReader` implements `sessions.Reader` from
[`omnidevx-core`](https://github.com/plexusone/omnidevx-core) so Codex
sessions can be listed and resumed alongside Claude Code sessions. Unlike the
collector, it reads titles and prompt text so that a person can recognize a
session. That content stays in memory: it is never turned into an event or
written to the event store, and `NoContent` suppresses it.

```go
reader, err := codex.NewSessionReader(codex.Config{})
if err != nil {
    return err
}
cat := sessions.NewCatalog(reader)
list, diagnostics, err := cat.List(ctx, sessions.ListOptions{})
```

What it reports for each session:

- **Identity and place.** The thread ID, working directory, branch, and
  origin URL from the newest `~/.codex/state_N.sqlite`.
- **Title.** The name or title Codex recorded, else the first prompt, else
  the directory name.
- **Recent prompts.** The last few prompts a person typed, read from the end
  of the rollout file with a bounded read that grows only while no typed
  prompt has been found.
- **Running state.** Codex keeps no per-process liveness file, so a session
  is `running` only when a `codex` process names its thread ID on the command
  line, as `codex resume <id>` does. A session started fresh and still open is
  reported as `unknown`, not guessed. Where `ps` is unavailable, including
  Windows, every session is `unknown`.
- **Resume.** `codex resume <id>` from the session's working directory.

The reader tolerates schema drift in the same way as the collector: columns
that a Codex version lacks degrade to empty, and only `id`, `cwd`, and the
timestamps are required.

## Format stability

Codex's local files are not a stable public API. The collector treats the
SQLite index as an optimization for richer session metadata, not as the only
source of truth:

- If `~/.codex/state_N.sqlite` is missing, collection continues from rollout
  JSONL files.
- If the SQLite schema changes or expected columns disappear, collection emits
  a diagnostic and falls back to rollout-only collection.
- If rollout record types change, unknown records are skipped; recognized
  metadata-only records still become OmniDevX events.

When Codex changes local paths or schemas, update this provider in
`omni-openai/omnidevx` rather than moving SQLite support into
`omnidevx-core`. The core module should remain dependency-light and continue
to receive normalized events through the `omnidevx-core.Collector` interface.

## Usage

```go
import (
    codex "github.com/plexusone/omni-openai/omnidevx"
    core "github.com/plexusone/omnidevx-core"
)

collector, err := codex.New(codex.Config{
    Dir: "", // default ~/.codex
})
result, err := collector.Collect(ctx, core.CollectRequest{
    Period:  core.Period{Start: weekStart, End: weekEnd},
    Subject: core.SubjectRef{PersonID: "person:jane"},
})
```

## Events emitted

| Event | Source |
|-------|--------|
| `ai.session.started` / `ai.session.ended` | SQLite threads (or rollout `session_meta` for unindexed sessions) |
| `ai.prompt.submitted` | `user_message` records (no text captured) |
| `ai.message.completed` | `agent_message` records |
| `ai.task.started` / `ai.task.completed` | Turn records, with `duration_ms` and `time_to_first_token_ms` |
| `ai.tool.completed` | `function_call` / `custom_tool_call` outputs, with tool name attribution |
| `ai.patch.applied` | `patch_apply_end`, with success |
| `ai.usage.recorded` | `token_count` per-turn deltas (input, cached, output, reasoning) |

Only metadata is extracted — prompt text, agent output, and command
arguments never enter events. For live token capture via OpenTelemetry,
see [omnidevx-otel](https://github.com/plexusone/omnidevx-otel).
