package main

import "testing"

func TestCalendarInterval_HasValue(t *testing.T) {
	ci := CalendarInterval{}
	if ci.HasValue() {
		t.Error("空のCalendarIntervalはHasValue=falseであるべき")
	}

	minute := 30
	ci.Minute = &minute
	if !ci.HasValue() {
		t.Error("Minuteが設定されているのでHasValue=trueであるべき")
	}
}

func intPtr(n int) *int {
	return &n
}
