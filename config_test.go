package main

import "testing"

func TestParseEnvVars(t *testing.T) {
	got := parseEnvVars("PATH=/usr/local/bin\nHOME=/Users/test")
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got["PATH"] != "/usr/local/bin" {
		t.Errorf("PATH: got %q, want /usr/local/bin", got["PATH"])
	}
	if got["HOME"] != "/Users/test" {
		t.Errorf("HOME: got %q, want /Users/test", got["HOME"])
	}
}

func TestParseEnvVars_Empty(t *testing.T) {
	got := parseEnvVars("")
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestParseEnvVars_WithSpaces(t *testing.T) {
	got := parseEnvVars("  KEY = VALUE  \n\n  KEY2=VALUE2  \n")
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got["KEY"] != "VALUE" {
		t.Errorf("KEY: got %q, want VALUE", got["KEY"])
	}
}
