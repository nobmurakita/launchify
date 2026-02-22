package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlistPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, "Library", "LaunchAgents", "com.example.app.plist")

	got, err := PlistPath("com.example.app")
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Errorf("PlistPath() = %q, want %q", got, expected)
	}
}

func TestPlistPath_ContainsLabel(t *testing.T) {
	got, err := PlistPath("com.test.myservice")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "com.test.myservice.plist") {
		t.Errorf("PlistPath() = %q, should end with label.plist", got)
	}
}
