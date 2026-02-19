package main

// ScheduleType はスケジュールの種別を表す
type ScheduleType string

const (
	ScheduleNone     ScheduleType = "none"
	ScheduleInterval ScheduleType = "interval"
	ScheduleCalendar ScheduleType = "calendar"
)

// KeepAliveType はKeepAliveの種別を表す
type KeepAliveType string

const (
	KeepAliveNone         KeepAliveType = "none"
	KeepAliveAlways       KeepAliveType = "always"
	KeepAliveOnFailure    KeepAliveType = "on_failure"
)

// Config はLaunchAgentの設定を保持する
type Config struct {
	Label            string
	Program          string
	WorkingDirectory string
	RunAtLoad        bool
	KeepAlive        KeepAliveType
	ScheduleType     ScheduleType
	StartInterval    int
	Calendar         CalendarInterval
	EnvironmentVars  map[string]string
	LogFilePath      string
}
