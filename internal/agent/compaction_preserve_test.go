package agent

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/askspecter/holt/internal/holtruntime"
)

// stateConversation is a long enough conversation that Compact elides a middle
// containing an update_plan call and a loaded skill (call + result).
func stateConversation() []holtruntime.Message {
	return []holtruntime.Message{
		{Role: holtruntime.MessageRoleSystem, Content: "system"},
		{Role: holtruntime.MessageRoleUser, Content: "build the thing"},
		{Role: holtruntime.MessageRoleAssistant, Content: "planning", ToolCalls: []holtruntime.ToolCall{
			{ID: "p1", Name: "update_plan", Arguments: `{"plan":[{"content":"write code","status":"in_progress"},{"content":"add tests","status":"pending"}]}`},
		}},
		{Role: holtruntime.MessageRoleTool, Content: "plan updated", ToolCallID: "p1"},
		{Role: holtruntime.MessageRoleAssistant, Content: "loading skill", ToolCalls: []holtruntime.ToolCall{
			{ID: "s1", Name: "skill", Arguments: `{"name":"deploy"}`},
		}},
		{Role: holtruntime.MessageRoleTool, Content: "Deploy skill: run `make deploy` then tag the release.", ToolCallID: "s1"},
		{Role: holtruntime.MessageRoleAssistant, Content: "done step 1"},
		{Role: holtruntime.MessageRoleUser, Content: "continue"},
		{Role: holtruntime.MessageRoleAssistant, Content: "continuing"},
	}
}

func compactStateConversation(t *testing.T, messages []holtruntime.Message) string {
	t.Helper()
	compacted, err := Compact(messages, CompactionOptions{
		PreserveLast: 2,
		Summarize:    func([]holtruntime.Message) (string, error) { return "SUMMARY", nil },
	})
	if err != nil {
		t.Fatalf("Compact returned error: %v", err)
	}
	// [system, summaryUserMsg, ...suffix] — the summary is the message after system.
	if len(compacted) < 2 || compacted[1].Role != holtruntime.MessageRoleUser {
		t.Fatalf("unexpected compacted shape: %#v", compacted)
	}
	if !strings.Contains(compacted[1].Content, summaryLabel) {
		t.Fatalf("summary message missing label: %q", compacted[1].Content)
	}
	return compacted[1].Content
}

func TestCompactPreservesActivePlan(t *testing.T) {
	summary := compactStateConversation(t, stateConversation())
	if !strings.Contains(summary, preservedStateLabel) {
		t.Fatalf("expected preserved-state block, got %q", summary)
	}
	for _, want := range []string{"- [in_progress] write code", "- [pending] add tests"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("plan item %q not preserved in %q", want, summary)
		}
	}
}

func TestCompactPreservesLoadedSkills(t *testing.T) {
	summary := compactStateConversation(t, stateConversation())
	if !strings.Contains(summary, preservedStateLabel) {
		t.Fatalf("expected preserved-state block, got %q", summary)
	}
	if !strings.Contains(summary, `"name":"deploy"`) || !strings.Contains(summary, "make deploy") {
		t.Fatalf("skill name/body not preserved in %q", summary)
	}
}

func TestCompactPreservesLoadedToolSearchSchemas(t *testing.T) {
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleSystem, Content: "system"},
		{Role: holtruntime.MessageRoleUser, Content: "load weather tool"},
		{Role: holtruntime.MessageRoleAssistant, Content: "loading", ToolCalls: []holtruntime.ToolCall{
			{ID: "t1", Name: "tool_search", Arguments: `{"query":"select:weather_lookup"}`},
		}},
		{Role: holtruntime.MessageRoleTool, ToolCallID: "t1", Content: "Loaded 1 tool. Full schemas follow; call them on the next turn.\n\n## weather_lookup\nLook up weather.\ninput schema:\n{\n  \"type\": \"object\"\n}"},
		{Role: holtruntime.MessageRoleAssistant, Content: "ready"},
		{Role: holtruntime.MessageRoleUser, Content: "continue"},
		{Role: holtruntime.MessageRoleAssistant, Content: "continuing"},
	}
	summary := compactStateConversation(t, messages)
	if !strings.Contains(summary, `"name":"weather_lookup"`) || !strings.Contains(summary, "input schema") {
		t.Fatalf("loaded tool schema not preserved in %q", summary)
	}
}

func TestCompactPreservesProjectInstructions(t *testing.T) {
	projectInstructions := "# AGENTS.md instructions for D:\\repo\n\n<INSTRUCTIONS>\nUse `go test ./internal/agent` for agent changes.\nDo not touch TUI code.\n</INSTRUCTIONS>\n\n<environment_context>\nignored runtime context\n</environment_context>"
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleSystem, Content: "system"},
		{Role: holtruntime.MessageRoleUser, Content: projectInstructions},
		{Role: holtruntime.MessageRoleAssistant, Content: "ack"},
		{Role: holtruntime.MessageRoleUser, Content: "work on compaction"},
		{Role: holtruntime.MessageRoleAssistant, Content: "working"},
		{Role: holtruntime.MessageRoleUser, Content: "continue"},
		{Role: holtruntime.MessageRoleAssistant, Content: "continuing"},
	}
	summary := compactStateConversation(t, messages)
	state := parsePreservedStateBlock(summary)
	if len(state.ProjectInstructions) != 1 {
		t.Fatalf("expected one preserved project instruction block, got %#v", state.ProjectInstructions)
	}
	body := state.ProjectInstructions[0].Body
	if state.ProjectInstructions[0].Source != "AGENTS.md instructions for D:\\repo" ||
		!strings.Contains(body, "# AGENTS.md instructions for D:\\repo") ||
		!strings.Contains(body, "Do not touch TUI code.") ||
		strings.Contains(body, "ignored runtime context") {
		t.Fatalf("project instructions not preserved cleanly in %#v", state.ProjectInstructions[0])
	}
}

func TestProjectInstructionBlockAcceptsProjectGuidelineFilename(t *testing.T) {
	source, body := projectInstructionBlock("# HOLT.md instructions for /repo\n\n<INSTRUCTIONS>\nPrefer Go commands.\n</INSTRUCTIONS>")
	if source != "HOLT.md instructions for /repo" || !strings.Contains(body, "Prefer Go commands.") {
		t.Fatalf("expected HOLT.md instruction block to parse, got source=%q body=%q", source, body)
	}
}

func TestCompactWithoutStateHasNoPreserveSections(t *testing.T) {
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleSystem, Content: "system"},
		{Role: holtruntime.MessageRoleUser, Content: "hello"},
		{Role: holtruntime.MessageRoleAssistant, Content: "hi there"},
		{Role: holtruntime.MessageRoleUser, Content: "tell me more"},
		{Role: holtruntime.MessageRoleAssistant, Content: "sure"},
		{Role: holtruntime.MessageRoleUser, Content: "and more"},
		{Role: holtruntime.MessageRoleAssistant, Content: "ok"},
	}
	summary := compactStateConversation(t, messages)
	if strings.Contains(summary, preservedStateLabel) {
		t.Fatalf("did not expect a preserved-state block without plan/skill: %q", summary)
	}
}

func TestCompactCarriesPreservedStateAcrossRepeatedCompaction(t *testing.T) {
	// First compaction: real update_plan + skill load in the elided middle.
	first, err := Compact(stateConversation(), CompactionOptions{
		PreserveLast: 2,
		Summarize:    func([]holtruntime.Message) (string, error) { return "FIRST SUMMARY", nil },
	})
	if err != nil {
		t.Fatalf("first Compact: %v", err)
	}

	// Grow the history so the first summary (which holds the preserved sections,
	// but no real tool calls) falls into the SECOND compaction's middle.
	second := append([]holtruntime.Message{}, first...)
	second = append(second,
		holtruntime.Message{Role: holtruntime.MessageRoleUser, Content: "what next"},
		holtruntime.Message{Role: holtruntime.MessageRoleAssistant, Content: "continuing"},
		holtruntime.Message{Role: holtruntime.MessageRoleUser, Content: "keep going"},
		holtruntime.Message{Role: holtruntime.MessageRoleAssistant, Content: "done"},
	)

	// The second summarizer deliberately DROPS the preserved sections.
	out, err := Compact(second, CompactionOptions{
		PreserveLast: 2,
		Summarize:    func([]holtruntime.Message) (string, error) { return "SECOND SUMMARY with no preserved sections", nil },
	})
	if err != nil {
		t.Fatalf("second Compact: %v", err)
	}
	if len(out) < 2 || out[1].Role != holtruntime.MessageRoleUser {
		t.Fatalf("unexpected compacted shape: %#v", out)
	}
	newSummary := out[1].Content
	// Plan and skill must survive even though the summarizer didn't copy them.
	if !strings.Contains(newSummary, preservedStateLabel) || !strings.Contains(newSummary, "write code") {
		t.Fatalf("active plan lost on the second compaction: %q", newSummary)
	}
	if !strings.Contains(newSummary, `"name":"deploy"`) || !strings.Contains(newSummary, "make deploy") {
		t.Fatalf("loaded skill lost on the second compaction: %q", newSummary)
	}
}

func TestCompactCarriesLoadedToolsAndProjectInstructionsAcrossRepeatedCompaction(t *testing.T) {
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleSystem, Content: "system"},
		{Role: holtruntime.MessageRoleUser, Content: "# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>\nStay in internal/agent.\n</INSTRUCTIONS>"},
		{Role: holtruntime.MessageRoleAssistant, Content: "loading", ToolCalls: []holtruntime.ToolCall{
			{ID: "t1", Name: "tool_search", Arguments: `{"query":"select:weather_lookup"}`},
		}},
		{Role: holtruntime.MessageRoleTool, ToolCallID: "t1", Content: "Loaded 1 tool. Full schemas follow; call them on the next turn.\n\n## weather_lookup\nLook up weather.\ninput schema:\n{\n  \"type\": \"object\"\n}"},
		{Role: holtruntime.MessageRoleAssistant, Content: "ready"},
		{Role: holtruntime.MessageRoleUser, Content: "continue"},
		{Role: holtruntime.MessageRoleAssistant, Content: "continuing"},
	}

	first, err := Compact(messages, CompactionOptions{
		PreserveLast: 2,
		Summarize:    func([]holtruntime.Message) (string, error) { return "FIRST SUMMARY", nil },
	})
	if err != nil {
		t.Fatalf("first Compact: %v", err)
	}
	second := append(append([]holtruntime.Message{}, first...),
		holtruntime.Message{Role: holtruntime.MessageRoleUser, Content: "more"},
		holtruntime.Message{Role: holtruntime.MessageRoleAssistant, Content: "ok"},
		holtruntime.Message{Role: holtruntime.MessageRoleUser, Content: "again"},
		holtruntime.Message{Role: holtruntime.MessageRoleAssistant, Content: "fine"},
	)

	out, err := Compact(second, CompactionOptions{
		PreserveLast: 2,
		Summarize:    func([]holtruntime.Message) (string, error) { return "SECOND SUMMARY", nil },
	})
	if err != nil {
		t.Fatalf("second Compact: %v", err)
	}
	state := parsePreservedStateBlock(out[1].Content)
	if len(state.Tools) != 1 || state.Tools[0].Name != "weather_lookup" || !strings.Contains(state.Tools[0].Body, "input schema") {
		t.Fatalf("loaded tool state was not carried forward: %#v", state.Tools)
	}
	if len(state.ProjectInstructions) != 1 ||
		state.ProjectInstructions[0].Source != "AGENTS.md instructions for /repo" ||
		!strings.Contains(state.ProjectInstructions[0].Body, "Stay in internal/agent.") {
		t.Fatalf("project instructions were not carried forward: %#v", state.ProjectInstructions)
	}
}

// TestCompactPreservesSkillBodyWithMarkdownHeadings is CodeRabbit's regression:
// a verbatim skill body containing ## / ### headings (and a code fence) must
// round-trip across TWO compactions without truncation or bogus extra skills —
// which the old markdown-delimited format could not guarantee.
func TestCompactPreservesSkillBodyWithMarkdownHeadings(t *testing.T) {
	body := "## Usage\nrun it\n### Examples\n```\nzero do\n```\n## Notes\ndone"
	conv := []holtruntime.Message{
		{Role: holtruntime.MessageRoleSystem, Content: "system"},
		{Role: holtruntime.MessageRoleUser, Content: "load a skill"},
		{Role: holtruntime.MessageRoleAssistant, Content: "loading", ToolCalls: []holtruntime.ToolCall{
			{ID: "s1", Name: "skill", Arguments: `{"name":"guide"}`},
		}},
		{Role: holtruntime.MessageRoleTool, Content: body, ToolCallID: "s1"},
		{Role: holtruntime.MessageRoleAssistant, Content: "done step 1"},
		{Role: holtruntime.MessageRoleUser, Content: "continue"},
		{Role: holtruntime.MessageRoleAssistant, Content: "continuing"},
	}

	mustContainBody := func(label string, messages []holtruntime.Message) []holtruntime.Message {
		out, err := Compact(messages, CompactionOptions{
			PreserveLast: 2,
			Summarize:    func([]holtruntime.Message) (string, error) { return "SUMMARY", nil },
		})
		if err != nil {
			t.Fatalf("%s Compact: %v", label, err)
		}
		if len(out) < 2 || out[1].Role != holtruntime.MessageRoleUser {
			t.Fatalf("%s: unexpected compacted shape: %#v", label, out)
		}
		_, skills := parsePreservedState(out[1].Content)
		if len(skills) != 1 || skills[0].name != "guide" || skills[0].body != body {
			t.Fatalf("%s: skill body not round-tripped intact: %#v", label, skills)
		}
		return out
	}

	first := mustContainBody("first", conv)
	// Second compaction with NO fresh tool calls and a summarizer that drops it.
	second := append(append([]holtruntime.Message{}, first...),
		holtruntime.Message{Role: holtruntime.MessageRoleUser, Content: "more"},
		holtruntime.Message{Role: holtruntime.MessageRoleAssistant, Content: "ok"},
		holtruntime.Message{Role: holtruntime.MessageRoleUser, Content: "again"},
		holtruntime.Message{Role: holtruntime.MessageRoleAssistant, Content: "fine"},
	)
	mustContainBody("second", second)
}

func TestExtractLatestPlanReturnsMostRecent(t *testing.T) {
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleAssistant, ToolCalls: []holtruntime.ToolCall{
			{ID: "a", Name: "update_plan", Arguments: `{"plan":[{"content":"old step","status":"completed"}]}`},
		}},
		{Role: holtruntime.MessageRoleAssistant, ToolCalls: []holtruntime.ToolCall{
			{ID: "b", Name: "update_plan", Arguments: `{"plan":[{"content":"new step","status":"in_progress"}]}`},
		}},
	}
	got := extractLatestPlan(messages)
	if !strings.Contains(got, "new step") || strings.Contains(got, "old step") {
		t.Fatalf("extractLatestPlan should return the most recent plan, got %q", got)
	}
}

func TestFormatPlanArgumentsAcceptsStepAlias(t *testing.T) {
	got := formatPlanArguments(`{"plan":[{"step":"write failing test","status":"in_progress"},{"content":"keep existing shape","status":"pending"}]}`)
	for _, want := range []string{"- [in_progress] write failing test", "- [pending] keep existing shape"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in formatted plan, got %q", want, got)
		}
	}
}

func TestFormatPlanArgumentsPreservesNotes(t *testing.T) {
	got := formatPlanArguments(`{"plan":[{"content":"finish preservation","status":"in_progress","notes":"keep TUI untouched"}]}`)
	if !strings.Contains(got, "- [in_progress] finish preservation") || !strings.Contains(got, "Notes: keep TUI untouched") {
		t.Fatalf("expected plan content and notes to be preserved, got %q", got)
	}
}

func TestCapBodyShortBodyUnchanged(t *testing.T) {
	body := "short skill body"
	if got := capBody(body); got != body {
		t.Fatalf("capBody changed a short body: %q", got)
	}
	if strings.Contains(capBody(body), "truncated") {
		t.Fatalf("note added without truncation")
	}
}

func TestCapBodyRespectsByteBudgetForMultibyte(t *testing.T) {
	// Each '世' is 3 bytes; build a body well over the byte budget.
	body := strings.Repeat("世", maxPreservedSkillBytes)
	got := capBody(body)
	if len(got) > maxPreservedSkillBytes {
		t.Fatalf("capBody returned %d bytes, want <= %d (byte budget)", len(got), maxPreservedSkillBytes)
	}
	if !strings.Contains(got, "truncated") {
		t.Fatalf("expected truncation note on an over-budget body")
	}
	if !utf8.ValidString(got) {
		t.Fatalf("capBody split a multibyte rune (invalid UTF-8)")
	}
}

func TestCapBodyNoteOnlyWhenTruncated(t *testing.T) {
	// A body whose RUNE count is under the cap but BYTE length is over it must
	// still be byte-capped (the old rune-based check mishandled this case).
	body := strings.Repeat("世", (maxPreservedSkillBytes/3)+100)
	if len(body) <= maxPreservedSkillBytes {
		t.Fatalf("test setup: body should exceed the byte budget, got %d", len(body))
	}
	got := capBody(body)
	if len(got) > maxPreservedSkillBytes {
		t.Fatalf("capBody returned %d bytes, want <= %d", len(got), maxPreservedSkillBytes)
	}
	if !strings.Contains(got, "truncated") || !utf8.ValidString(got) {
		t.Fatalf("expected a valid, truncated body, got %q", got)
	}
}

func TestLoadedSkillsSkipsCallsWithoutResult(t *testing.T) {
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleAssistant, ToolCalls: []holtruntime.ToolCall{
			{ID: "s1", Name: "skill", Arguments: `{"name":"ghost"}`}, // no matching tool result
		}},
	}
	if got := loadedSkills(messages); len(got) != 0 {
		t.Fatalf("expected no skills without a result body, got %#v", got)
	}
}

func TestLoadedSkillsAcceptsSkillArgumentAlias(t *testing.T) {
	messages := []holtruntime.Message{
		{Role: holtruntime.MessageRoleAssistant, ToolCalls: []holtruntime.ToolCall{
			{ID: "s1", Name: "skill", Arguments: `{"skill":"deploy"}`},
		}},
		{Role: holtruntime.MessageRoleTool, ToolCallID: "s1", Content: "deploy instructions"},
	}
	got := loadedSkills(messages)
	if len(got) != 1 || got[0].name != "deploy" || got[0].body != "deploy instructions" {
		t.Fatalf("loadedSkills should honor skill alias, got %#v", got)
	}
}
