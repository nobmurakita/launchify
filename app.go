package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// appPhase はアプリケーションの画面状態を表す
type appPhase int

const (
	phaseForm    appPhase = iota // メインフォーム
	phaseDetail                  // ドリルダウン画面
	phasePreview                 // プレビュー画面
)

// appResult はアプリケーションの終了結果を表す
type appResult int

const (
	appResultNone    appResult = iota
	appResultInstall           // インストール実行
	appResultQuit              // 終了
)

// appModel はアプリケーション全体のモデル。
// フォーム、ドリルダウン、プレビューの3つの子モデルを管理する。
type appModel struct {
	phase   appPhase
	config  *Config
	state   *formState
	form    formModel
	detail  detailModel
	preview previewModel
	result  appResult
	plist   string // 生成されたplist XML
	width   int
	height  int
}

func newAppModel() appModel {
	c := &Config{RunAtLoad: true}
	s := &formState{}
	return appModel{
		phase:  phaseForm,
		config: c,
		state:  s,
		form:   newFormModel(c, s),
	}
}

func (m appModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// WindowSizeMsgは全子モデルに配布
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	switch m.phase {
	case phaseForm:
		return m.updateForm(msg)
	case phaseDetail:
		return m.updateDetail(msg)
	case phasePreview:
		return m.updatePreview(msg)
	}
	return m, nil
}

func (m appModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	// フォームからのメッセージをハンドリング
	switch msg := msg.(type) {
	case formSubmitMsg:
		return m.submitForm()
	case openDetailMsg:
		return m.openDetail(msg)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m appModel) submitForm() (tea.Model, tea.Cmd) {
	m.form.errMsg = "" // 前回のエラーをクリア

	// フィールドの値を収集
	m.form.CollectValues()
	applyFormValues(m.config, m.state)

	// プログラムのパス解決
	resolved, err := resolveProgram(m.config.Program)
	if err != nil {
		m.form.errMsg = err.Error()
		return m, nil
	}
	m.config.Program = resolved

	// plist生成
	plist, err := GeneratePlist(m.config)
	if err != nil {
		m.form.errMsg = fmt.Sprintf("plist生成エラー: %v", err)
		return m, nil
	}
	m.plist = plist

	plistPath, err := PlistPath(m.config.Label)
	if err != nil {
		m.form.errMsg = fmt.Sprintf("plistパス解決エラー: %v", err)
		return m, nil
	}

	if m.state.directInstall {
		// インストール直接実行
		m.result = appResultInstall
		return m, tea.Quit
	}

	// プレビュー画面へ遷移
	m.phase = phasePreview
	m.preview = newPreviewModel(plist, plistPath)
	// WindowSizeMsgを送信してviewportを初期化
	return m, func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m appModel) openDetail(msg openDetailMsg) (tea.Model, tea.Cmd) {
	m.phase = phaseDetail
	m.detail = newDetailModel(msg.kind, m.config, m.state, m.width, m.height)
	cmd := m.detail.Init()
	return m, cmd
}

func (m appModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(detailDoneMsg); ok {
		m.phase = phaseForm
		// フォームのフィールドを再構築して値を反映
		m.form.fields = m.form.buildFields()
		// 元のフィールドに戻る
		cmd := m.form.FocusField(m.form.focused)
		return m, cmd
	}

	var cmd tea.Cmd
	m.detail, cmd = m.detail.Update(msg)
	return m, cmd
}

func (m appModel) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	if pm, ok := msg.(previewDoneMsg); ok {
		switch pm.result {
		case resultInstall:
			m.result = appResultInstall
			return m, tea.Quit
		case resultBack:
			m.phase = phaseForm
			// フォームのフィールドを再構築（プレビューから戻った時に値を反映）
			m.form.fields = m.form.buildFields()
			cmd := m.form.FocusField(m.form.focused)
			return m, cmd
		case resultQuit:
			m.result = appResultQuit
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m appModel) View() string {
	switch m.phase {
	case phaseForm:
		return m.form.View()
	case phaseDetail:
		return m.detail.View()
	case phasePreview:
		return m.preview.View()
	}
	return ""
}

// RunApp はTUIアプリケーションを実行し、結果を返す
func RunApp() (*Config, string, error) {
	m := newAppModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, "", err
	}

	app := finalModel.(appModel)
	switch app.result {
	case appResultInstall:
		return app.config, app.plist, nil
	default:
		return nil, "", nil
	}
}

// resolveProgram はコマンド文字列の先頭プログラム名をフルパスに解決する。
// 既にフルパス（/で始まる）場合はそのまま返す。
func resolveProgram(program string) (string, error) {
	parts := strings.Fields(program)
	if len(parts) == 0 {
		return program, nil
	}
	cmd := parts[0]
	if strings.HasPrefix(cmd, "/") {
		return program, nil
	}
	resolved, err := exec.LookPath(cmd)
	if err != nil {
		return "", fmt.Errorf("コマンド %q が見つかりません: %w", cmd, err)
	}
	parts[0] = resolved
	return strings.Join(parts, " "), nil
}
