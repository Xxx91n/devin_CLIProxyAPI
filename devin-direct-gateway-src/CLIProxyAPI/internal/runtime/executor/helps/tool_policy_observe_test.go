package helps

import (
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

const observeTriggerDescription = "Reads a file from the local filesystem. You can access any file directly by using this tool."

func observeTestConfig() *config.Config {
	return &config.Config{
		OAuthToolPolicyObserve: map[string][]config.ToolPolicyObserveRule{
			"devin": {
				{
					ID: "desc-unrestricted-fs-read",
					MatchRegexr: []string{
						`(?i)file ?system[\s\S]{0,200}access\s+any\s+file`,
						`(?i)access\s+any\s+file[\s\S]{0,200}file ?system`,
					},
				},
			},
		},
	}
}

func TestObserveToolPolicyHitOnConfirmedTrigger(t *testing.T) {
	payload := []byte(`{"model":"devin/swe-2","tools":[{"type":"function","function":{"name":"Read","description":"` + observeTriggerDescription + `","parameters":{"type":"object"}}}]}`)

	notes := ObserveToolPolicy(observeTestConfig(), "devin", payload)
	if len(notes) == 0 {
		t.Fatal("expected observe notes for the confirmed trigger description")
	}
	found := false
	for _, n := range notes {
		if strings.Contains(n, "rule=desc-unrestricted-fs-read") && strings.Contains(n, "tool=Read") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notes missing rule id / tool name: %v", notes)
	}
}

func TestObserveToolPolicyNoHitOnBenignDescription(t *testing.T) {
	payload := []byte(`{"tools":[{"type":"function","function":{"name":"Write","description":"Writes a file to the local filesystem.","parameters":{"type":"object"}}}]}`)

	if notes := ObserveToolPolicy(observeTestConfig(), "devin", payload); len(notes) != 0 {
		t.Fatalf("expected no notes for benign description, got %v", notes)
	}
}

func TestObserveToolPolicyMatchSubstringAndNameField(t *testing.T) {
	cfg := &config.Config{
		OAuthToolPolicyObserve: map[string][]config.ToolPolicyObserveRule{
			"devin": {{ID: "name-hit", Match: []string{"access any file"}}},
		},
	}
	// Trigger via tool name field rather than description.
	payload := []byte(`{"tools":[{"name":"access any file tool","description":"innocent"}]}`)
	notes := ObserveToolPolicy(cfg, "devin", payload)
	if len(notes) == 0 || !strings.Contains(notes[0], "field=name") {
		t.Fatalf("expected name-field note, got %v", notes)
	}
}

func TestObserveToolPolicyGuards(t *testing.T) {
	cfg := observeTestConfig()
	payload := []byte(`{"tools":[{"function":{"name":"Read","description":"` + observeTriggerDescription + `"}}]}`)

	if notes := ObserveToolPolicy(nil, "devin", payload); notes != nil {
		t.Fatal("nil cfg must produce nil notes")
	}
	if notes := ObserveToolPolicy(cfg, "codex", payload); notes != nil {
		t.Fatal("provider without rules must produce nil notes")
	}
	if notes := ObserveToolPolicy(cfg, "devin", nil); notes != nil {
		t.Fatal("nil payload must produce nil notes")
	}
	if notes := ObserveToolPolicy(cfg, "devin", []byte(`{"messages":[]}`)); notes != nil {
		t.Fatal("payload without tools must produce nil notes")
	}
	if notes := ObserveToolPolicy(cfg, "devin", []byte(`not json`)); notes != nil {
		t.Fatal("non-json payload must produce nil notes")
	}
}

func TestObserveToolPolicyMultiToolSingleHit(t *testing.T) {
	payload := []byte(`{"tools":[
		{"type":"function","function":{"name":"Bash","description":"run shell","parameters":{}}},
		{"type":"function","function":{"name":"Read","description":"` + observeTriggerDescription + `","parameters":{}}},
		{"type":"function","function":{"name":"Grep","description":"search","parameters":{}}}
	]}`)

	notes := ObserveToolPolicy(observeTestConfig(), "devin", payload)
	if len(notes) == 0 {
		t.Fatal("expected notes for the matching tool")
	}
	for _, n := range notes {
		if strings.Contains(n, "tool=Bash") || strings.Contains(n, "tool=Grep") {
			t.Fatalf("unexpected note for non-matching tool: %v", n)
		}
	}
}

func TestObserveToolPolicySkipsInvalidRegex(t *testing.T) {
	cfg := &config.Config{
		OAuthToolPolicyObserve: map[string][]config.ToolPolicyObserveRule{
			"devin": {{ID: "bad", MatchRegexr: []string{"([invalid"}}},
		},
	}
	payload := []byte(`{"tools":[{"function":{"name":"Read","description":"` + observeTriggerDescription + `"}}]}`)
	if notes := ObserveToolPolicy(cfg, "devin", payload); notes != nil {
		t.Fatalf("invalid regex must be skipped, got %v", notes)
	}
}

func TestAPIRequestLogBuilderWritesNotes(t *testing.T) {
	info := UpstreamRequestLog{
		URL:    "https://example.invalid/upstream",
		Method: "POST",
		Notes:  []string{"tool-policy-observe: rule=x tool=Read", "   ", ""},
	}
	builder := newAPIRequestLogBuilder(1, info, time.Now())
	out := builder.String()
	if !strings.Contains(out, "Note: tool-policy-observe: rule=x tool=Read\n") {
		t.Fatalf("expected Note line in request log, got:\n%s", out)
	}
	if strings.Count(out, "Note:") != 1 {
		t.Fatalf("blank notes must be skipped, got:\n%s", out)
	}
}
