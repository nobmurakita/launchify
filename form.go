package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// RunForm はTUIフォームを表示し、入力結果をConfigとして返す
func RunForm() (*Config, error) {
	c := &Config{
		RunAtLoad: true,
	}

	var processType string
	var keepAlive string
	var scheduleType string
	var logType string
	var intervalStr string
	var minuteStr, hourStr, dayStr, monthStr, weekdayStr string
	var enableEnvVars bool
	var envVarsStr string

	theme := formTheme()

	form := huh.NewForm(
		// グループ1: 基本設定
		huh.NewGroup(
			huh.NewInput().
				Title("Program").
				Description("実行するコマンド").
				Value(&c.Program).
				Validate(huh.ValidateNotEmpty()),

			huh.NewInput().
				Title("Label").
				Description("LaunchAgentの識別子（例: com.user.mytool）").
				PlaceholderFunc(func() string {
					if c.Program == "" {
						return ""
					}
					return filepath.Base(strings.Fields(c.Program)[0])
				}, &c.Program).
				Value(&c.Label),

			huh.NewInput().
				Title("WorkingDirectory").
				Description("作業ディレクトリ（省略可）").
				Value(&c.WorkingDirectory),

			huh.NewConfirm().
				Title("環境変数").
				Description("環境変数を設定する").
				Value(&enableEnvVars),
		),

		// グループ1.5: 環境変数入力（条件付き）
		huh.NewGroup(
			huh.NewText().
				Title("環境変数").
				Description("KEY=VALUE 形式で1行ずつ入力").
				Lines(8).
				Value(&envVarsStr),
		).WithHideFunc(func() bool {
			return !enableEnvVars
		}),

		// グループ2: スケジュール
		huh.NewGroup(
			huh.NewConfirm().
				Title("RunAtLoad").
				Description("ロード時に即実行する").
				Value(&c.RunAtLoad),

			huh.NewSelect[string]().
				Title("スケジュール").
				Options(
					huh.NewOption("なし", "none"),
					huh.NewOption("Interval（秒指定）", "interval"),
					huh.NewOption("Calendar（日時指定）", "calendar"),
				).
				Value(&scheduleType),
		),

		// グループ2.5: Interval入力（条件付き）
		huh.NewGroup(
			huh.NewInput().
				Title("StartInterval").
				Description("実行間隔（秒）。空欄でスキップ").
				Value(&intervalStr).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					n, err := strconv.Atoi(s)
					if err != nil || n <= 0 {
						return fmt.Errorf("正の整数を入力してください")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return scheduleType != "interval"
		}),

		// グループ2.6: Calendar入力（条件付き）
		huh.NewGroup(
			huh.NewInput().
				Title("分 (Minute)").
				Description("0-59（空欄=毎分）").
				Value(&minuteStr),

			huh.NewInput().
				Title("時 (Hour)").
				Description("0-23（空欄=毎時）").
				Value(&hourStr),

			huh.NewInput().
				Title("日 (Day)").
				Description("1-31（空欄=毎日）").
				Value(&dayStr),

			huh.NewInput().
				Title("月 (Month)").
				Description("1-12（空欄=毎月）").
				Value(&monthStr),

			huh.NewInput().
				Title("曜日 (Weekday)").
				Description("0=日, 1=月, ..., 6=土（空欄=毎日）").
				Value(&weekdayStr),
		).WithHideFunc(func() bool {
			return scheduleType != "calendar"
		}),

		// グループ3: プロセス設定
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("ProcessType").
				Description("プロセスの優先度").
				Options(
					huh.NewOption("Standard（デフォルト）", "Standard"),
					huh.NewOption("Background（リソース制限あり）", "Background"),
					huh.NewOption("Interactive（リソース制限なし）", "Interactive"),
				).
				Value(&processType),

			huh.NewSelect[string]().
				Title("KeepAlive").
				Options(
					huh.NewOption("しない", "none"),
					huh.NewOption("常に再起動", "always"),
					huh.NewOption("異常終了時のみ再起動", "on_failure"),
				).
				Value(&keepAlive),
		),

		// グループ4: ログ出力
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("ログファイル出力").
				Options(
					huh.NewOption("出力しない", "none"),
					huh.NewOption("stdout + stderr", "both"),
					huh.NewOption("stdoutのみ", "stdout"),
					huh.NewOption("stderrのみ", "stderr"),
				).
				Value(&logType),
		),

		// グループ4.5: stdout+stderrパス（条件付き）
		huh.NewGroup(
			huh.NewInput().
				Title("StandardOutPath").
				DescriptionFunc(func() string {
					return "stdout出力先（空欄の場合: ~/Library/Logs/" + resolveLabel(c) + ".log）"
				}, c).
				Value(&c.StdoutPath),

			huh.NewInput().
				Title("StandardErrorPath").
				Description("stderr出力先（空欄の場合: stdoutと同じ）").
				Value(&c.StderrPath),
		).WithHideFunc(func() bool {
			return logType != "both"
		}),

		// グループ4.6: stdoutのみパス（条件付き）
		huh.NewGroup(
			huh.NewInput().
				Title("StandardOutPath").
				DescriptionFunc(func() string {
					return "stdout出力先（空欄の場合: ~/Library/Logs/" + resolveLabel(c) + ".log）"
				}, c).
				Value(&c.StdoutPath),
		).WithHideFunc(func() bool {
			return logType != "stdout"
		}),

		// グループ4.7: stderrのみパス（条件付き）
		huh.NewGroup(
			huh.NewInput().
				Title("StandardErrorPath").
				DescriptionFunc(func() string {
					return "stderr出力先（空欄の場合: ~/Library/Logs/" + resolveLabel(c) + ".log）"
				}, c).
				Value(&c.StderrPath),
		).WithHideFunc(func() bool {
			return logType != "stderr"
		}),
	)

	if err := form.WithTheme(theme).Run(); err != nil {
		return nil, err
	}

	// Labelが空欄ならProgramのベース名を使用
	if c.Label == "" {
		fields := strings.Fields(c.Program)
		if len(fields) > 0 {
			c.Label = filepath.Base(fields[0])
		}
	}

	// ローカル変数をConfigに反映
	c.ProcessType = ProcessType(processType)
	c.KeepAlive = KeepAliveType(keepAlive)
	c.ScheduleType = ScheduleType(scheduleType)
	if scheduleType == "interval" && intervalStr != "" {
		n, _ := strconv.Atoi(intervalStr)
		c.StartInterval = n
	} else if scheduleType == "interval" && intervalStr == "" {
		c.ScheduleType = ScheduleNone
	}
	if scheduleType == "calendar" {
		ci := CalendarInterval{
			Minute:  parseOptionalInt(minuteStr),
			Hour:    parseOptionalInt(hourStr),
			Day:     parseOptionalInt(dayStr),
			Month:   parseOptionalInt(monthStr),
			Weekday: parseOptionalInt(weekdayStr),
		}
		if ci.HasValue() {
			c.Calendar = ci
		} else {
			c.ScheduleType = ScheduleNone
		}
	}
	if enableEnvVars {
		c.EnvironmentVars = parseEnvVars(envVarsStr)
	}
	defaultLogPath := func() string {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Logs", c.Label+".log")
	}
	switch logType {
	case "both":
		if c.StdoutPath == "" {
			c.StdoutPath = defaultLogPath()
		}
		if c.StderrPath == "" {
			c.StderrPath = c.StdoutPath
		}
	case "stdout":
		if c.StdoutPath == "" {
			c.StdoutPath = defaultLogPath()
		}
		c.StderrPath = ""
	case "stderr":
		c.StdoutPath = ""
		if c.StderrPath == "" {
			c.StderrPath = defaultLogPath()
		}
	default:
		c.StdoutPath = ""
		c.StderrPath = ""
	}
	c.StdoutPath = toAbsPath(c.StdoutPath)
	c.StderrPath = toAbsPath(c.StderrPath)

	return c, nil
}

// formTheme はフォーカス状態を強調するカスタムテーマを返す
func formTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// フォーカス中: 太い左ボーダー + 明るいタイトル
	cyan := lipgloss.AdaptiveColor{Light: "#0891b2", Dark: "#22d3ee"}
	dimGray := lipgloss.AdaptiveColor{Light: "#a1a1aa", Dark: "#71717a"}

	t.Focused.Base = t.Focused.Base.
		BorderForeground(cyan)
	t.Focused.Title = t.Focused.Title.
		Foreground(cyan).Bold(true)

	// 非フォーカス: 薄いグレーで控えめに
	t.Blurred.Base = t.Blurred.Base.
		BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Title = t.Blurred.Title.
		Foreground(dimGray).Bold(false)
	t.Blurred.Description = t.Blurred.Description.
		Foreground(dimGray)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.
		Foreground(dimGray)

	return t
}

// toAbsPath は~展開と絶対パス変換を行う。空文字列はそのまま返す。
func toAbsPath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	path, _ = filepath.Abs(path)
	return path
}

// resolveLabel はConfigからLabel（未入力ならProgramのベース名）を返す
func resolveLabel(c *Config) string {
	if c.Label != "" {
		return c.Label
	}
	fields := strings.Fields(c.Program)
	if len(fields) > 0 {
		return filepath.Base(fields[0])
	}
	return ""
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
