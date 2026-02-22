package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// formFooterLines はフォームフッターが占める行数（操作ガイド + 空行）
const formFooterLines = 2

// formState はフォーム入力用のローカル変数をまとめた構造体。
// Configへの変換前の中間状態を保持する。
type formState struct {
	processType  string
	keepAlive    string
	scheduleType string

	// directInstall はアクション選択の結果（true=即インストール、false=プレビュー表示）
	directInstall bool
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
			WithOnEnterFunc(m.scheduleOnDrillDownCmd),
			WithOptionDetailFn(m.scheduleDetail),
			WithSelectValidateFunc(m.scheduleValidate),
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
			WithLabels("インストール", "プレビュー"), WithReverseOrder(), WithButtonStyle()),
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
	m.setError("")

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

	// バリデーションチェック（エラー時はフィールドにエラー表示を指示）
	if m.focused < len(visible) {
		if errMsg := visible[m.focused].Validate(); errMsg != "" {
			if tf, ok := visible[m.focused].(*TextInputField); ok {
				tf.showError = true
			}
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

	// SelectField: onDrillDownFnが設定されていれば呼び出す
	if sf, ok := f.(*SelectField); ok && sf.onDrillDownFn != nil {
		return sf.onDrillDownFn()
	}

	return nil
}

func (m formModel) updateFocused(msg tea.Msg) (formModel, tea.Cmd) {
	visible := m.visibleFields()
	if m.focused >= len(visible) {
		return m, nil
	}

	idx := m.fieldIndexFromVisibleIndex(m.focused)
	updated, cmd := m.fields[idx].Update(msg)
	m.fields[idx] = updated
	return m, cmd
}

// moveFocus はフォーカスをdelta分移動する
func (m formModel) moveFocus(delta int) (formModel, tea.Cmd) {
	cmd := m.focusVisibleFieldAt(m.focused + delta)
	return m, cmd
}

// focusVisibleFieldAt はフォーカスを指定されたvisibleインデックスに移動する
func (m *formModel) focusVisibleFieldAt(visibleIdx int) tea.Cmd {
	visible := m.visibleFields()
	if len(visible) == 0 {
		return nil
	}

	// 現在のフィールドをBlur
	if m.focused < len(visible) {
		visible[m.focused].Blur()
	}

	m.focused = max(min(visibleIdx, len(visible)-1), 0)
	m.adjustScroll()
	return visible[m.focused].Focus()
}

// rebuildAndFocus はフィールドを再構築し、現在のフォーカス位置を復元する
func (m *formModel) rebuildAndFocus() tea.Cmd {
	m.fields = m.buildFields()
	return m.focusVisibleFieldAt(m.focused)
}

// setError はエラーメッセージを設定する
func (m *formModel) setError(msg string) {
	m.errMsg = msg
}

// adjustScroll はフォーカス位置に応じてスクロールオフセットを調整する
func (m *formModel) adjustScroll() {
	if m.height == 0 {
		return
	}

	visible := m.visibleFields()
	usableHeight := m.height - formFooterLines // フッター分

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
			view = prefixLines(view, bar+" ")
		} else {
			view = prefixLines(view, pad)
		}
		lines = append(lines, view)
	}
	content := strings.Join(lines, "\n\n")

	var b strings.Builder
	b.WriteString(applyScroll(content, m.scrollOffset, m.height-formFooterLines))
	b.WriteString("\n\n")
	if m.errMsg != "" {
		b.WriteString(errorStyle.Render("  "+m.errMsg) + "\n")
	}
	b.WriteString(formFooterStyle.Render("tab 次へ · shift+tab 戻る · enter 確定/詳細 · ctrl+c 終了"))

	return b.String()
}

// visibleFields は現在表示可能なフィールドのみを返す
func (m formModel) visibleFields() []Field {
	result := make([]Field, 0, len(m.fields))
	for _, f := range m.fields {
		if f.Visible() {
			result = append(result, f)
		}
	}
	return result
}

// fieldIndexFromVisibleIndex はvisibleインデックスからm.fieldsの絶対インデックスを返す
func (m formModel) fieldIndexFromVisibleIndex(visibleIdx int) int {
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

// ---------- スケジュールフィールドのメソッド ----------

// scheduleOnDrillDownCmd はスケジュール選択でEnter押下時のドリルダウンコマンドを返す
func (m *formModel) scheduleOnDrillDownCmd() tea.Cmd {
	switch m.state.scheduleType {
	case string(ScheduleInterval):
		return func() tea.Msg { return openDetailMsg{kind: detailInterval} }
	case string(ScheduleCalendar):
		return func() tea.Msg { return openDetailMsg{kind: detailCalendar} }
	}
	return nil
}

// scheduleValidate はスケジュール選択のバリデーションを行う
func (m *formModel) scheduleValidate(value string) string {
	if (value == string(ScheduleInterval) || value == string(ScheduleCalendar)) && scheduleOptionDetail(m.state, value) == "" {
		return "enter で詳細を入力してください"
	}
	return ""
}

// scheduleDetail はスケジュールオプションの詳細テキストを返す
func (m *formModel) scheduleDetail(value string) string {
	if d := scheduleOptionDetail(m.state, value); d != "" {
		return d
	}
	if (value == string(ScheduleInterval) || value == string(ScheduleCalendar)) && value == m.state.scheduleType {
		return "enter で編集"
	}
	return ""
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

	// 未選択（空文字列）の場合はデフォルト値を使用
	if s.processType == "" {
		s.processType = string(ProcessStandard)
	}
	if s.keepAlive == "" {
		s.keepAlive = string(KeepAliveNone)
	}
	if s.scheduleType == "" {
		s.scheduleType = string(ScheduleNone)
	}

	if !validProcessType(s.processType) {
		return fmt.Errorf("不正なProcessType: %q", s.processType)
	}
	if !validKeepAliveType(s.keepAlive) {
		return fmt.Errorf("不正なKeepAliveType: %q", s.keepAlive)
	}
	if !validScheduleType(s.scheduleType) {
		return fmt.Errorf("不正なScheduleType: %q", s.scheduleType)
	}
	c.ProcessType = ProcessType(s.processType)
	c.KeepAlive = KeepAliveType(s.keepAlive)
	c.ScheduleType = ScheduleType(s.scheduleType)

	if err := applyScheduleValues(c, s); err != nil {
		return err
	}

	envVars, err := parseEnvVars(s.envVarsStr)
	if err != nil {
		return err
	}
	c.EnvironmentVars = envVars

	return applyLogPaths(c, s)
}

// applyScheduleValues はスケジュール関連の値をConfigに適用する
func applyScheduleValues(c *Config, s *formState) error {
	if s.scheduleType == string(ScheduleInterval) {
		if s.intervalStr != "" {
			n, err := strconv.Atoi(s.intervalStr)
			if err != nil {
				return fmt.Errorf("StartIntervalの値が不正です: %q", s.intervalStr)
			}
			if n <= 0 {
				return fmt.Errorf("StartIntervalは正の整数を指定してください: %d", n)
			}
			c.StartInterval = n
		} else {
			c.ScheduleType = ScheduleNone
		}
	}

	if s.scheduleType == string(ScheduleCalendar) {
		ci := CalendarInterval{
			Minute:  parseOptionalInt(s.minuteStr),
			Hour:    parseOptionalInt(s.hourStr),
			Day:     parseOptionalInt(s.dayStr),
			Month:   parseOptionalInt(s.monthStr),
			Weekday: parseOptionalInt(s.weekdayStr),
		}
		if err := validateCalendarRange(ci); err != nil {
			return err
		}
		if ci.HasValue() {
			c.Calendar = ci
		} else {
			c.ScheduleType = ScheduleNone
		}
	}
	return nil
}

// validateCalendarRange はCalendarIntervalの各フィールドが有効な範囲内かを検証する
func validateCalendarRange(ci CalendarInterval) error {
	checks := []struct {
		name string
		val  *int
		min  int
		max  int
	}{
		{"月", ci.Month, 1, 12},
		{"日", ci.Day, 1, 31},
		{"曜日", ci.Weekday, 0, 6},
		{"時", ci.Hour, 0, 23},
		{"分", ci.Minute, 0, 59},
	}
	for _, c := range checks {
		if c.val != nil && (*c.val < c.min || *c.val > c.max) {
			return fmt.Errorf("%sの値が範囲外です: %d（%d〜%d）", c.name, *c.val, c.min, c.max)
		}
	}
	return nil
}

// applyLogPaths はログパスをConfigに適用する（~展開・絶対パス化）
func applyLogPaths(c *Config, s *formState) error {
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
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("絶対パスの解決に失敗: %w", err)
	}
	return abs, nil
}

// parseEnvVars は "KEY=VALUE\nKEY2=VALUE2" 形式の文字列をmapに変換する。
// "=" を含まない行がある場合はエラーを返す。
func parseEnvVars(s string) (map[string]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	result := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("環境変数の書式が不正です（KEY=VALUE形式で入力してください）: %q", line)
		}
		result[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func validProcessType(s string) bool {
	switch ProcessType(s) {
	case ProcessStandard, ProcessBackground, ProcessInteractive:
		return true
	}
	return false
}

func validKeepAliveType(s string) bool {
	switch KeepAliveType(s) {
	case KeepAliveNone, KeepAliveAlways, KeepAliveOnFailure:
		return true
	}
	return false
}

func validScheduleType(s string) bool {
	switch ScheduleType(s) {
	case ScheduleNone, ScheduleInterval, ScheduleCalendar:
		return true
	}
	return false
}

// applyScroll はテキストにスクロールオフセットを適用し、表示範囲の行のみを返す。
// usableHeight が1未満の場合はスクロールせずそのまま返す。
func applyScroll(content string, offset, usableHeight int) string {
	lines := strings.Split(content, "\n")
	if usableHeight < 1 {
		return content
	}
	start := min(offset, len(lines))
	end := min(start+usableHeight, len(lines))
	return strings.Join(lines[start:end], "\n")
}

// prefixLines はテキストの各行にプレフィックスを付加する
func prefixLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
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
