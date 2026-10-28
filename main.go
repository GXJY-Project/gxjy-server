package main

import (
	"os"

	"github.com/GXJY-Project/gxjy-server/config"
	"github.com/GXJY-Project/gxjy-server/flag"
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/GXJY-Project/gxjy-server/router"
)

func main() {
	flag.Parse()
	config.InitLog()
	if err := run(); err != nil {
		global.Logger.Error("启动失败：", err)
		os.Exit(1)
	}
}

func run() error {
	if flag.PrintVersion() {
		return nil
	}
	if err := config.InitConfig(flag.ConfigFile()); err != nil {
		return err
	}
	flag.Apply()
	if err := config.InitDB(); err != nil {
		return err
	}
	defer config.CloseDB()
	if done, err := flag.Run(); done || err != nil {
		return err
	}
	return router.Run()
}
