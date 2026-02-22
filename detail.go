package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// detailKind はドリルダウン画面の種類を表す
type detailKind int

const (
	detailInterval   detailKind = iota // StartInterval 入力
	detailCalendar                     // 分/時/日/月/曜日 入力
	detailEnvVars                      // 環境変数テキストエリア
	detailStdoutPath                   // StandardOutPath 入力
	detailStderrPath                   // StandardErrorPath 入力
)

// detailDoneMsg はドリルダウン画面の完了を通知するメッセージ
type detailDoneMsg struct {
	cancelled bool
}

// detailModel はドリルダウン画面のモデル
type detailModel struct {
	kind     detailKind
	title    string         // 画面タイトル
	footer   string         // フッターテキスト
	config   *Config
	state    *formState
	fields   []textinput.Model // Interval/Calendar 用
	labels   []string          // フィールドラベル
	textarea textarea.Model    // EnvVars 用
	focused  int
	width    int
	height   int
}

func newDetailModel(kind detailKind, c *Config, s *formState, width, height int) detailModel {
	m := detailModel{
		kind:   kind,
		config: c,
		state:  s,
		width:  width,
		height: height,
	}

	switch kind {
	case detailInterval:
		m.title = "StartInterval"
		m.footer = "enter/tab 次へ · esc キャンセル"
		m.fields, m.labels = buildIntervalFields(s)
	case detailCalendar:
		m.title = "StartCalendarInterval"
		m.footer = "enter/tab 次へ · esc キャンセル"
		m.fields, m.labels = buildCalendarFields(s)
	case detailEnvVars:
		m.title = "環境変数"
		m.footer = "tab 確定 · esc キャンセル"
		m.textarea = buildEnvVarsTextarea(s, width)
	case detailStdoutPath:
		m.title = "StandardOutPath"
		m.footer = "enter/tab 次へ · esc キャンセル"
		m.fields, m.labels = buildLogPathField(s.stdoutPath, c.Label)
	case detailStderrPath:
		m.title = "StandardErrorPath"
		m.footer = "enter/tab 次へ · esc キャンセル"
		m.fields, m.labels = buildLogPathField(s.stderrPath, c.Label)
	}

	return m
}

func buildIntervalFields(s *formState) ([]textinput.Model, []string) {
	ti := textinput.New()
	ti.Placeholder = "例: 300"
	ti.SetValue(s.intervalStr)
	ti.Focus()
	ti.Prompt = "  "
	ti.Cursor.Style = focusedCursorStyle
	return []textinput.Model{ti}, []string{"StartInterval（秒）"}
}

func buildCalendarFields(s *formState) ([]textinput.Model, []string) {
	values := []string{s.monthStr, s.dayStr, s.weekdayStr, s.hourStr, s.minuteStr}
	labels := []string{
		"月 (Month) 1-12",
		"日 (Day) 1-31",
		"曜日 (Weekday) 0=日..6=土",
		"時 (Hour) 0-23",
		"分 (Minute) 0-59",
	}
	placeholders := []string{"空欄=毎月", "空欄=毎日", "空欄=毎日", "空欄=毎時", "空欄=毎分"}

	fields := make([]textinput.Model, len(values))
	for i, val := range values {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.SetValue(val)
		ti.Prompt = "  "
		ti.Cursor.Style = focusedCursorStyle
		if i == 0 {
			ti.Focus()
		}
		fields[i] = ti
	}
	return fields, labels
}

func buildLogPathField(currentPath, label string) ([]textinput.Model, []string) {
	val := currentPath
	if val == "" {
		val = "~/Library/Logs/" + label + ".log"
	}
	ti := textinput.New()
	ti.Placeholder = "ファイルパス"
	ti.SetValue(val)
	ti.Focus()
	ti.Prompt = "  "
	ti.Cursor.Style = focusedCursorStyle
	return []textinput.Model{ti}, []string{"出力先パス"}
}

func buildEnvVarsTextarea(s *formState, width int) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "KEY=VALUE 形式で1行ずつ入力"
	ta.SetValue(s.envVarsStr)
	ta.Focus()
	ta.ShowLineNumbers = false
	if width > 4 {
		ta.SetWidth(width - 4)
	}
	ta.SetHeight(8)
	ta.Prompt = "  "
	return ta
}

func (m detailModel) Init() tea.Cmd {
	if m.kind == detailEnvVars {
		return textarea.Blink
	}
	if len(m.fields) > 0 {
		return m.fields[0].Focus()
	}
	return nil
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.kind == detailEnvVars && m.width > 4 {
			m.textarea.SetWidth(m.width - 4)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateFocused(msg)
}

func (m detailModel) handleKey(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return detailDoneMsg{cancelled: true} }
	case "ctrl+c":
		return m, tea.Quit
	}

	if m.kind == detailEnvVars {
		return m.handleEnvVarsKey(msg)
	}
	return m.handleFieldsKey(msg)
}

func (m detailModel) handleEnvVarsKey(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	// Tab で確定（textarea内ではEnterは改行に使うため）
	if msg.String() == "tab" {
		m.state.envVarsStr = m.textarea.Value()
		return m, func() tea.Msg { return detailDoneMsg{cancelled: false} }
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m detailModel) handleFieldsKey(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	switch msg.String() {
	case "tab", "enter":
		// 次のフィールドへ移動、最後なら確定
		if m.focused >= len(m.fields)-1 {
			m.applyFieldValues()
			return m, func() tea.Msg { return detailDoneMsg{cancelled: false} }
		}
		m.fields[m.focused].Blur()
		m.focused++
		cmd := m.fields[m.focused].Focus()
		return m, cmd

	case "shift+tab":
		if m.focused > 0 {
			m.fields[m.focused].Blur()
			m.focused--
			cmd := m.fields[m.focused].Focus()
			return m, cmd
		}
		return m, nil
	}

	// フォーカス中フィールドに委譲
	return m.updateFocused(msg)
}

func (m detailModel) updateFocused(msg tea.Msg) (detailModel, tea.Cmd) {
	if m.kind == detailEnvVars {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	if m.focused < len(m.fields) {
		var cmd tea.Cmd
		m.fields[m.focused], cmd = m.fields[m.focused].Update(msg)
		return m, cmd
	}
	return m, nil
}

// applyFieldValues はフィールドの値をformStateに書き戻す
func (m *detailModel) applyFieldValues() {
	switch m.kind {
	case detailInterval:
		if len(m.fields) > 0 {
			m.state.intervalStr = m.fields[0].Value()
		}
	case detailCalendar:
		vals := make([]string, len(m.fields))
		for i, f := range m.fields {
			vals[i] = f.Value()
		}
		if len(vals) >= 5 {
			m.state.monthStr = vals[0]
			m.state.dayStr = vals[1]
			m.state.weekdayStr = vals[2]
			m.state.hourStr = vals[3]
			m.state.minuteStr = vals[4]
		}
	case detailStdoutPath:
		if len(m.fields) > 0 {
			m.state.stdoutPath = m.fields[0].Value()
		}
	case detailStderrPath:
		if len(m.fields) > 0 {
			m.state.stderrPath = m.fields[0].Value()
		}
	}
}

func (m detailModel) View() string {
	var b strings.Builder

	// タイトル
	b.WriteString(detailTitleStyle.Render(m.title))
	b.WriteString("\n\n")

	// 入力フィールド
	if m.kind == detailEnvVars {
		b.WriteString(m.textarea.View())
	} else {
		for i, f := range m.fields {
			label := m.labels[i]
			if i == m.focused {
				b.WriteString(focusedTitleStyle.Render(label))
				b.WriteString("\n")
				b.WriteString(f.View())
			} else {
				val := f.Value()
				style := blurredValueStyle
				if val == "" {
					val = f.Placeholder
					style = blurredMutedStyle
				}
				if val == "" {
					val = "-"
					style = blurredMutedStyle
				}
				b.WriteString(blurredTitleStyle.Render(label))
				b.WriteString("\n")
				b.WriteString("  ")
				b.WriteString(style.Render(val))
			}
			if i < len(m.fields)-1 {
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n\n")
	b.WriteString(detailFooterStyle.Render(m.footer))

	return b.String()
}
