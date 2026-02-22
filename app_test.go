package main

import "testing"

func TestResolveProgram_FullPath(t *testing.T) {
	got, err := resolveProgram("/usr/bin/env foo bar")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/usr/bin/env foo bar" {
		t.Errorf("got %q, want %q", got, "/usr/bin/env foo bar")
	}
}

func TestResolveProgram_QuotedArgs(t *testing.T) {
	input := `/usr/bin/env --msg "hello world"`
	got, err := resolveProgram(input)
	if err != nil {
		t.Fatal(err)
	}
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestResolveProgram_Empty(t *testing.T) {
	got, err := resolveProgram("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestResolveProgram_ResolvesCommand(t *testing.T) {
	got, err := resolveProgram("ls")
	if err != nil {
		t.Fatal(err)
	}
	if got == "ls" {
		t.Error("expected ls to be resolved to full path")
	}
	if got == "" {
		t.Error("expected non-empty result")
	}
}

func TestResolveProgram_ResolvesCommandWithArgs(t *testing.T) {
	got, err := resolveProgram("ls -la")
	if err != nil {
		t.Fatal(err)
	}
	if got == "ls -la" {
		t.Error("expected ls to be resolved to full path")
	}
	if len(got) < 4 {
		t.Errorf("unexpected result: %q", got)
	}
}
