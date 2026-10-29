package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

type OAuthToolPolicyObserveSummary struct {
	hash  string
	count int
}

// SummarizeOAuthToolPolicyObserve summarizes OAuth tool-policy observe rules per channel.
func SummarizeOAuthToolPolicyObserve(entries map[string][]config.ToolPolicyObserveRule) map[string]OAuthToolPolicyObserveSummary {
	if len(entries) == 0 {
		return nil
	}
	out := make(map[string]OAuthToolPolicyObserveSummary, len(entries))
	for k, v := range entries {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		out[key] = summarizeOAuthToolPolicyObserveList(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DiffOAuthToolPolicyObserveChanges compares OAuth tool-policy observe maps.
func DiffOAuthToolPolicyObserveChanges(oldMap, newMap map[string][]config.ToolPolicyObserveRule) ([]string, []string) {
	oldSummary := SummarizeOAuthToolPolicyObserve(oldMap)
	newSummary := SummarizeOAuthToolPolicyObserve(newMap)
	keys := make(map[string]struct{}, len(oldSummary)+len(newSummary))
	for k := range oldSummary {
		keys[k] = struct{}{}
	}
	for k := range newSummary {
		keys[k] = struct{}{}
	}
	changes := make([]string, 0, len(keys))
	affected := make([]string, 0, len(keys))
	for key := range keys {
		oldInfo, okOld := oldSummary[key]
		newInfo, okNew := newSummary[key]
		switch {
		case okOld && !okNew:
			changes = append(changes, fmt.Sprintf("oauth-tool-policy-observe[%s]: removed", key))
			affected = append(affected, key)
		case !okOld && okNew:
			changes = append(changes, fmt.Sprintf("oauth-tool-policy-observe[%s]: added (%d entries)", key, newInfo.count))
			affected = append(affected, key)
		case okOld && okNew && oldInfo.hash != newInfo.hash:
			changes = append(changes, fmt.Sprintf("oauth-tool-policy-observe[%s]: updated (%d -> %d entries)", key, oldInfo.count, newInfo.count))
			affected = append(affected, key)
		}
	}
	sort.Strings(changes)
	sort.Strings(affected)
	return changes, affected
}

func summarizeOAuthToolPolicyObserveList(list []config.ToolPolicyObserveRule) OAuthToolPolicyObserveSummary {
	if len(list) == 0 {
		return OAuthToolPolicyObserveSummary{}
	}
	var b strings.Builder
	valid := 0
	for _, entry := range list {
		if len(entry.Match) == 0 && len(entry.MatchRegexr) == 0 {
			continue
		}
		valid++
		b.WriteString(fmt.Sprintf("%s|%s|%s\n", entry.ID, strings.Join(entry.Match, ","), strings.Join(entry.MatchRegexr, ",")))
	}
	if valid == 0 {
		return OAuthToolPolicyObserveSummary{}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return OAuthToolPolicyObserveSummary{
		hash:  hex.EncodeToString(sum[:]),
		count: valid,
	}
}
