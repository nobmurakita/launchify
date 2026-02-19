package main

// CalendarInterval はlaunchdのStartCalendarIntervalに対応する構造体。
// 各フィールドがnilの場合、そのフィールドはplistに出力されない（ワイルドカード相当）。
type CalendarInterval struct {
	Minute  *int
	Hour    *int
	Day     *int
	Month   *int
	Weekday *int
}

// HasValue はいずれかのフィールドが設定されているかを返す
func (ci *CalendarInterval) HasValue() bool {
	return ci.Minute != nil || ci.Hour != nil || ci.Day != nil || ci.Month != nil || ci.Weekday != nil
}
