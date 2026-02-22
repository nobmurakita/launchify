package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------- TextInputField ----------

func TestTextInputField_FocusAndBlur(t *testing.T) {
	val := ""
	f := NewTextInputField("Program", "実行するコマンド", &val)

	f.Focus()
	if !f.focused {
		t.Error("Focus() should set focused=true")
	}

	f.Blur()
	if f.focused {
		t.Error("Blur() should set focused=false")
	}
}

func TestTextInputField_RequiredValidation(t *testing.T) {
	val := ""
	f := NewTextInputField("Program", "", &val, WithRequired())
	f.Focus()

	if msg := f.Validate(); msg == "" {
		t.Error("Validate() should return error for empty required field")
	}

	// 値を入力
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ls")})
	if msg := f.Validate(); msg != "" {
		t.Errorf("Validate() should pass after input, got: %s", msg)
	}
}

func TestTextInputField_CustomValidation(t *testing.T) {
	val := ""
	f := NewTextInputField("Port", "", &val, WithValidateFunc(func(v string) string {
		if v != "" && v != "8080" {
			return "ポートは8080のみ"
		}
		return ""
	}))
	f.Focus()

	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9999")})
	if msg := f.Validate(); msg == "" {
		t.Error("Validate() should return error for invalid port")
	}
}

func TestTextInputField_ViewFocused(t *testing.T) {
	val := ""
	f := NewTextInputField("Program", "実行するコマンド", &val)
	f.Focus()

	view := f.View()
	if !strings.Contains(view, "Program") {
		t.Error("focused View() should contain title")
	}
}

func TestTextInputField_ViewBlurred(t *testing.T) {
	val := "myapp"
	f := NewTextInputField("Program", "", &val)
	f.Blur()

	view := f.View()
	if !strings.Contains(view, "Program") {
		t.Error("blurred View() should contain title")
	}
}

func TestTextInputField_Visible(t *testing.T) {
	val := ""
	f := NewTextInputField("Test", "", &val)
	if !f.Visible() {
		t.Error("default Visible() should be true")
	}

	show := false
	f2 := NewTextInputField("Test", "", &val, WithVisibleFunc(func() bool { return show }))
	if f2.Visible() {
		t.Error("Visible() should be false when visibleFn returns false")
	}
	show = true
	if !f2.Visible() {
		t.Error("Visible() should be true when visibleFn returns true")
	}
}

func TestTextInputField_DefaultFunc(t *testing.T) {
	val := ""
	label := "com.user.mytool"
	f := NewTextInputField("Program", "", &val, WithDefaultFunc(func() string {
		parts := strings.Split(label, ".")
		return parts[len(parts)-1]
	}))

	// Focus時に値が空ならデフォルト値が設定される
	f.Focus()
	if f.model.Value() != "mytool" {
		t.Errorf("Value = %q, want %q", f.model.Value(), "mytool")
	}
	if val != "mytool" {
		t.Errorf("bound value = %q, want %q", val, "mytool")
	}
}

// ---------- SelectField ----------

func TestSelectField_InitialValueSync(t *testing.T) {
	// 空文字列（どのオプションにもマッチしない）の場合、先頭オプションの値が設定される
	val := ""
	f := NewSelectField("スケジュール", []SelectOption{
		{"なし", "none"},
		{"Interval", "interval"},
		{"Calendar", "calendar"},
	}, &val)

	if val != "none" {
		t.Errorf("initial value = %q, want %q (first option)", val, "none")
	}
	if f.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", f.cursor)
	}
}

func TestSelectField_FocusValueSync(t *testing.T) {
	// Focus時にも値がオプションにマッチしなければ先頭に同期
	val := "invalid"
	f := NewSelectField("Test", []SelectOption{
		{"A", "a"},
		{"B", "b"},
	}, &val)

	// NewSelectFieldで既に同期されるが、外部で値が変わった場合も想定
	val = "unknown"
	f.Focus()

	if val != "a" {
		t.Errorf("value after Focus() = %q, want %q", val, "a")
	}
	if f.cursor != 0 {
		t.Errorf("cursor after Focus() = %d, want 0", f.cursor)
	}
}

func TestSelectField_ViewBlurredWithSyncedValue(t *testing.T) {
	// 初期値が空でも、New後は先頭オプションに同期されるため正しいラベルが表示される
	val := ""
	f := NewSelectField("KeepAlive", []SelectOption{
		{"しない", "none"},
		{"常に再起動", "always"},
	}, &val)
	f.Blur()

	view := f.View()
	if !strings.Contains(view, "しない") {
		t.Errorf("blurred View() should show first option label, got: %s", view)
	}
}

func TestSelectField_Navigation(t *testing.T) {
	val := "none"
	f := NewSelectField("スケジュール", []SelectOption{
		{"なし", "none"},
		{"Interval", "interval"},
		{"Calendar", "calendar"},
	}, &val)
	f.Focus()

	// 下に移動
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if val != "interval" {
		t.Errorf("value = %q after down, want %q", val, "interval")
	}

	// 上に移動
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if val != "none" {
		t.Errorf("value = %q after up, want %q", val, "none")
	}

	// 上限を超えない
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if val != "none" {
		t.Errorf("value = %q, should not go past first option", val)
	}
}

func TestSelectField_ViewFocused(t *testing.T) {
	val := "none"
	f := NewSelectField("スケジュール", []SelectOption{
		{"なし", "none"},
		{"Interval", "interval"},
	}, &val)
	f.Focus()

	view := f.View()
	if !strings.Contains(view, "なし") {
		t.Error("focused View() should contain option labels")
	}
}

func TestSelectField_ViewBlurred(t *testing.T) {
	val := "interval"
	f := NewSelectField("スケジュール", []SelectOption{
		{"なし", "none"},
		{"Interval", "interval"},
	}, &val)
	f.Blur()

	view := f.View()
	if !strings.Contains(view, "Interval") {
		t.Error("blurred View() should show selected option label")
	}
}

func TestSelectField_Height(t *testing.T) {
	val := "none"
	f := NewSelectField("Test", []SelectOption{
		{"A", "a"},
		{"B", "b"},
		{"C", "c"},
	}, &val)

	f.Focus()
	if h := f.Height(); h != 4 { // title + 3 options
		t.Errorf("focused Height() = %d, want 4", h)
	}

	f.Blur()
	if h := f.Height(); h != 4 { // title + 3 options (always expanded)
		t.Errorf("blurred Height() = %d, want 4", h)
	}
}

// ---------- ConfirmField ----------

func TestConfirmField_Toggle(t *testing.T) {
	val := false
	f := NewConfirmField("RunAtLoad", "", &val)
	f.Focus()

	// yで Yes
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !val {
		t.Error("value should be true after 'y'")
	}

	// nで No
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if val {
		t.Error("value should be false after 'n'")
	}

	// left/right でトグル
	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if !val {
		t.Error("value should toggle with right key")
	}
}

func TestConfirmField_ViewFocused(t *testing.T) {
	val := true
	f := NewConfirmField("RunAtLoad", "ロード時に実行", &val)
	f.Focus()

	view := f.View()
	if !strings.Contains(view, "Yes") || !strings.Contains(view, "No") {
		t.Error("focused View() should contain Yes and No")
	}
}

func TestConfirmField_ViewBlurred(t *testing.T) {
	val := true
	f := NewConfirmField("RunAtLoad", "", &val)
	f.Blur()

	view := f.View()
	if !strings.Contains(view, "Yes") {
		t.Error("blurred View() should show current value")
	}
}

func TestConfirmField_Height(t *testing.T) {
	val := false
	f := NewConfirmField("Test", "", &val)

	f.Focus()
	if h := f.Height(); h != 2 {
		t.Errorf("focused Height() = %d, want 2", h)
	}

	f.Blur()
	if h := f.Height(); h != 2 { // title + toggle (always expanded)
		t.Errorf("blurred Height() = %d, want 2", h)
	}
}

// ---------- DrillDownField ----------

func TestDrillDownField_ViewEmpty(t *testing.T) {
	f := NewDrillDownField("環境変数", detailEnvVars, func() string { return "" })

	// Blurred: 空の場合「-」を表示
	view := f.View()
	if !strings.Contains(view, "-") {
		t.Errorf("blurred View() should show '-' when empty, got: %s", view)
	}
}

func TestDrillDownField_ViewWithValue(t *testing.T) {
	f := NewDrillDownField("環境変数", detailEnvVars, func() string { return "PATH, HOME" })

	view := f.View()
	if !strings.Contains(view, "PATH, HOME") {
		t.Errorf("blurred View() should show summary, got: %s", view)
	}
}

func TestDrillDownField_ViewFocused(t *testing.T) {
	f := NewDrillDownField("環境変数", detailEnvVars, func() string { return "PATH" })
	f.Focus()

	view := f.View()
	if !strings.Contains(view, "PATH") {
		t.Errorf("focused View() should show summary, got: %s", view)
	}
	if !strings.Contains(view, "(enter で編集)") {
		t.Errorf("focused View() should show edit hint in title line, got: %s", view)
	}
}

func TestDrillDownField_Height(t *testing.T) {
	// 空: title(+hint) + "なし" = 2
	f := NewDrillDownField("Test", detailEnvVars, func() string { return "" })
	if h := f.Height(); h != 2 {
		t.Errorf("empty Height() = %d, want 2", h)
	}

	// 1行: title(+hint) + 1 content line = 2
	f2 := NewDrillDownField("Test", detailEnvVars, func() string { return "PATH=/usr/bin" })
	if h := f2.Height(); h != 2 {
		t.Errorf("single-line Height() = %d, want 2", h)
	}

	// 複数行: title(+hint) + 2 content lines = 3
	f3 := NewDrillDownField("Test", detailEnvVars, func() string { return "PATH=/usr/bin\nHOME=/Users/me" })
	if h := f3.Height(); h != 3 {
		t.Errorf("multi-line Height() = %d, want 3", h)
	}
}
