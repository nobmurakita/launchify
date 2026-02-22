package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFormModel_BuildFields(t *testing.T) {
	c := &Config{RunAtLoad: true}
	s := &formState{}
	m := newFormModel(c, s)

	if len(m.fields) != 11 {
		t.Errorf("fields count = %d, want 11", len(m.fields))
	}
}

func TestFormModel_LogPathDrillDownFields(t *testing.T) {
	c := &Config{}
	s := &formState{}
	m := newFormModel(c, s)

	found := map[detailKind]bool{}
	for _, f := range m.fields {
		if df, ok := f.(*DrillDownField); ok {
			found[df.kind] = true
		}
	}
	if !found[detailStdoutPath] {
		t.Error("detailStdoutPath DrillDownField should exist")
	}
	if !found[detailStderrPath] {
		t.Error("detailStderrPath DrillDownField should exist")
	}
}

func TestFormModel_FocusNavigation(t *testing.T) {
	c := &Config{}
	s := &formState{}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40
	m.Init()

	if m.focused != 0 {
		t.Errorf("initial focused = %d, want 0", m.focused)
	}

	// Tab で次へ（Labelが空なのでバリデーションエラーで進めない）
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focused != 0 {
		t.Errorf("focused should stay 0 due to validation, got %d", m.focused)
	}
}

func TestFormModel_ShiftTabAtFirst(t *testing.T) {
	c := &Config{}
	s := &formState{}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40
	m.Init()

	// shift+tab で0未満にならないことを確認
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.focused != 0 {
		t.Errorf("focused = %d, should not go below 0", m.focused)
	}
}

func TestFormModel_PointerBindingSync(t *testing.T) {
	c := &Config{}
	s := &formState{}
	m := newFormModel(c, s)

	// Program フィールド（index 1）に値を入力
	m.fields[1].Focus()
	m.fields[1], _ = m.fields[1].Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("myapp")})

	// CollectValues不要 — ポインタバインディングで自動同期
	if c.Program != "myapp" {
		t.Errorf("Program = %q, want %q (should auto-sync via pointer binding)", c.Program, "myapp")
	}
}

func TestFormModel_ViewContainsFooter(t *testing.T) {
	c := &Config{}
	s := &formState{}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40
	m.Init()

	view := m.View()
	if !strings.Contains(view, "tab 次へ") {
		t.Error("View() should contain footer with navigation hints")
	}
}

func TestFormModel_SubmitOnLastField(t *testing.T) {
	c := &Config{Program: "ls"}
	s := &formState{}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40

	// アクションボタン（最後のvisibleフィールド）にフォーカスを移動
	visible := m.visibleFields()
	m.focusVisibleFieldAt(len(visible) - 1)

	// Enter でformSubmitMsgが発行される（ボタンの確定はEnterのみ）
	var resultCmd tea.Cmd
	m, resultCmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if resultCmd == nil {
		t.Fatal("expected a cmd from submit")
	}
	msg := resultCmd()
	if _, ok := msg.(formSubmitMsg); !ok {
		t.Errorf("expected formSubmitMsg, got %T", msg)
	}
}

func TestFormModel_DrillDownOnScheduleInterval(t *testing.T) {
	c := &Config{}
	s := &formState{scheduleType: "interval"}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40

	// onDrillDownFnが設定されたSelectFieldを見つけてフォーカス
	visible := m.visibleFields()
	schedIdx := -1
	for i, f := range visible {
		if sf, ok := f.(*SelectField); ok && sf.onDrillDownFn != nil {
			schedIdx = i
			break
		}
	}
	if schedIdx < 0 {
		t.Fatal("onDrillDownFn付きSelectFieldが見つからない")
	}

	m.focusVisibleFieldAt(schedIdx)

	// interval が選択されている状態でEnter → openDetailMsg
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected openDetailMsg cmd")
	}
	msg := cmd()
	dm, ok := msg.(openDetailMsg)
	if !ok {
		t.Fatalf("expected openDetailMsg, got %T", msg)
	}
	if dm.kind != detailInterval {
		t.Errorf("kind = %v, want detailInterval", dm.kind)
	}
}

func TestFormModel_ErrMsgDisplayedOnSubmitError(t *testing.T) {
	c := &Config{Program: "ls"}
	s := &formState{}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40

	// エラーメッセージを直接設定してView()に表示されることを確認
	m.errMsg = "テストエラー"
	view := m.View()
	if !strings.Contains(view, "テストエラー") {
		t.Error("View() should display errMsg when set")
	}
}

func TestFormModel_ErrMsgClearedOnKeyPress(t *testing.T) {
	c := &Config{Program: "ls"}
	s := &formState{}
	m := newFormModel(c, s)
	m.width = 80
	m.height = 40
	m.Init()

	m.errMsg = "何かのエラー"

	// キー操作でエラーメッセージがクリアされる
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.errMsg != "" {
		t.Errorf("errMsg should be cleared after key press, got %q", m.errMsg)
	}
}

func TestScheduleOptionDetail(t *testing.T) {
	tests := []struct {
		name        string
		state       formState
		optionValue string
		want        string
	}{
		{"none", formState{}, "none", ""},
		{"interval with value", formState{intervalStr: "300"}, "interval", "300秒"},
		{"interval empty", formState{}, "interval", ""},
		{"calendar hour+minute", formState{hourStr: "9", minuteStr: "30"}, "calendar", "時=9 分=30"},
		{"calendar all empty", formState{}, "calendar", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scheduleOptionDetail(&tt.state, tt.optionValue)
			if got != tt.want {
				t.Errorf("scheduleOptionDetail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormModel_SelectFieldInitialValues(t *testing.T) {
	// formState の各フィールドが空文字列でも、newFormModel後に先頭オプション値が設定される
	c := &Config{}
	s := &formState{}
	newFormModel(c, s)

	if s.scheduleType == "" {
		t.Error("scheduleType should be initialized to first option value")
	}
	if s.processType == "" {
		t.Error("processType should be initialized to first option value")
	}
	if s.keepAlive == "" {
		t.Error("keepAlive should be initialized to first option value")
	}
}

func TestParseEnvVars(t *testing.T) {
	got := parseEnvVars("PATH=/usr/local/bin\nHOME=/Users/test")
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got["PATH"] != "/usr/local/bin" {
		t.Errorf("PATH: got %q, want /usr/local/bin", got["PATH"])
	}
	if got["HOME"] != "/Users/test" {
		t.Errorf("HOME: got %q, want /Users/test", got["HOME"])
	}
}

func TestParseEnvVars_Empty(t *testing.T) {
	got := parseEnvVars("")
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestParseEnvVars_WithSpaces(t *testing.T) {
	got := parseEnvVars("  KEY = VALUE  \n\n  KEY2=VALUE2  \n")
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got["KEY"] != "VALUE" {
		t.Errorf("KEY: got %q, want VALUE", got["KEY"])
	}
}

func TestApplyFormValues_Basic(t *testing.T) {
	c := &Config{
		Label:   "com.test.app",
		Program: "/usr/bin/test",
	}
	s := &formState{
		processType:  "Background",
		keepAlive:    "always",
		scheduleType: "none",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.ProcessType != ProcessBackground {
		t.Errorf("ProcessType = %q, want %q", c.ProcessType, ProcessBackground)
	}
	if c.KeepAlive != KeepAliveAlways {
		t.Errorf("KeepAlive = %q, want %q", c.KeepAlive, KeepAliveAlways)
	}
	if c.ScheduleType != ScheduleNone {
		t.Errorf("ScheduleType = %q, want %q", c.ScheduleType, ScheduleNone)
	}
}

func TestApplyFormValues_IntervalWithValue(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "interval",
		intervalStr:  "300",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.ScheduleType != ScheduleInterval {
		t.Errorf("ScheduleType = %q, want %q", c.ScheduleType, ScheduleInterval)
	}
	if c.StartInterval != 300 {
		t.Errorf("StartInterval = %d, want 300", c.StartInterval)
	}
}

func TestApplyFormValues_IntervalEmpty_FallsBackToNone(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "interval",
		intervalStr:  "",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.ScheduleType != ScheduleNone {
		t.Errorf("ScheduleType = %q, want %q (empty interval should fallback)", c.ScheduleType, ScheduleNone)
	}
}

func TestApplyFormValues_CalendarWithValues(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "calendar",
		hourStr:      "9",
		minuteStr:    "30",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.ScheduleType != ScheduleCalendar {
		t.Errorf("ScheduleType = %q, want %q", c.ScheduleType, ScheduleCalendar)
	}
	if c.Calendar.Hour == nil || *c.Calendar.Hour != 9 {
		t.Errorf("Calendar.Hour = %v, want 9", c.Calendar.Hour)
	}
	if c.Calendar.Minute == nil || *c.Calendar.Minute != 30 {
		t.Errorf("Calendar.Minute = %v, want 30", c.Calendar.Minute)
	}
}

func TestApplyFormValues_CalendarEmpty_FallsBackToNone(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "calendar",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.ScheduleType != ScheduleNone {
		t.Errorf("ScheduleType = %q, want %q (empty calendar should fallback)", c.ScheduleType, ScheduleNone)
	}
}

func TestApplyFormValues_ProgramDefaultFromLabel(t *testing.T) {
	c := &Config{Label: "com.user.mytool", Program: ""}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "none",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.Program != "mytool" {
		t.Errorf("Program = %q, want %q", c.Program, "mytool")
	}
}

func TestApplyFormValues_EnvVars(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "none",
		envVarsStr:   "PATH=/usr/bin\nHOME=/Users/test",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.EnvironmentVars) != 2 {
		t.Fatalf("EnvironmentVars count = %d, want 2", len(c.EnvironmentVars))
	}
	if c.EnvironmentVars["PATH"] != "/usr/bin" {
		t.Errorf("PATH = %q", c.EnvironmentVars["PATH"])
	}
}

func TestApplyFormValues_LogPaths(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "none",
		stdoutPath:   "/tmp/stdout.log",
		stderrPath:   "/tmp/stderr.log",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if c.StdoutPath != "/tmp/stdout.log" {
		t.Errorf("StdoutPath = %q, want /tmp/stdout.log", c.StdoutPath)
	}
	if c.StderrPath != "/tmp/stderr.log" {
		t.Errorf("StderrPath = %q, want /tmp/stderr.log", c.StderrPath)
	}
}

func TestApplyFormValues_TildeExpansion(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "none",
		stdoutPath:   "~/logs/test.log",
	}
	err := applyFormValues(c, s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(c.StdoutPath, "~/") {
		t.Errorf("StdoutPath should expand tilde, got %q", c.StdoutPath)
	}
	if !strings.HasSuffix(c.StdoutPath, "/logs/test.log") {
		t.Errorf("StdoutPath should end with /logs/test.log, got %q", c.StdoutPath)
	}
}

func TestApplyFormValues_InvalidIntervalStr(t *testing.T) {
	c := &Config{Label: "test", Program: "/bin/test"}
	s := &formState{
		processType:  "Standard",
		keepAlive:    "none",
		scheduleType: "interval",
		intervalStr:  "abc",
	}
	err := applyFormValues(c, s)
	if err == nil {
		t.Fatal("expected error for non-numeric intervalStr")
	}
	if !strings.Contains(err.Error(), "StartInterval") {
		t.Errorf("error should mention StartInterval, got: %v", err)
	}
}
