package main

import (
	"strings"
	"testing"
)

func intPtr(n int) *int {
	return &n
}

func mustContain(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("plistに %q が含まれていません:\n%s", substr, s)
	}
}

func mustNotContain(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("plistに %q が含まれるべきではありません", substr)
	}
}
