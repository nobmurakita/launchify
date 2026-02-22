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

// initViewport はテスト用にWindowSizeMsgを送信してviewportを初期化する
func initViewport(m tea.Model) tea.Model {
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func TestPreviewModel_EnterReturnsInstall(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initViewport(m)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	pm := m.(previewModel)
	if pm.result != PreviewInstall {
		t.Errorf("result = %v, want PreviewInstall", pm.result)
	}
	if !pm.quitting {
		t.Error("quitting should be true")
	}
}

func TestPreviewModel_EscReturnsBack(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initViewport(m)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	pm := m.(previewModel)
	if pm.result != PreviewBack {
		t.Errorf("result = %v, want PreviewBack", pm.result)
	}
	if !pm.quitting {
		t.Error("quitting should be true")
	}
}

func TestPreviewModel_QReturnsQuit(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initViewport(m)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	pm := m.(previewModel)
	if pm.result != PreviewQuit {
		t.Errorf("result = %v, want PreviewQuit", pm.result)
	}
	if !pm.quitting {
		t.Error("quitting should be true")
	}
}

func TestPreviewModel_ViewContainsPlist(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")
	m = initViewport(m)

	view := m.(previewModel).View()
	if !strings.Contains(view, "com.test.example") {
		t.Error("View() should contain plist content")
	}
}

func TestPreviewModel_ViewShowsWarningForExistingFile(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")

	// fileExistsを直接設定してテスト
	pm := m.(previewModel)
	pm.fileExists = true
	m = pm

	m = initViewport(m)

	view := m.(previewModel).View()
	if !strings.Contains(view, "既に存在") {
		t.Error("View() should contain warning for existing file")
	}
}

func TestPreviewModel_ViewNoWarningForNewFile(t *testing.T) {
	m := newPreviewModel(testPlistContent, "/tmp/nonexistent.plist")

	// fileExistsがfalseであることを確認
	pm := m.(previewModel)
	pm.fileExists = false
	m = pm

	m = initViewport(m)

	view := m.(previewModel).View()
	if strings.Contains(view, "既に存在") {
		t.Error("View() should not contain warning for new file")
	}
}
