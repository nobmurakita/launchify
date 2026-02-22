package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

const testPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.test.example</string>
</dict>
</plist>`

// initPreviewViewport はテスト用にWindowSizeMsgを送信してviewportを初期化する
func initPreviewViewport(m previewModel) previewModel {
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func TestPreviewModel_EnterReturnsInstall(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initPreviewViewport(m)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	dm, ok := msg.(previewDoneMsg)
	if !ok {
		t.Fatalf("expected previewDoneMsg, got %T", msg)
	}
	if dm.result != resultInstall {
		t.Errorf("result = %v, want resultInstall", dm.result)
	}
}

func TestPreviewModel_EscReturnsBack(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initPreviewViewport(m)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	dm, ok := msg.(previewDoneMsg)
	if !ok {
		t.Fatalf("expected previewDoneMsg, got %T", msg)
	}
	if dm.result != resultBack {
		t.Errorf("result = %v, want resultBack", dm.result)
	}
}

func TestPreviewModel_QReturnsQuit(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initPreviewViewport(m)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	dm, ok := msg.(previewDoneMsg)
	if !ok {
		t.Fatalf("expected previewDoneMsg, got %T", msg)
	}
	if dm.result != resultQuit {
		t.Errorf("result = %v, want resultQuit", dm.result)
	}
}

func TestPreviewModel_ViewContainsPlist(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initPreviewViewport(m)

	view := m.View()
	if !strings.Contains(view, "com.test.example") {
		t.Error("View() should contain plist content")
	}
}

func TestPreviewModel_ViewShowsWarningForExistingFile(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m.fileExists = true
	m = initPreviewViewport(m)

	view := m.View()
	if !strings.Contains(view, "既に存在") {
		t.Error("View() should contain warning for existing file")
	}
}

func TestPreviewModel_ViewNoWarningForNewFile(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m.fileExists = false
	m = initPreviewViewport(m)

	view := m.View()
	if strings.Contains(view, "既に存在") {
		t.Error("View() should not contain warning for new file")
	}
}
