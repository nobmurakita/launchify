package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailModel_Interval_Confirm(t *testing.T) {
	s := &formState{intervalStr: ""}
	m := newDetailModel(detailInterval, &Config{}, s, 80, 24)
	m.Init()

	// 値を入力
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("300")})

	// Enter で確定
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected detailDoneMsg cmd")
	}
	msg := cmd()
	dm, ok := msg.(detailDoneMsg)
	if !ok {
		t.Fatalf("expected detailDoneMsg, got %T", msg)
	}
	if dm.cancelled {
		t.Error("should not be cancelled")
	}
	if s.intervalStr != "300" {
		t.Errorf("intervalStr = %q, want %q", s.intervalStr, "300")
	}
}

func TestDetailModel_Interval_Cancel(t *testing.T) {
	s := &formState{intervalStr: "existing"}
	m := newDetailModel(detailInterval, &Config{}, s, 80, 24)
	m.Init()

	// Esc でキャンセル
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected detailDoneMsg cmd")
	}
	msg := cmd()
	dm, ok := msg.(detailDoneMsg)
	if !ok {
		t.Fatalf("expected detailDoneMsg, got %T", msg)
	}
	if !dm.cancelled {
		t.Error("should be cancelled")
	}
	// キャンセル時は値が変わらない
	if s.intervalStr != "existing" {
		t.Errorf("intervalStr should remain %q, got %q", "existing", s.intervalStr)
	}
}

func TestDetailModel_Calendar_Navigation(t *testing.T) {
	s := &formState{}
	m := newDetailModel(detailCalendar, &Config{}, s, 80, 24)
	m.Init()

	if len(m.fields) != 5 {
		t.Fatalf("calendar fields count = %d, want 5", len(m.fields))
	}

	// 最初のフィールドに値を入力
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("30")})

	// Tab で次のフィールドへ
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focused != 1 {
		t.Errorf("focused = %d, want 1", m.focused)
	}

	// Shift+Tab で前へ
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.focused != 0 {
		t.Errorf("focused = %d, want 0", m.focused)
	}
}

func TestDetailModel_Calendar_Confirm(t *testing.T) {
	s := &formState{}
	m := newDetailModel(detailCalendar, &Config{}, s, 80, 24)
	m.Init()

	// 月（最初のフィールド）に6を入力
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})

	// 最後のフィールドまで進む
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	// 最後のフィールドでEnter → 確定
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected detailDoneMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(detailDoneMsg); !ok {
		t.Fatalf("expected detailDoneMsg, got %T", msg)
	}
	if s.monthStr != "6" {
		t.Errorf("monthStr = %q, want %q", s.monthStr, "6")
	}
}

func TestDetailModel_EnvVars_TabConfirms(t *testing.T) {
	s := &formState{envVarsStr: ""}
	m := newDetailModel(detailEnvVars, &Config{}, s, 80, 24)
	m.Init()

	// テキストを入力
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("FOO=bar")})

	// Tab で確定
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd == nil {
		t.Fatal("expected detailDoneMsg cmd")
	}
	msg := cmd()
	dm, ok := msg.(detailDoneMsg)
	if !ok {
		t.Fatalf("expected detailDoneMsg, got %T", msg)
	}
	if dm.cancelled {
		t.Error("should not be cancelled")
	}
	if s.envVarsStr != "FOO=bar" {
		t.Errorf("envVarsStr = %q, want %q", s.envVarsStr, "FOO=bar")
	}
}

func TestDetailModel_View_Interval(t *testing.T) {
	s := &formState{}
	m := newDetailModel(detailInterval, &Config{}, s, 80, 24)
	m.Init()

	view := m.View()
	if !strings.Contains(view, "StartInterval") {
		t.Error("View() should contain StartInterval title")
	}
}

func TestDetailModel_View_Calendar(t *testing.T) {
	s := &formState{}
	m := newDetailModel(detailCalendar, &Config{}, s, 80, 24)
	m.Init()

	view := m.View()
	if !strings.Contains(view, "Minute") {
		t.Error("View() should contain Minute label")
	}
}

func TestDetailModel_View_EnvVars(t *testing.T) {
	s := &formState{}
	m := newDetailModel(detailEnvVars, &Config{}, s, 80, 24)
	m.Init()

	view := m.View()
	if !strings.Contains(view, "環境変数") {
		t.Error("View() should contain 環境変数 title")
	}
}
