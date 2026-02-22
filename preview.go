package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// previewResult はプレビュー画面でのユーザーの選択結果を表す
type previewResult int

const (
	previewResultInstall previewResult = iota // インストール実行
	previewResultBack                         // フォームに戻る
	previewResultQuit                         // 終了
)

// previewDoneMsg はプレビュー画面の操作結果を通知するメッセージ
type previewDoneMsg struct {
	result previewResult
}

// previewModel はplist XMLのスクロール可能なプレビュー画面のモデル
type previewModel struct {
	viewport   viewport.Model
	content    string // plist XMLの全文
	plistPath  string // 既存ファイル警告の表示用
	fileExists bool   // 既存ファイルがあるかどうか
	ready      bool   // viewport初期化済みフラグ
}

// ヘッダー1行 + 空行 = 2行、フッター（警告行含む）= 最大2行 + 空行 = 3行
const (
	previewHeaderHeight = 2
	previewFooterHeight = 3
)

func newPreviewModel(content, plistPath string) previewModel {
	_, err := os.Stat(plistPath)
	return previewModel{
		content:    content,
		plistPath:  plistPath,
		fileExists: err == nil,
	}
}

func (m previewModel) Init() tea.Cmd {
	return nil
}

func (m previewModel) Update(msg tea.Msg) (previewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport = viewport.New(msg.Width, msg.Height-previewHeaderHeight-previewFooterHeight)
		m.viewport.SetContent(m.content)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, func() tea.Msg { return previewDoneMsg{result: previewResultInstall} }
		case tea.KeyEscape, tea.KeyShiftTab:
			return m, func() tea.Msg { return previewDoneMsg{result: previewResultBack} }
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyRunes:
			if string(msg.Runes) == "q" {
				return m, func() tea.Msg { return previewDoneMsg{result: previewResultQuit} }
			}
		}
	}

	// スクロール操作などはviewportに委譲
	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m previewModel) View() string {
	if !m.ready {
		return ""
	}

	var b strings.Builder

	// タイトル
	b.WriteString(previewTitleStyle.Render("生成されるplist"))
	b.WriteString("\n\n")

	// plist XMLのスクロール表示
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// 既存ファイル警告（存在する場合のみ）
	if m.fileExists {
		warning := fmt.Sprintf("[!] %s は既に存在します（上書きされます）", m.plistPath)
		b.WriteString(previewWarningStyle.Render(warning))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// フッター（操作ガイド）
	b.WriteString(previewFooterStyle.Render("enter インストール · shift+tab/esc 戻る · q 終了"))

	return b.String()
}
