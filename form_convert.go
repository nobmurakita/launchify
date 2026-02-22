package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
