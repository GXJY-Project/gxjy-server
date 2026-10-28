package config

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLogFormatterWithoutCaller(t *testing.T) {
	formatter := &LogFormatter{}
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Time:    time.Date(2026, 10, 22, 13, 30, 0, 0, time.Local),
		Level:   logrus.WarnLevel,
		Message: "配置缺失",
	}
	got, err := formatter.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "[2026-10-22 13:30:00] \x1b[33m[warning]\x1b[0m 配置缺失\n"
	if string(got) != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestLogFormatterWithCaller(t *testing.T) {
	formatter := &LogFormatter{}
	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Time:    time.Date(2026, 10, 22, 13, 30, 0, 0, time.Local),
		Level:   logrus.InfoLevel,
		Message: "服务启动",
		Caller: &runtime.Frame{
			Function: "gxjy-server/config.TestLogFormatterWithCaller",
			File:     `E:\Go Project\gxjy\gxjy-server\config\log_test.go`,
			Line:     42,
		},
	}
	got, err := formatter.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "\x1b[36m[info]\x1b[0m") ||
		!strings.Contains(text, "log_test.go:42") ||
		!strings.Contains(text, entry.Caller.Function) {
		t.Fatalf("Format() = %q", text)
	}
}
