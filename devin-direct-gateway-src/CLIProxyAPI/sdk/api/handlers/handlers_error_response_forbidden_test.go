package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Upstream 403s are permission conditions; only bodies that self-identify as
// quota or credit exhaustion may keep the insufficient_quota code. Content-policy
// rejections (permission_denied trailers) must not read as quota exhaustion.
func TestBuildErrorResponseBody_ForbiddenSplitsQuotaFromPermission(t *testing.T) {
	policyBody := BuildErrorResponseBody(http.StatusForbidden,
		"devin upstream error (permission_denied): Your request was blocked by our content policy. Please remove sensitive or unsafe content from your prompt, memories, and other settings and try again.")
	var policy struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(policyBody, &policy); err != nil {
		t.Fatalf("policy body is not valid JSON: %v (%s)", err, policyBody)
	}
	if policy.Error.Type != "permission_error" {
		t.Fatalf("policy type = %q, want permission_error", policy.Error.Type)
	}
	if policy.Error.Code != "upstream_permission_denied" {
		t.Fatalf("policy code = %q, want upstream_permission_denied", policy.Error.Code)
	}
	if !strings.Contains(policy.Error.Message, "content policy") {
		t.Fatalf("policy message was not preserved: %q", policy.Error.Message)
	}

	quotaBody := BuildErrorResponseBody(http.StatusForbidden, "organization has insufficient quota for this model")
	var quota struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(quotaBody, &quota); err != nil {
		t.Fatalf("quota body is not valid JSON: %v (%s)", err, quotaBody)
	}
	if quota.Error.Code != "insufficient_quota" {
		t.Fatalf("quota code = %q, want insufficient_quota kept for quota-identified 403s", quota.Error.Code)
	}
}
