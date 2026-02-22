package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Field はフォーム内の入力フィールドを抽象化するインターフェース
type Field interface {
	Update(msg tea.Msg) (Field, tea.Cmd)
	View() string
	Focus() tea.Cmd
	Blur()
	Visible() bool
	Validate() string // エラーメッセージ（空文字ならOK）
	Height() int
}

// ---------- TextInputField ----------

// TextInputField は単一行テキスト入力フィールド
type TextInputField struct {
	title       string
	description string
	model       textinput.Model
	bound       *string // バインド先のポインタ（値の自動同期用）
	required    bool
	focused     bool
	visibleFn   func() bool
	defaultFn  func() string // 動的デフォルト値（nilなら固定）
	validateFn func(string) string
}

// TextInputOption はTextInputFieldの設定関数
type TextInputOption func(*TextInputField)

func WithRequired() TextInputOption {
	return func(f *TextInputField) { f.required = true }
}

func WithDefaultFunc(fn func() string) TextInputOption {
	return func(f *TextInputField) { f.defaultFn = fn }
}

func WithVisibleFunc(fn func() bool) TextInputOption {
	return func(f *TextInputField) { f.visibleFn = fn }
}

func WithValidateFunc(fn func(string) string) TextInputOption {
	return func(f *TextInputField) { f.validateFn = fn }
}

func NewTextInputField(title, description string, value *string, opts ...TextInputOption) *TextInputField {
	ti := textinput.New()
	ti.Prompt = "  "
	ti.SetValue(*value)
	ti.Cursor.Style = focusedCursorStyle

	f := &TextInputField{
		title:       title,
		description: description,
		model:       ti,
		bound:       value,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *TextInputField) Value() string { return f.model.Value() }

func (f *TextInputField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	f.model, cmd = f.model.Update(msg)
	// バインド先へ値を同期
	if f.bound != nil {
		*f.bound = f.model.Value()
	}
	return f, cmd
}

func (f *TextInputField) View() string {
	if f.focused {
		return f.viewFocused()
	}
	return f.viewBlurred()
}

func (f *TextInputField) viewFocused() string {
	var b strings.Builder
	b.WriteString(focusedTitleStyle.Render(f.title))
	if f.description != "" {
		b.WriteString("  ")
		b.WriteString(focusedDescStyle.Render(f.description))
	}
	b.WriteString("\n")
	b.WriteString(f.model.View())
	if errMsg := f.Validate(); errMsg != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  " + errMsg))
	}
	return b.String()
}

func (f *TextInputField) viewBlurred() string {
	var b strings.Builder
	b.WriteString(blurredTitleStyle.Render(f.title))
	if f.description != "" {
		b.WriteString("  ")
		b.WriteString(blurredMutedStyle.Render(f.description))
	}
	b.WriteString("\n")
	val := f.model.Value()
	style := blurredValueStyle
	if val == "" && f.defaultFn != nil {
		val = f.defaultFn()
	}
	if val == "" {
		val = "-"
	}
	b.WriteString("  ")
	b.WriteString(style.Render(val))
	return b.String()
}

func (f *TextInputField) Focus() tea.Cmd {
	f.focused = true
	if f.defaultFn != nil && f.model.Value() == "" {
		if def := f.defaultFn(); def != "" {
			f.model.SetValue(def)
			if f.bound != nil {
				*f.bound = def
			}
		}
	}
	return f.model.Focus()
}

func (f *TextInputField) Blur() {
	f.focused = false
	f.model.Blur()
}

func (f *TextInputField) Visible() bool {
	if f.visibleFn != nil {
		return f.visibleFn()
	}
	return true
}

func (f *TextInputField) Validate() string {
	v := f.model.Value()
	if f.required && strings.TrimSpace(v) == "" {
		return "必須項目です"
	}
	if f.validateFn != nil {
		return f.validateFn(v)
	}
	return ""
}

func (f *TextInputField) Height() int {
	h := 2 // title + input
	if f.focused && f.Validate() != "" {
		h++
	}
	return h
}

// ---------- SelectField ----------

// SelectOption は選択肢の1つ
type SelectOption struct {
	Label string
	Value string
}

// SelectOption func はSelectFieldの設定関数
type SelectFieldOption func(*SelectField)

// WithOptionDetailFn はオプションごとの詳細文字列を返す関数を設定する。
// 非空文字列を返した場合、オプションラベルの横にカッコ付きで表示する。
func WithOptionDetailFn(fn func(value string) string) SelectFieldOption {
	return func(f *SelectField) { f.optionDetailFn = fn }
}

// WithSelectValidateFunc はSelectFieldのバリデーション関数を設定する。
// 選択中の値を受け取り、エラーメッセージを返す（空文字ならOK）。
func WithSelectValidateFunc(fn func(string) string) SelectFieldOption {
	return func(f *SelectField) { f.validateFn = fn }
}

// WithOnEnterFunc はEnter押下時に実行するコマンド生成関数を設定する。
// nilを返した場合はドリルダウン遷移しない。
func WithOnEnterFunc(fn func() tea.Cmd) SelectFieldOption {
	return func(f *SelectField) { f.onEnterFn = fn }
}

// SelectField は選択式フィールド
type SelectField struct {
	title          string
	options        []SelectOption
	cursor         int
	value          *string
	focused        bool
	optionDetailFn func(value string) string
	validateFn     func(string) string
	onEnterFn      func() tea.Cmd
}

func NewSelectField(title string, options []SelectOption, value *string, opts ...SelectFieldOption) *SelectField {
	cursor := 0
	matched := false
	for i, opt := range options {
		if opt.Value == *value {
			cursor = i
			matched = true
			break
		}
	}
	// 値がどのオプションにもマッチしない場合、先頭オプションの値に同期
	if !matched && len(options) > 0 {
		*value = options[0].Value
	}
	f := &SelectField{
		title:   title,
		options: options,
		cursor:  cursor,
		value:   value,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *SelectField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if f.cursor > 0 {
				f.cursor--
			}
		case "down", "j":
			if f.cursor < len(f.options)-1 {
				f.cursor++
			}
		}
		*f.value = f.options[f.cursor].Value
	}
	return f, nil
}

func (f *SelectField) View() string {
	if f.focused {
		return f.viewFocused()
	}
	return f.viewBlurred()
}

func (f *SelectField) viewFocused() string {
	var b strings.Builder
	b.WriteString(focusedTitleStyle.Render(f.title))
	b.WriteString("\n")
	for i, opt := range f.options {
		label := "  " + opt.Label
		if d := f.optionDetail(opt.Value); d != "" {
			label += "（" + d + "）"
		}
		if i == f.cursor {
			b.WriteString(selectedOptionStyle.Render(label))
		} else {
			b.WriteString(unselectedOptionStyle.Render(label))
		}
		if i < len(f.options)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (f *SelectField) viewBlurred() string {
	var b strings.Builder
	b.WriteString(blurredTitleStyle.Render(f.title))
	b.WriteString("\n")
	for i, opt := range f.options {
		label := "  " + opt.Label
		if d := f.optionDetail(opt.Value); d != "" {
			label += "（" + d + "）"
		}
		if opt.Value == *f.value {
			b.WriteString(blurredValueStyle.Render(label))
		} else {
			b.WriteString(blurredMutedStyle.Render(label))
		}
		if i < len(f.options)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (f *SelectField) optionDetail(value string) string {
	if f.optionDetailFn == nil {
		return ""
	}
	return f.optionDetailFn(value)
}

func (f *SelectField) Focus() tea.Cmd {
	f.focused = true
	// カーソルを現在の値に合わせる
	matched := false
	for i, opt := range f.options {
		if opt.Value == *f.value {
			f.cursor = i
			matched = true
			break
		}
	}
	// 値がどのオプションにもマッチしない場合、先頭オプションに同期
	if !matched && len(f.options) > 0 {
		f.cursor = 0
		*f.value = f.options[0].Value
	}
	return nil
}

func (f *SelectField) Blur() {
	f.focused = false
}

func (f *SelectField) Visible() bool { return true }

func (f *SelectField) Validate() string {
	if f.validateFn != nil {
		return f.validateFn(*f.value)
	}
	return ""
}

func (f *SelectField) Height() int {
	return 1 + len(f.options) // title + options
}

func (f *SelectField) SelectedValue() string { return *f.value }

// ---------- ConfirmField ----------

// ConfirmField はYes/Noトグルフィールド
type ConfirmField struct {
	title        string
	description  string
	value        *bool
	focused      bool
	trueLabel    string // カスタムラベル（空ならYes）
	falseLabel   string // カスタムラベル（空ならNo）
	reverseOrder bool   // trueならfalseLabel→trueLabelの順で表示
	buttonStyle  bool   // trueならボタン風表示
}

// ConfirmFieldOption はConfirmFieldの設定関数
type ConfirmFieldOption func(*ConfirmField)

// WithLabels はトグルのラベルをカスタマイズする。
func WithLabels(falseLabel, trueLabel string) ConfirmFieldOption {
	return func(f *ConfirmField) {
		f.falseLabel = falseLabel
		f.trueLabel = trueLabel
	}
}

// WithReverseOrder はラベルの表示順を falseLabel → trueLabel にする。
func WithReverseOrder() ConfirmFieldOption {
	return func(f *ConfirmField) { f.reverseOrder = true }
}

// WithButtonStyle はボタン風の枠付き表示にする。
func WithButtonStyle() ConfirmFieldOption {
	return func(f *ConfirmField) { f.buttonStyle = true }
}

func NewConfirmField(title, description string, value *bool, opts ...ConfirmFieldOption) *ConfirmField {
	f := &ConfirmField{
		title:       title,
		description: description,
		value:       value,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *ConfirmField) labels() (string, string) {
	yes, no := "Yes", "No"
	if f.trueLabel != "" {
		yes = f.trueLabel
	}
	if f.falseLabel != "" {
		no = f.falseLabel
	}
	return yes, no
}

func (f *ConfirmField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "left", "right", "h", "l":
			*f.value = !*f.value
		case "y":
			*f.value = true
		case "n":
			*f.value = false
		}
	}
	return f, nil
}

func (f *ConfirmField) View() string {
	if f.focused {
		return f.viewFocused()
	}
	return f.viewBlurred()
}

func (f *ConfirmField) renderView(focused bool) string {
	var b strings.Builder

	// スタイル選択
	var titleStyle, descStyle lipgloss.Style
	var activeStyle, inactiveStyle lipgloss.Style
	if focused {
		titleStyle, descStyle = focusedTitleStyle, focusedDescStyle
		if f.buttonStyle {
			activeStyle, inactiveStyle = activeButtonStyle, inactiveButtonStyle
		} else {
			activeStyle, inactiveStyle = activeToggleStyle, inactiveToggleStyle
		}
	} else {
		titleStyle, descStyle = blurredTitleStyle, blurredMutedStyle
		if f.buttonStyle {
			activeStyle, inactiveStyle = blurredActiveButtonStyle, blurredInactiveButtonStyle
		} else {
			activeStyle, inactiveStyle = blurredValueStyle, blurredMutedStyle
		}
	}

	// タイトル（ボタンスタイルでない場合のみ表示）
	if !f.buttonStyle {
		b.WriteString(titleStyle.Render(f.title))
		if f.description != "" {
			b.WriteString("  ")
			b.WriteString(descStyle.Render(f.description))
		}
		b.WriteString("\n")
	}

	// ラベル
	yes, no := f.labels()
	if *f.value {
		yes = activeStyle.Render(yes)
		no = inactiveStyle.Render(no)
	} else {
		yes = inactiveStyle.Render(yes)
		no = activeStyle.Render(no)
	}
	left, right := yes, no
	if f.reverseOrder {
		left, right = no, yes
	}
	if f.buttonStyle {
		fmt.Fprintf(&b, "%s  %s", left, right)
	} else {
		fmt.Fprintf(&b, "  %s  %s", left, right)
	}
	return b.String()
}

func (f *ConfirmField) viewFocused() string { return f.renderView(true) }
func (f *ConfirmField) viewBlurred() string { return f.renderView(false) }

func (f *ConfirmField) Focus() tea.Cmd {
	f.focused = true
	return nil
}

func (f *ConfirmField) Blur() {
	f.focused = false
}

func (f *ConfirmField) Visible() bool { return true }

func (f *ConfirmField) Validate() string { return "" }

func (f *ConfirmField) Height() int {
	if f.buttonStyle {
		return 1 // buttons only
	}
	return 2 // title + toggle
}

// ---------- DrillDownField ----------

// DrillDownField は現在の値を表示し、Enterでドリルダウンに遷移する読み取り専用フィールド
type DrillDownField struct {
	title     string
	kind      detailKind    // ドリルダウン画面の種類
	summaryFn func() string // 現在値のサマリーを返す
	focused   bool
}

func NewDrillDownField(title string, kind detailKind, summaryFn func() string) *DrillDownField {
	return &DrillDownField{title: title, kind: kind, summaryFn: summaryFn}
}

func (f *DrillDownField) Update(tea.Msg) (Field, tea.Cmd) { return f, nil }

func (f *DrillDownField) View() string {
	summary := f.summaryFn()

	var b strings.Builder
	if f.focused {
		b.WriteString(focusedTitleStyle.Render(f.title))
		b.WriteString("  ")
		b.WriteString(focusedDescStyle.Render("(enter で編集)"))
		if summary == "" {
			b.WriteString("\n")
			b.WriteString(selectedOptionStyle.Render("  -"))
		} else {
			for _, line := range strings.Split(summary, "\n") {
				b.WriteString("\n")
				b.WriteString(selectedOptionStyle.Render("  " + line))
			}
		}
	} else {
		b.WriteString(blurredTitleStyle.Render(f.title))
		b.WriteString("  ")
		b.WriteString(blurredMutedStyle.Render("(enter で編集)"))
		if summary == "" {
			b.WriteString("\n")
			b.WriteString(blurredValueStyle.Render("  -"))
		} else {
			for _, line := range strings.Split(summary, "\n") {
				b.WriteString("\n")
				b.WriteString(blurredValueStyle.Render("  " + line))
			}
		}
	}
	return b.String()
}

func (f *DrillDownField) Focus() tea.Cmd { f.focused = true; return nil }
func (f *DrillDownField) Blur()          { f.focused = false }
func (f *DrillDownField) Visible() bool { return true }
func (f *DrillDownField) Validate() string { return "" }

func (f *DrillDownField) Height() int {
	summary := f.summaryFn()
	if summary == "" {
		return 2 // title(+hint) + "-"
	}
	lines := strings.Split(summary, "\n")
	return 1 + len(lines) // title(+hint) + content lines
}
