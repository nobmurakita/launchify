package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PreviewResult はプレビュー画面でのユーザーの選択結果を表す
type PreviewResult int

const (
	PreviewInstall PreviewResult = iota // インストール実行
	PreviewBack                         // フォームに戻る
	PreviewQuit                         // 終了
)

// previewModel はplist XMLのスクロール可能なプレビュー画面のモデル
type previewModel struct {
	viewport   viewport.Model
	content    string // plist XMLの全文
	plistPath  string // 既存ファイル警告の表示用
	fileExists bool   // 既存ファイルがあるかどうか
	result     PreviewResult
	quitting   bool
	ready      bool // viewport初期化済みフラグ
}

// ヘッダー1行 + 空行 = 2行、フッター（警告行含む）= 最大2行 + 空行 = 3行
const (
	headerHeight = 2
	footerHeight = 3
)

// プレビュー画面用のスタイル定義
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0891b2", Dark: "#22d3ee"}).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#ca8a04", Dark: "#facc15"})

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#a1a1aa", Dark: "#71717a"})
)

// newPreviewModel はプレビュー画面のモデルを生成する
func newPreviewModel(content, plistPath string) tea.Model {
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

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
		m.viewport.SetContent(m.content)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.result = PreviewInstall
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEscape:
			m.result = PreviewBack
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.result = PreviewQuit
			m.quitting = true
			return m, tea.Quit
		case tea.KeyRunes:
			if string(msg.Runes) == "q" {
				m.result = PreviewQuit
				m.quitting = true
				return m, tea.Quit
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
	b.WriteString(titleStyle.Render("生成されるplist"))
	b.WriteString("\n\n")

	// plist XMLのスクロール表示
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// 既存ファイル警告（存在する場合のみ）
	if m.fileExists {
		warning := fmt.Sprintf("⚠ %s は既に存在します（上書きされます）", m.plistPath)
		b.WriteString(warningStyle.Render(warning))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// フッター（操作ガイド）
	b.WriteString(footerStyle.Render("enter インストール · esc 戻る · q 終了"))

	return b.String()
}

// RunPreview はプレビュー画面を表示し、ユーザーの選択結果を返す
func RunPreview(content, plistPath string) PreviewResult {
	p := tea.NewProgram(newPreviewModel(content, plistPath), tea.WithAltScreen())
	finalModel, _ := p.Run()
	return finalModel.(previewModel).result
}
