package config

import (
	"testing"
)

func TestParseConfigOAuthToolPolicyObserve(t *testing.T) {
	const yamlConfig = `
oauth-tool-policy-observe:
  devin:
    - id: "desc-unrestricted-fs-read"
      match:
        - "access any file"
      match-regexr:
        - "(?i)file ?system[\\s\\S]{0,200}access\\s+any\\s+file"
  codex:
    - id: "other-rule"
      match:
        - "some pattern"
`

	cfg, err := ParseConfigBytes([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("ParseConfigBytes failed: %v", err)
	}

	if len(cfg.OAuthToolPolicyObserve) != 2 {
		t.Fatalf("cfg.OAuthToolPolicyObserve len = %d, want 2", len(cfg.OAuthToolPolicyObserve))
	}

	devinRules, ok := cfg.OAuthToolPolicyObserve["devin"]
	if !ok || len(devinRules) != 1 {
		t.Fatalf("devin rules missing or len != 1: %#v", devinRules)
	}
	rule := devinRules[0]
	if rule.ID != "desc-unrestricted-fs-read" {
		t.Errorf("unexpected rule id: %+v", rule)
	}
	if len(rule.Match) != 1 || len(rule.MatchRegexr) != 1 {
		t.Errorf("unexpected match lens: %+v", rule)
	}
}

func TestSanitizeOAuthToolPolicyObserve(t *testing.T) {
	cfg := &Config{
		OAuthToolPolicyObserve: map[string][]ToolPolicyObserveRule{
			" Devin ": {
				{
					ID:          "  good-rule  ",
					Match:       []string{"  needle  ", ""},
					MatchRegexr: []string{"  ^bad.*  ", ""},
				},
				{
					ID: "empty-rule", // no match patterns
				},
				{
					ID:          "bad-regex-rule",
					MatchRegexr: []string{"([invalid"},
				},
			},
			" empty-channel ": {},
		},
	}

	cfg.SanitizeOAuthToolPolicyObserve()

	if len(cfg.OAuthToolPolicyObserve) != 1 {
		t.Fatalf("expected 1 sanitized channel, got %d", len(cfg.OAuthToolPolicyObserve))
	}

	rules := cfg.OAuthToolPolicyObserve["devin"]
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule for devin, got %d", len(rules))
	}
	if rules[0].ID != "good-rule" {
		t.Errorf("unexpected sanitized id: %+v", rules[0])
	}
	if len(rules[0].Match) != 1 || rules[0].Match[0] != "needle" {
		t.Errorf("unexpected sanitized match: %+v", rules[0].Match)
	}
	if len(rules[0].MatchRegexr) != 1 || rules[0].MatchRegexr[0] != "^bad.*" {
		t.Errorf("unexpected sanitized regexr: %+v", rules[0].MatchRegexr)
	}
}

func TestSanitizeOAuthToolPolicyObserveEmpty(t *testing.T) {
	cfg := &Config{OAuthToolPolicyObserve: map[string][]ToolPolicyObserveRule{
		"devin": {{ID: "no-patterns"}},
	}}
	cfg.SanitizeOAuthToolPolicyObserve()
	if cfg.OAuthToolPolicyObserve != nil {
		t.Fatalf("expected nil map after sanitizing all-invalid rules, got %#v", cfg.OAuthToolPolicyObserve)
	}
}
