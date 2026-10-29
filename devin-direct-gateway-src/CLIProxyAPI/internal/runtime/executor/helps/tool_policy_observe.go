package helps

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// ObserveToolPolicy scans client tool declarations in the downstream request payload
// against per-provider denylist shape rules (observe-only). It returns annotation
// notes for the request log; matched rules are also emitted as warn-level log events.
// The request itself is never altered or rejected.
func ObserveToolPolicy(cfg *config.Config, provider string, payload []byte) []string {
	if cfg == nil || len(cfg.OAuthToolPolicyObserve) == 0 || len(payload) == 0 {
		return nil
	}
	rules := cfg.OAuthToolPolicyObserve[strings.ToLower(strings.TrimSpace(provider))]
	if len(rules) == 0 {
		return nil
	}
	tools := gjson.GetBytes(payload, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return nil
	}
	var notes []string
	for _, tool := range tools.Array() {
		name := tool.Get("function.name").String()
		if name == "" {
			name = tool.Get("name").String()
		}
		candidates := [][2]string{
			{"function.description", tool.Get("function.description").String()},
			{"description", tool.Get("description").String()},
			{"function.name", tool.Get("function.name").String()},
			{"name", tool.Get("name").String()},
		}
		for ri, rule := range rules {
			ruleID := strings.TrimSpace(rule.ID)
			if ruleID == "" {
				ruleID = fmt.Sprintf("rule-%d", ri+1)
			}
			for _, pattern := range rule.Match {
				for _, cand := range candidates {
					if cand[1] == "" || !strings.Contains(cand[1], pattern) {
						continue
					}
					notes = appendToolPolicyNote(notes, provider, ruleID, name, cand[0], pattern)
				}
			}
			for _, pattern := range rule.MatchRegexr {
				re, errCompile := regexp.Compile(pattern)
				if errCompile != nil {
					continue
				}
				for _, cand := range candidates {
					if cand[1] == "" || !re.MatchString(cand[1]) {
						continue
					}
					notes = appendToolPolicyNote(notes, provider, ruleID, name, cand[0], pattern)
				}
			}
		}
	}
	return notes
}

func appendToolPolicyNote(notes []string, provider, ruleID, toolName, field, pattern string) []string {
	if len(pattern) > 120 {
		pattern = pattern[:120] + "..."
	}
	note := fmt.Sprintf("tool-policy-observe: rule=%s tool=%s field=%s pattern=%s", ruleID, toolName, field, pattern)
	log.WithFields(log.Fields{
		"provider": provider,
		"rule":     ruleID,
		"tool":     toolName,
		"field":    field,
		"pattern":  pattern,
	}).Warn("tool policy observe: client tool declaration matched denylist shape rule")
	return append(notes, note)
}
