package server

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/global"
)

func TestMain(m *testing.M) {
	config.InitLog()
	global.Logger.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func captureLogs(t *testing.T, output *bytes.Buffer) {
	t.Helper()
	previous := global.Logger.Out
	global.Logger.SetOutput(output)
	t.Cleanup(func() { global.Logger.SetOutput(previous) })
}
