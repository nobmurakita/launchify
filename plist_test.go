package main

import (
	"strings"
	"testing"
)

func TestGeneratePlist_Minimal(t *testing.T) {
	c := Config{
		Label:   "com.user.test",
		Program: "/usr/local/bin/test",
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>Label</key>")
	mustContain(t, plist, "<string>com.user.test</string>")
	mustContain(t, plist, "<key>ProgramArguments</key>")
	mustContain(t, plist, "<string>/usr/local/bin/test</string>")
	mustNotContain(t, plist, "StandardOutPath")
	mustNotContain(t, plist, "StandardErrorPath")
}

func TestGeneratePlist_WithArgs(t *testing.T) {
	c := Config{
		Label:   "com.user.test",
		Program: "/usr/local/bin/test --flag value",
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<string>/usr/local/bin/test</string>")
	mustContain(t, plist, "<string>--flag</string>")
	mustContain(t, plist, "<string>value</string>")
	mustNotContain(t, plist, "<string>/usr/local/bin/test --flag value</string>")
}

func TestGeneratePlist_WithQuotedArgs(t *testing.T) {
	c := Config{
		Label:   "com.user.test",
		Program: `/usr/local/bin/test --msg "hello world"`,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<string>/usr/local/bin/test</string>")
	mustContain(t, plist, "<string>--msg</string>")
	mustContain(t, plist, "<string>hello world</string>")
}

func TestParseProgramArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"/bin/cmd", []string{"/bin/cmd"}},
		{"/bin/cmd -a -b", []string{"/bin/cmd", "-a", "-b"}},
		{`/bin/cmd --msg "hello world"`, []string{"/bin/cmd", "--msg", "hello world"}},
		{`/bin/cmd 'single quoted'`, []string{"/bin/cmd", "single quoted"}},
		{`/bin/cmd --a="val"`, []string{"/bin/cmd", "--a=val"}},
	}
	for _, tt := range tests {
		got := parseProgramArgs(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseProgramArgs(%q): got %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseProgramArgs(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestGeneratePlist_WithFileLog(t *testing.T) {
	c := Config{
		Label:      "com.user.test",
		Program:    "/usr/local/bin/test",
		StdoutPath: "/tmp/test.log",
		StderrPath: "/tmp/test.log",
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>StandardOutPath</key>")
	mustContain(t, plist, "<key>StandardErrorPath</key>")
	mustContain(t, plist, "<string>/tmp/test.log</string>")
}

func TestGeneratePlist_WithSeparateLogPaths(t *testing.T) {
	c := Config{
		Label:      "com.user.test",
		Program:    "/usr/local/bin/test",
		StdoutPath: "/tmp/stdout.log",
		StderrPath: "/tmp/stderr.log",
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>StandardOutPath</key>")
	mustContain(t, plist, "<string>/tmp/stdout.log</string>")
	mustContain(t, plist, "<key>StandardErrorPath</key>")
	mustContain(t, plist, "<string>/tmp/stderr.log</string>")
}

func TestGeneratePlist_WithInterval(t *testing.T) {
	c := Config{
		Label:         "com.user.test",
		Program:       "/usr/local/bin/test",
		ScheduleType:  ScheduleInterval,
		StartInterval: 3600,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>StartInterval</key>")
	mustContain(t, plist, "<integer>3600</integer>")
}

func TestGeneratePlist_WithCalendar(t *testing.T) {
	c := Config{
		Label:        "com.user.test",
		Program:      "/usr/local/bin/test",
		ScheduleType: ScheduleCalendar,
		Calendar:     CalendarInterval{Minute: intPtr(30), Hour: intPtr(9)},
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>StartCalendarInterval</key>")
	mustContain(t, plist, "<key>Minute</key>")
	mustContain(t, plist, "<integer>30</integer>")
	mustContain(t, plist, "<key>Hour</key>")
	mustContain(t, plist, "<integer>9</integer>")
}

func TestGeneratePlist_WithCalendarMidnight(t *testing.T) {
	// Minute=0, Hour=0 のケース — 0値のポインタが正しく出力されることを確認
	c := Config{
		Label:        "com.user.test",
		Program:      "/usr/local/bin/test",
		ScheduleType: ScheduleCalendar,
		Calendar:     CalendarInterval{Minute: intPtr(0), Hour: intPtr(0)},
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>StartCalendarInterval</key>")
	mustContain(t, plist, "<key>Minute</key>")
	mustContain(t, plist, "<key>Hour</key>")
	count := strings.Count(plist, "<integer>0</integer>")
	if count != 2 {
		t.Errorf("<integer>0</integer> の出現回数: got %d, want 2\n%s", count, plist)
	}
}

func TestGeneratePlist_KeepAliveAlways(t *testing.T) {
	c := Config{
		Label:     "com.user.test",
		Program:   "/usr/local/bin/test",
		KeepAlive: KeepAliveAlways,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>KeepAlive</key>")
	mustContain(t, plist, "<true/>")
	mustNotContain(t, plist, "SuccessfulExit")
}

func TestGeneratePlist_KeepAliveOnFailure(t *testing.T) {
	c := Config{
		Label:     "com.user.test",
		Program:   "/usr/local/bin/test",
		KeepAlive: KeepAliveOnFailure,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>KeepAlive</key>")
	mustContain(t, plist, "<key>SuccessfulExit</key>")
	mustContain(t, plist, "<false/>")
}

func TestGeneratePlist_KeepAliveNone(t *testing.T) {
	c := Config{
		Label:     "com.user.test",
		Program:   "/usr/local/bin/test",
		KeepAlive: KeepAliveNone,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustNotContain(t, plist, "KeepAlive")
}

func TestGeneratePlist_ProcessTypeBackground(t *testing.T) {
	c := Config{
		Label:       "com.user.test",
		Program:     "/usr/local/bin/test",
		ProcessType: ProcessBackground,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>ProcessType</key>")
	mustContain(t, plist, "<string>Background</string>")
}

func TestGeneratePlist_ProcessTypeInteractive(t *testing.T) {
	c := Config{
		Label:       "com.user.test",
		Program:     "/usr/local/bin/test",
		ProcessType: ProcessInteractive,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>ProcessType</key>")
	mustContain(t, plist, "<string>Interactive</string>")
}

func TestGeneratePlist_ProcessTypeStandard(t *testing.T) {
	c := Config{
		Label:       "com.user.test",
		Program:     "/usr/local/bin/test",
		ProcessType: ProcessStandard,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustNotContain(t, plist, "ProcessType")
}

func TestGeneratePlist_XmlEscape(t *testing.T) {
	c := Config{
		Label:   "com.user.test&debug",
		Program: `/usr/local/bin/test --msg "a<b"`,
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<string>com.user.test&amp;debug</string>")
	mustContain(t, plist, "<string>a&lt;b</string>")
	mustNotContain(t, plist, "test&debug</string>")
}

func TestGeneratePlist_WithEnvVars(t *testing.T) {
	c := Config{
		Label:   "com.user.test",
		Program: "/usr/local/bin/test",
		EnvironmentVars: map[string]string{
			"PATH": "/usr/local/bin:/usr/bin",
			"HOME": "/Users/test",
		},
	}
	plist, err := GeneratePlist(&c)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, plist, "<key>EnvironmentVariables</key>")
	mustContain(t, plist, "<key>PATH</key>")
	mustContain(t, plist, "<string>/usr/local/bin:/usr/bin</string>")
	mustContain(t, plist, "<key>HOME</key>")
	mustContain(t, plist, "<string>/Users/test</string>")
}

func mustContain(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("plistに %q が含まれていません:\n%s", substr, s)
	}
}

func mustNotContain(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("plistに %q が含まれるべきではありません", substr)
	}
}
