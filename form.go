package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// formState はフォーム入力用のローカル変数をまとめた構造体。
// Configへの変換前の中間状態を保持する。
type formState struct {
	processType    string
	keepAlive      string
	scheduleType   string
	directInstall  bool
	intervalStr    string
	minuteStr     string
	hourStr       string
	dayStr        string
	monthStr      string
	weekdayStr    string
	envVarsStr     string
	stdoutPath     string
	stderrPath     string
}

// formSubmitMsg はフォーム送信時に発行されるメッセージ
type formSubmitMsg struct{}

// openDetailMsg はドリルダウン画面への遷移を要求するメッセージ
type openDetailMsg struct {
	kind detailKind
}

// formModel はフォーム画面のモデル
type formModel struct {
	config   *Config
	state    *formState
	fields   []Field
	focused  int
	width    int
	height   int
	// スクロール: 表示開始行のオフセット
	scrollOffset int
	// エラーメッセージ（送信失敗時にフッター付近に表示）
	errMsg string
}

func newFormModel(c *Config, s *formState) formModel {
	m := formModel{config: c, state: s}
	m.fields = m.buildFields()
	return m
}

// buildFields はConfig/formStateのポインタにバインドされたフィールドを生成する
func (m *formModel) buildFields() []Field {
	c := m.config
	s := m.state
	return []Field{
		// 1. Label
		NewTextInputField("Label", "LaunchAgentの識別子（例: com.user.mytool）", &c.Label, WithRequired()),

		// 2. Program
		NewTextInputField("Program", "実行するコマンド", &c.Program, WithRequired(),
			WithDefaultFunc(func() string {
				if c.Label == "" {
					return ""
				}
				// com.user.mytool → mytool
				parts := strings.Split(c.Label, ".")
				return parts[len(parts)-1]
			})),

		// 3. WorkingDirectory
		NewTextInputField("WorkingDirectory", "作業ディレクトリ（省略可）", &c.WorkingDirectory),

		// 4. 環境変数（Enterでドリルダウンへ遷移）
		NewDrillDownField("環境変数", detailEnvVars, func() string {
			return strings.TrimSpace(s.envVarsStr)
		}),

		// 5. ProcessType
		NewSelectField("ProcessType", []SelectOption{
			{"Standard（デフォルト）", "Standard"},
			{"Background（リソース制限あり）", "Background"},
			{"Interactive（リソース制限なし）", "Interactive"},
		}, &s.processType),

		// 6. RunAtLoad
		NewConfirmField("RunAtLoad", "ロード時に即実行する", &c.RunAtLoad),

		// 7. スケジュール（interval/calendar → ドリルダウンへ遷移）
		NewSelectField("スケジュール", []SelectOption{
			{"なし", "none"},
			{"Interval", "interval"},
			{"Calendar", "calendar"},
		}, &s.scheduleType,
			WithOnEnterFunc(func() tea.Cmd {
				switch s.scheduleType {
				case string(ScheduleInterval):
					return func() tea.Msg { return openDetailMsg{kind: detailInterval} }
				case string(ScheduleCalendar):
					return func() tea.Msg { return openDetailMsg{kind: detailCalendar} }
				}
				return nil
			}),
			WithOptionDetailFn(func(value string) string {
				if d := scheduleOptionDetail(s, value); d != "" {
					return d
				}
				if (value == string(ScheduleInterval) || value == string(ScheduleCalendar)) && value == s.scheduleType {
					return "enter で編集"
				}
				return ""
			}),
			WithSelectValidateFunc(func(value string) string {
				if (value == string(ScheduleInterval) || value == string(ScheduleCalendar)) && scheduleOptionDetail(s, value) == "" {
					return "enter で詳細を入力してください"
				}
				return ""
			}),
		),

		// 8. KeepAlive
		NewSelectField("KeepAlive", []SelectOption{
			{"しない", "none"},
			{"常に再起動", "always"},
			{"異常終了時のみ再起動", "on_failure"},
		}, &s.keepAlive),

		// 9. StandardOutPath（ドリルダウンでパス編集）
		NewDrillDownField("StandardOutPath", detailStdoutPath, func() string {
			return s.stdoutPath
		}),

		// 10. StandardErrorPath（ドリルダウンでパス編集）
		NewDrillDownField("StandardErrorPath", detailStderrPath, func() string {
			return s.stderrPath
		}),

		// 11. アクション選択
		NewConfirmField("アクション", "", &s.directInstall,
			WithLabels("プレビュー", "インストール"), WithReverseOrder(), WithButtonStyle()),
	}
}

func (m formModel) Init() tea.Cmd {
	if len(m.fields) > 0 {
		return m.fields[0].Focus()
	}
	return nil
}

func (m formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// フォーカス中のフィールドにメッセージを委譲
	return m.updateFocused(msg)
}

func (m formModel) handleKey(msg tea.KeyMsg) (formModel, tea.Cmd) {
	// キー操作時にエラーメッセージをクリア
	m.errMsg = ""

	switch msg.String() {
	case "enter":
		return m.handleAdvance(true)
	case "tab":
		return m.handleAdvance(false)
	case "shift+tab":
		return m.moveFocus(-1)
	case "ctrl+c":
		return m, tea.Quit
	}

	// フォーカス中のフィールドにキーを委譲
	return m.updateFocused(msg)
}

// handleAdvance はフォーカスを次へ進める。
// drillDown=true（Enter）ならドリルダウン遷移・ボタン確定を判定し、
// false（Tab）なら次のフィールドへ移動する。
func (m formModel) handleAdvance(drillDown bool) (formModel, tea.Cmd) {
	visible := m.visibleFields()

	// Enterの場合
	if drillDown {
		// ドリルダウン遷移を判定
		if cmd := m.checkDrillDown(); cmd != nil {
			return m, cmd
		}
		// ボタンフィールドなら送信
		if m.focused < len(visible) {
			if cf, ok := visible[m.focused].(*ConfirmField); ok && cf.buttonStyle {
				return m, func() tea.Msg { return formSubmitMsg{} }
			}
		}
	}

	// バリデーションチェック
	if m.focused < len(visible) {
		if errMsg := visible[m.focused].Validate(); errMsg != "" {
			return m, nil
		}
	}

	// 最終フィールドならそれ以上進まない
	if m.focused >= len(visible)-1 {
		return m, nil
	}

	return m.moveFocus(1)
}

// checkDrillDown は現在のフィールドの種類に応じてドリルダウンメッセージを返す
func (m formModel) checkDrillDown() tea.Cmd {
	visible := m.visibleFields()
	if m.focused >= len(visible) {
		return nil
	}
	f := visible[m.focused]

	// DrillDownField: 常にドリルダウン遷移
	if df, ok := f.(*DrillDownField); ok {
		return func() tea.Msg { return openDetailMsg{kind: df.kind} }
	}

	// SelectField: onEnterFnが設定されていれば呼び出す
	if sf, ok := f.(*SelectField); ok && sf.onEnterFn != nil {
		return sf.onEnterFn()
	}

	return nil
}

func (m formModel) updateFocused(msg tea.Msg) (formModel, tea.Cmd) {
	visible := m.visibleFields()
	if m.focused >= len(visible) {
		return m, nil
	}

	idx := m.absoluteIndex(m.focused)
	updated, cmd := m.fields[idx].Update(msg)
	m.fields[idx] = updated
	return m, cmd
}

// moveFocus はフォーカスをdelta分移動する
func (m formModel) moveFocus(delta int) (formModel, tea.Cmd) {
	visible := m.visibleFields()
	if len(visible) == 0 {
		return m, nil
	}

	// 現在のフィールドをBlur
	if m.focused < len(visible) {
		visible[m.focused].Blur()
	}

	m.focused = max(m.focused+delta, 0)
	m.focused = min(m.focused, len(visible)-1)

	m.adjustScroll()

	// 新しいフィールドをFocus
	cmd := visible[m.focused].Focus()
	return m, cmd
}

// FocusField はフォーカスを指定されたvisibleインデックスに移動する
func (m *formModel) FocusField(visibleIdx int) tea.Cmd {
	visible := m.visibleFields()
	if len(visible) == 0 {
		return nil
	}

	// 現在のフィールドをBlur
	if m.focused < len(visible) {
		visible[m.focused].Blur()
	}

	m.focused = max(visibleIdx, 0)
	if m.focused >= len(visible) {
		m.focused = len(visible) - 1
	}

	m.adjustScroll()
	return visible[m.focused].Focus()
}

// rebuildAndFocus はフィールドを再構築し、現在のフォーカス位置を復元する
func (m *formModel) rebuildAndFocus() tea.Cmd {
	m.fields = m.buildFields()
	return m.FocusField(m.focused)
}

// adjustScroll はフォーカス位置に応じてスクロールオフセットを調整する
func (m *formModel) adjustScroll() {
	if m.height == 0 {
		return
	}

	visible := m.visibleFields()
	usableHeight := m.height - 2 // フッター分

	// フォーカス位置までの累積行数を計算
	// Join("\n\n") により各フィールド間に空行1行が入る
	topY := 0
	for i := 0; i < m.focused && i < len(visible); i++ {
		topY += visible[i].Height() + 1 // +1 はフィールド間の空行
	}
	focusedH := 0
	if m.focused < len(visible) {
		focusedH = visible[m.focused].Height()
	}
	bottomY := topY + focusedH

	// 上にスクロール
	if topY < m.scrollOffset {
		m.scrollOffset = topY
	}
	// 下にスクロール
	if bottomY > m.scrollOffset+usableHeight {
		m.scrollOffset = bottomY - usableHeight
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m formModel) View() string {
	visible := m.visibleFields()
	if len(visible) == 0 {
		return ""
	}

	// 各フィールドのViewを行として構築
	bar := focusBarStyle.Render("│")
	pad := strings.Repeat(" ", focusBarWidth)
	var lines []string
	for i, f := range visible {
		view := f.View()
		if i == m.focused {
			// フォーカス中のフィールドに色付きバーを付加
			var decorated []string
			for _, line := range strings.Split(view, "\n") {
				decorated = append(decorated, bar+" "+line)
			}
			view = strings.Join(decorated, "\n")
		} else {
			// 非フォーカスはバー幅分のパディング
			var padded []string
			for _, line := range strings.Split(view, "\n") {
				padded = append(padded, pad+line)
			}
			view = strings.Join(padded, "\n")
		}
		lines = append(lines, view)
	}
	content := strings.Join(lines, "\n\n")

	// スクロール処理
	allLines := strings.Split(content, "\n")
	usableHeight := m.height - 2 // フッター分
	if usableHeight < 1 {
		usableHeight = len(allLines)
	}

	start := min(m.scrollOffset, len(allLines))
	end := min(start+usableHeight, len(allLines))

	var b strings.Builder
	b.WriteString(strings.Join(allLines[start:end], "\n"))
	b.WriteString("\n\n")
	if m.errMsg != "" {
		b.WriteString(errorStyle.Render("  "+m.errMsg) + "\n")
	}
	b.WriteString(formFooterStyle.Render("tab 次へ · shift+tab 戻る · enter 確定/詳細 · ctrl+c 終了"))

	return b.String()
}

// visibleFields は現在表示可能なフィールドのみを返す
func (m formModel) visibleFields() []Field {
	var result []Field
	for _, f := range m.fields {
		if f.Visible() {
			result = append(result, f)
		}
	}
	return result
}

// absoluteIndex はvisibleインデックスからm.fieldsの絶対インデックスを返す
func (m formModel) absoluteIndex(visibleIdx int) int {
	count := 0
	for i, f := range m.fields {
		if f.Visible() {
			if count == visibleIdx {
				return i
			}
			count++
		}
	}
	return len(m.fields) - 1
}

// ---------- ユーティリティ関数 ----------

// scheduleOptionDetail はスケジュールオプションの詳細文字列を返す。
// ドリルダウンで値が入力済みの場合のみ文字列を返し、未入力なら空文字列を返す。
func scheduleOptionDetail(s *formState, optionValue string) string {
	switch optionValue {
	case string(ScheduleInterval):
		if s.intervalStr != "" {
			return s.intervalStr + "秒"
		}
	case string(ScheduleCalendar):
		parts := []struct {
			label string
			value string
		}{
			{"月", s.monthStr},
			{"日", s.dayStr},
			{"曜日", s.weekdayStr},
			{"時", s.hourStr},
			{"分", s.minuteStr},
		}
		var items []string
		for _, p := range parts {
			if p.value != "" {
				items = append(items, p.label+"="+p.value)
			}
		}
		if len(items) > 0 {
			return strings.Join(items, " ")
		}
	}
	return ""
}

// applyFormValues はフォームの入力値（formState）をConfigに反映する
func applyFormValues(c *Config, s *formState) error {
	// Programが空欄ならLabelの末尾を使用
	if c.Program == "" && c.Label != "" {
		parts := strings.Split(c.Label, ".")
		c.Program = parts[len(parts)-1]
	}

	c.ProcessType = ProcessType(s.processType)
	c.KeepAlive = KeepAliveType(s.keepAlive)
	c.ScheduleType = ScheduleType(s.scheduleType)

	if s.scheduleType == string(ScheduleInterval) && s.intervalStr != "" {
		n, _ := strconv.Atoi(s.intervalStr)
		c.StartInterval = n
	} else if s.scheduleType == string(ScheduleInterval) && s.intervalStr == "" {
		c.ScheduleType = ScheduleNone
	}

	if s.scheduleType == string(ScheduleCalendar) {
		ci := CalendarInterval{
			Minute:  parseOptionalInt(s.minuteStr),
			Hour:    parseOptionalInt(s.hourStr),
			Day:     parseOptionalInt(s.dayStr),
			Month:   parseOptionalInt(s.monthStr),
			Weekday: parseOptionalInt(s.weekdayStr),
		}
		if ci.HasValue() {
			c.Calendar = ci
		} else {
			c.ScheduleType = ScheduleNone
		}
	}

	c.EnvironmentVars = parseEnvVars(s.envVarsStr)

	stdoutPath, err := toAbsPath(s.stdoutPath)
	if err != nil {
		return err
	}
	c.StdoutPath = stdoutPath

	stderrPath, err := toAbsPath(s.stderrPath)
	if err != nil {
		return err
	}
	c.StderrPath = stderrPath

	return nil
}

// toAbsPath は~展開と絶対パス変換を行う。空文字列はそのまま返す。
func toAbsPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("絶対パスの解決に失敗: %w", err)
	}
	return abs, nil
}

// parseEnvVars は "KEY=VALUE\nKEY2=VALUE2" 形式の文字列をmapに変換する
func parseEnvVars(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	result := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			result[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// parseOptionalInt は空文字列ならnil、数値文字列なら*intを返す
func parseOptionalInt(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
