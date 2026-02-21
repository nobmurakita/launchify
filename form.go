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
	var enableLog bool
	var intervalStr string
	var minuteStr, hourStr, dayStr, monthStr, weekdayStr string
	var envVarsStr string

	theme := formTheme()

	form := huh.NewForm(
		// グループ1: 基本設定
		huh.NewGroup(
			huh.NewInput().
				Title("Label").
				Description("LaunchAgentの識別子（例: com.user.mytool）").
				Value(&c.Label).
				Validate(huh.ValidateNotEmpty()),

			huh.NewInput().
				Title("Program").
				Description("実行するコマンド").
				Value(&c.Program).
				Validate(huh.ValidateNotEmpty()),

			huh.NewInput().
				Title("WorkingDirectory").
				Description("作業ディレクトリ（省略可）").
				Value(&c.WorkingDirectory),
		),

		// グループ2: 実行条件
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

			huh.NewConfirm().
				Title("RunAtLoad").
				Description("ロード時に即実行する").
				Value(&c.RunAtLoad),

			huh.NewSelect[string]().
				Title("KeepAlive").
				Options(
					huh.NewOption("しない", "none"),
					huh.NewOption("常に再起動", "always"),
					huh.NewOption("異常終了時のみ再起動", "on_failure"),
				).
				Value(&keepAlive),

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

		// グループ3: 環境変数
		huh.NewGroup(
			huh.NewText().
				Title("環境変数").
				Description("KEY=VALUE 形式で1行ずつ入力（省略可）").
				Lines(4).
				Value(&envVarsStr),
		),

		// グループ4: ログ出力
		huh.NewGroup(
			huh.NewConfirm().
				Title("ログファイル出力").
				Description("stdout/stderrをファイルに出力する").
				Value(&enableLog),
		),

		// グループ4.5: ログファイルパス（条件付き）
		huh.NewGroup(
			huh.NewInput().
				Title("ログファイルパス").
				DescriptionFunc(func() string {
					return "空欄の場合: ~/Library/Logs/" + c.Label + ".log"
				}, &c.Label).
				Value(&c.LogFilePath),
		).WithHideFunc(func() bool {
			return !enableLog
		}),
	)

	if err := form.WithTheme(theme).Run(); err != nil {
		return nil, err
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
	c.EnvironmentVars = parseEnvVars(envVarsStr)
	if enableLog && c.LogFilePath == "" {
		home, _ := os.UserHomeDir()
		c.LogFilePath = filepath.Join(home, "Library", "Logs", c.Label+".log")
	}
	if !enableLog {
		c.LogFilePath = ""
	}

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
