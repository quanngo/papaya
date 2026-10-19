package rest

import (
	"testing"

	"claimconfig/i18n"
)

func TestReadableParams(t *testing.T) {
	vi := readableParams("vi", map[string]any{
		"claim_type": "DENTAL", "enabled": "OUTPATIENT, INPATIENT", "event": "claim_submitted",
		"channels": "email, sms", "deadline": "2026-10-23", "amount": 1500000.5, "days": 5, "role": "team_lead",
	})
	want := map[string]any{
		"claim_type": "Nha khoa", "enabled": "Ngoại trú, Nội trú", "event": "Đã nộp yêu cầu",
		"channels": "Email, SMS", "deadline": "23/10/2026", "amount": "1.500.000,5", "days": "5", "role": "Trưởng nhóm",
	}
	for k, w := range want {
		if vi[k] != w {
			t.Errorf("vi %s = %v, want %v", k, vi[k], w)
		}
	}
	en := readableParams("en", map[string]any{"amount": 15000.0, "deadline": "2026-10-23", "claim_type": "UNKNOWN"})
	if en["amount"] != "15,000" || en["deadline"] != "Oct 23, 2026" || en["claim_type"] != "UNKNOWN" {
		t.Errorf("en = %v", en)
	}
}

func TestTraceTextsAreReadable(t *testing.T) {
	got := i18n.T("vi", "trace.TIER_MATCHED", readableParams("vi", map[string]any{"amount": 15000.0, "min": 5000.0, "max": 50000.0, "role": "assessor"}))
	if got != "Số tiền 15.000 thuộc cấp từ 5.000 đến dưới 50.000: chuyển cho Chuyên viên thẩm định duyệt." {
		t.Errorf("got %q", got)
	}
	got = i18n.T("en", "issue.CLAIM_TYPE_NOT_ENABLED", readableParams("en", map[string]any{"claim_type": "DENTAL", "enabled": "OUTPATIENT, INPATIENT"}))
	if got != "This insurer does not offer Dental (enabled: Outpatient, Inpatient)." {
		t.Errorf("got %q", got)
	}
}

func TestPatternExample(t *testing.T) {
	cases := map[string]string{
		"^[A-Z]{3}-[0-9]{4}$": "ABC-1234",
		"^EMP-[0-9]{5}$":      "EMP-12345",
		`^NV\.\d{2}$`:         "NV.12",
		"^[a-z]{2}[0-9]$":     "ab1",
		"^(a|b)+$":            "", // beyond simple masks
		"[0-9]{3}":            "", // not anchored
	}
	for re, want := range cases {
		if got := patternExample(re); got != want {
			t.Errorf("patternExample(%q) = %q, want %q", re, got, want)
		}
	}
}

func TestPatternMismatchNeverShowsRegex(t *testing.T) {
	got := issueText("vi", "FIELD_PATTERN_MISMATCH", map[string]any{"field": "Budget Code", "pattern": "^[A-Z]{3}-[0-9]{4}$"})
	if got != `Giá trị của "Budget Code" không đúng định dạng. Ví dụ đúng: ABC-1234.` {
		t.Errorf("got %q", got)
	}
	got = issueText("en", "FIELD_PATTERN_MISMATCH", map[string]any{"field": "Code", "pattern": "^(a|b)+$"})
	if got != `The value of "Code" does not match the format the insurer requires.` {
		t.Errorf("got %q", got)
	}
}
