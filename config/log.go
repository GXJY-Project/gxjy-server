package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/sirupsen/logrus"
)

// 日志颜色。
const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
)

// LogFormatter 输出时间、颜色、级别和调用位置。
type LogFormatter struct{}

// Format 实现 logrus.Formatter 接口。
func (formatter *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	levelColor := blue
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	}

	buffer := entry.Buffer
	if buffer == nil {
		buffer = &bytes.Buffer{}
	}
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	if entry.HasCaller() {
		functionName := entry.Caller.Function
		fileName := fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
		_, err := fmt.Fprintf(
			buffer,
			"[%s] \x1b[%dm[%s]\x1b[0m %s %s %s\n",
			timestamp,
			levelColor,
			entry.Level,
			fileName,
			functionName,
			entry.Message,
		)
		return buffer.Bytes(), err
	}
	_, err := fmt.Fprintf(
		buffer,
		"[%s] \x1b[%dm[%s]\x1b[0m %s\n",
		timestamp,
		levelColor,
		entry.Level,
		entry.Message,
	)
	return buffer.Bytes(), err
}

// InitLog 初始化全局日志，启动入口在加载配置前调用
func InitLog() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetReportCaller(true)
	logger.SetFormatter(&LogFormatter{})
	logger.SetLevel(logrus.DebugLevel)
	global.Logger = logger
}
