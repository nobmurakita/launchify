package main

import "github.com/charmbracelet/lipgloss"

// カラーパレット
var (
	cyan      = lipgloss.AdaptiveColor{Light: "#0891b2", Dark: "#22d3ee"}
	dimGray   = lipgloss.AdaptiveColor{Light: "#a1a1aa", Dark: "#71717a"}
	lightGray = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#a1a1aa"}
	yellow    = lipgloss.AdaptiveColor{Light: "#ca8a04", Dark: "#facc15"}
)

// フォームフィールド用スタイル
var (
	// フォーカス中のフィールド
	focusedTitleStyle = lipgloss.NewStyle().
				Foreground(cyan).Bold(true)
	focusedDescStyle = lipgloss.NewStyle().
				Foreground(dimGray)
	focusedCursorStyle = lipgloss.NewStyle().
				Foreground(cyan)

	// 非フォーカスのフィールド
	blurredTitleStyle = lipgloss.NewStyle().Foreground(lightGray)
	blurredValueStyle = lipgloss.NewStyle() // デフォルト前景色（値を目立たせる）
	blurredMutedStyle = lipgloss.NewStyle().
				Foreground(dimGray)

	// Select フィールド用
	selectedOptionStyle   = lipgloss.NewStyle().Foreground(cyan)
	unselectedOptionStyle = lipgloss.NewStyle().Foreground(dimGray)

	// Confirm フィールド用
	activeToggleStyle   = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	inactiveToggleStyle = lipgloss.NewStyle().Foreground(dimGray)

	// ボタン風スタイル（フォーカス時）
	activeButtonStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#000")).Background(cyan).Bold(true).Padding(0, 2)
	inactiveButtonStyle = lipgloss.NewStyle().Foreground(dimGray).Padding(0, 2)

	// ボタン風スタイル（非フォーカス時）
	blurredActiveButtonStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#000")).Background(lightGray).Padding(0, 2)
	blurredInactiveButtonStyle = lipgloss.NewStyle().Foreground(dimGray).Padding(0, 2)

	// プレビュー画面用スタイル
	previewTitleStyle   = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	previewWarningStyle = lipgloss.NewStyle().Foreground(yellow)
	previewFooterStyle  = lipgloss.NewStyle().Foreground(dimGray)

	// フォーカスバー（フィールド左端）
	focusBarStyle = lipgloss.NewStyle().Foreground(cyan)
	focusBarWidth = 2 // "│" + space

	// フッター
	formFooterStyle = lipgloss.NewStyle().Foreground(dimGray)

	// バリデーションエラー
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"})

	// ドリルダウン画面用
	detailTitleStyle  = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	detailFooterStyle = lipgloss.NewStyle().Foreground(dimGray)
)
