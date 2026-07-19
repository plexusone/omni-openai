package omnidevx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	core "github.com/plexusone/omnidevx-core"
)

// rolloutRecord is one line of a Codex rollout JSONL file.
type rolloutRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// rolloutPayload is the union of payload fields OmniDevX reads across
// session_meta, event_msg, and response_item records. Content fields
// (arguments, output, messages) are never decoded.
type rolloutPayload struct {
	Type string `json:"type"`

	// session_meta
	ID         string      `json:"id"`
	CWD        string      `json:"cwd"`
	CLIVersion string      `json:"cli_version"`
	Git        *rolloutGit `json:"git"`

	// response_item tool calls
	Name   string `json:"name"`
	CallID string `json:"call_id"`

	// event_msg task_complete
	DurationMS       *int64 `json:"duration_ms"`
	TimeToFirstToken *int64 `json:"time_to_first_token_ms"`

	// event_msg patch_apply_end
	Success *bool `json:"success"`

	// event_msg token_count
	LastTokenUsage *tokenUsage `json:"last_token_usage"`
	Info           *tokenInfo  `json:"info"`
}

type rolloutGit struct {
	Branch        string `json:"branch"`
	RepositoryURL string `json:"repository_url"`
}

type tokenInfo struct {
	LastTokenUsage *tokenUsage `json:"last_token_usage"`
}

type tokenUsage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

// parseRolloutFile converts one rollout file into canonical events. Session
// start events are suppressed for threads already indexed in SQLite.
func parseRolloutFile(path string, req core.CollectRequest, seenThreads map[string]bool) ([]core.Event, []core.Diagnostic) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []core.Diagnostic{{
			Severity: core.SeverityError,
			Message:  fmt.Sprintf("open rollout file: %v", err),
			Path:     path,
		}}
	}
	defer f.Close() //nolint:errcheck // read-only file

	var (
		events    []core.Event
		diags     []core.Diagnostic
		sessionID string
		evtCtx    core.EventContext
		toolNames = map[string]string{}
		scanner   = bufio.NewReaderSize(f, 1<<20)
		lineNo    = 0
	)

	for {
		line, err := readLine(scanner)
		if err == io.EOF {
			break
		}
		if err != nil {
			diags = append(diags, core.Diagnostic{
				Severity: core.SeverityError,
				Message:  fmt.Sprintf("read line %d: %v", lineNo+1, err),
				Path:     path,
			})
			break
		}
		lineNo++
		if len(line) == 0 {
			continue
		}

		var rec rolloutRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			diags = append(diags, core.Diagnostic{
				Severity: core.SeverityWarning,
				Message:  fmt.Sprintf("skipping unparseable line %d: %v", lineNo, err),
				Path:     path,
			})
			continue
		}

		var payload rolloutPayload
		if len(rec.Payload) > 0 {
			if err := json.Unmarshal(rec.Payload, &payload); err != nil {
				diags = append(diags, core.Diagnostic{
					Severity: core.SeverityWarning,
					Message:  fmt.Sprintf("skipping unparseable payload at line %d: %v", lineNo, err),
					Path:     path,
				})
				continue
			}
		}

		ts, tsErr := time.Parse(time.RFC3339, rec.Timestamp)
		if tsErr != nil {
			ts = time.Time{}
		}

		if rec.Type == "session_meta" {
			sessionID = payload.ID
			evtCtx = core.EventContext{
				SessionID: sessionID,
				Workspace: payload.CWD,
			}
			if payload.Git != nil {
				evtCtx.GitBranch = payload.Git.Branch
				evtCtx.Repository = normalizeRepoURL(payload.Git.RepositoryURL)
			}
			// SQLite already emitted session events for indexed threads.
			if !seenThreads[sessionID] && req.Period.Contains(ts) {
				attrs := map[string]any{}
				if payload.CLIVersion != "" {
					attrs[core.AttrClientVersion] = payload.CLIVersion
				}
				if len(attrs) == 0 {
					attrs = nil
				}
				events = append(events, core.Event{
					ID:         "codex-cli:" + sessionID + ":start",
					Type:       core.EventSessionStarted,
					Timestamp:  ts,
					Source:     source,
					Context:    evtCtx,
					Attributes: attrs,
					Provenance: provenance(),
				})
			}
			continue
		}

		eventType, attrs := classifyRollout(rec.Type, &payload, toolNames)
		if eventType == "" || !req.Period.Contains(ts) {
			continue
		}
		events = append(events, core.Event{
			ID:         fmt.Sprintf("codex-cli:%s:%d", sessionID, lineNo),
			Type:       eventType,
			Timestamp:  ts,
			Source:     source,
			Context:    evtCtx,
			Attributes: attrs,
			Provenance: provenance(),
		})
	}
	return events, diags
}

// classifyRollout maps a rollout record to a canonical event type and
// attributes. It returns an empty type for records OmniDevX does not track.
// Tool names are registered on call records and resolved on output records.
func classifyRollout(recType string, p *rolloutPayload, toolNames map[string]string) (core.EventType, map[string]any) {
	switch recType {
	case "event_msg":
		switch p.Type {
		case "user_message":
			return core.EventPromptSubmitted, nil
		case "agent_message":
			return core.EventMessageCompleted, nil
		case "task_started":
			return core.EventTaskStarted, nil
		case "task_complete":
			attrs := map[string]any{}
			if p.DurationMS != nil {
				attrs[core.AttrDurationMS] = *p.DurationMS
			}
			if p.TimeToFirstToken != nil {
				attrs[core.AttrTimeToFirstTokenMS] = *p.TimeToFirstToken
			}
			if len(attrs) == 0 {
				attrs = nil
			}
			return core.EventTaskCompleted, attrs
		case "patch_apply_end":
			attrs := map[string]any{}
			if p.Success != nil {
				attrs[core.AttrSuccess] = *p.Success
			}
			if len(attrs) == 0 {
				attrs = nil
			}
			return core.EventPatchApplied, attrs
		case "token_count":
			u := p.LastTokenUsage
			if u == nil && p.Info != nil {
				u = p.Info.LastTokenUsage
			}
			if u == nil {
				return "", nil
			}
			return core.EventUsageRecorded, map[string]any{
				core.AttrInputTokens:     u.InputTokens,
				core.AttrCacheReadTokens: u.CachedInputTokens,
				core.AttrOutputTokens:    u.OutputTokens,
				core.AttrReasoningTokens: u.ReasoningOutputTokens,
				core.AttrTotalTokens:     u.TotalTokens,
			}
		}
	case "response_item":
		switch p.Type {
		case "function_call", "custom_tool_call":
			if p.CallID != "" {
				toolNames[p.CallID] = p.Name
			}
			return "", nil
		case "function_call_output", "custom_tool_call_output":
			attrs := map[string]any{}
			if name := toolNames[p.CallID]; name != "" {
				attrs[core.AttrTool] = name
			}
			if len(attrs) == 0 {
				attrs = nil
			}
			return core.EventToolCompleted, attrs
		}
	}
	return "", nil
}

// readLine reads one full line without a fixed length cap; rollout lines can
// exceed bufio.Scanner's default limits.
func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err == nil {
		return line[:len(line)-1], nil
	}
	if err == io.EOF && len(line) > 0 {
		return line, nil
	}
	return nil, err
}
