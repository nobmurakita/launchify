package main

import (
	"bytes"
	"fmt"
	"html"
	"maps"
	"slices"
	"text/template"

	"github.com/google/shlex"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label | xmlEscape}}</string>

	<key>ProgramArguments</key>
	<array>
{{- range .ProgramArgs}}
		<string>{{. | xmlEscape}}</string>
{{- end}}
	</array>
{{- if .ProcessType}}

	<key>ProcessType</key>
	<string>{{.ProcessType}}</string>
{{- end}}
{{- if .WorkingDirectory}}

	<key>WorkingDirectory</key>
	<string>{{.WorkingDirectory | xmlEscape}}</string>
{{- end}}
{{- if .RunAtLoad}}

	<key>RunAtLoad</key>
	<true/>
{{- end}}
{{- if eq .KeepAlive "always"}}

	<key>KeepAlive</key>
	<true/>
{{- else if eq .KeepAlive "on_failure"}}

	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
{{- end}}
{{- if .StartInterval}}

	<key>StartInterval</key>
	<integer>{{.StartInterval}}</integer>
{{- end}}
{{- if .CalendarInterval}}

	<key>StartCalendarInterval</key>
	<dict>
{{- if notNil .CalendarInterval.Minute}}
		<key>Minute</key>
		<integer>{{deref .CalendarInterval.Minute}}</integer>
{{- end}}
{{- if notNil .CalendarInterval.Hour}}
		<key>Hour</key>
		<integer>{{deref .CalendarInterval.Hour}}</integer>
{{- end}}
{{- if notNil .CalendarInterval.Day}}
		<key>Day</key>
		<integer>{{deref .CalendarInterval.Day}}</integer>
{{- end}}
{{- if notNil .CalendarInterval.Month}}
		<key>Month</key>
		<integer>{{deref .CalendarInterval.Month}}</integer>
{{- end}}
{{- if notNil .CalendarInterval.Weekday}}
		<key>Weekday</key>
		<integer>{{deref .CalendarInterval.Weekday}}</integer>
{{- end}}
	</dict>
{{- end}}
{{- if .EnvironmentVars}}

	<key>EnvironmentVariables</key>
	<dict>
{{- range .EnvironmentVars}}
		<key>{{.Key | xmlEscape}}</key>
		<string>{{.Value | xmlEscape}}</string>
{{- end}}
	</dict>
{{- end}}
{{- if .StdoutPath}}

	<key>StandardOutPath</key>
	<string>{{.StdoutPath | xmlEscape}}</string>
{{- end}}
{{- if .StderrPath}}

	<key>StandardErrorPath</key>
	<string>{{.StderrPath | xmlEscape}}</string>
{{- end}}
</dict>
</plist>
`

// plistTmpl は事前パース済みのplistテンプレート
var plistTmpl = template.Must(
	template.New("plist").Funcs(template.FuncMap{
		"deref":     func(p *int) int { return *p },
		"notNil":    func(p *int) bool { return p != nil },
		"xmlEscape": html.EscapeString,
	}).Parse(plistTemplate),
)

// envVar は環境変数のキー・値ペア（ソート済み出力用）
type envVar struct {
	Key   string
	Value string
}

// plistData はテンプレートに渡すデータ
type plistData struct {
	Label            string
	ProgramArgs      []string
	WorkingDirectory string
	ProcessType      ProcessType
	RunAtLoad        bool
	KeepAlive        KeepAliveType
	StartInterval    int
	CalendarInterval *CalendarInterval
	EnvironmentVars  []envVar
	StdoutPath string
	StderrPath string
}

// GeneratePlist はConfigからplist XML文字列を生成する
func GeneratePlist(c *Config) (string, error) {
	data := buildPlistData(c)
	return renderPlist(data)
}

// buildPlistData はConfigからテンプレート用のデータを構築する
func buildPlistData(c *Config) plistData {
	data := plistData{
		Label:            c.Label,
		ProgramArgs:      parseProgramArgs(c.Program),
		WorkingDirectory: c.WorkingDirectory,
		RunAtLoad:        c.RunAtLoad,
		KeepAlive:        c.KeepAlive,
		StdoutPath: c.StdoutPath,
		StderrPath: c.StderrPath,
	}

	if c.ProcessType != ProcessStandard {
		data.ProcessType = c.ProcessType
	}

	if c.ScheduleType == ScheduleInterval {
		data.StartInterval = c.StartInterval
	}

	if c.ScheduleType == ScheduleCalendar && c.Calendar.HasValue() {
		data.CalendarInterval = &c.Calendar
	}

	if len(c.EnvironmentVars) > 0 {
		keys := slices.Sorted(maps.Keys(c.EnvironmentVars))
		vars := make([]envVar, len(keys))
		for i, k := range keys {
			vars[i] = envVar{Key: k, Value: c.EnvironmentVars[k]}
		}
		data.EnvironmentVars = vars
	}

	return data
}

// renderPlist はplistDataからXML文字列をレンダリングする
func renderPlist(data plistData) (string, error) {
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("テンプレートの実行に失敗: %w", err)
	}
	return buf.String(), nil
}

// parseProgramArgs はコマンド文字列をシェルの字句解析ルールで分割する。
// パースに失敗した場合はそのまま単一要素として返す。
func parseProgramArgs(s string) []string {
	args, err := shlex.Split(s)
	if err != nil {
		return []string{s}
	}
	return args
}
