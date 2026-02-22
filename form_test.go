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

	// StandardOutPath と StandardErrorPath の DrillDownField が存在する
	found := map[string]bool{}
	for _, f := range m.fields {
		if df, ok := f.(*DrillDownField); ok {
			found[df.title] = true
		}
	}
	if !found["StandardOutPath"] {
		t.Error("StandardOutPath DrillDownField should exist")
	}
	if !found["StandardErrorPath"] {
		t.Error("StandardErrorPath DrillDownField should exist")
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

func TestFormModel_CollectValues(t *testing.T) {
	c := &Config{}
	s := &formState{}
	m := newFormModel(c, s)

	// Program フィールド（index 1）に値を入力
	m.fields[1].Focus()
	m.fields[1], _ = m.fields[1].Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("myapp")})

	m.CollectValues()
	if c.Program != "myapp" {
		t.Errorf("Program = %q, want %q", c.Program, "myapp")
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
	m.FocusField(len(visible) - 1)

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

	// スケジュールフィールドを見つけてフォーカス
	visible := m.visibleFields()
	schedIdx := -1
	for i, f := range visible {
		if sf, ok := f.(*SelectField); ok && sf.title == "スケジュール" {
			schedIdx = i
			break
		}
	}
	if schedIdx < 0 {
		t.Fatal("スケジュールフィールドが見つからない")
	}

	m.FocusField(schedIdx)

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
