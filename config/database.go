package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 按运行配置初始化唯一的共享数据库连接池
func InitDB() error {
	db, err := OpenDB(AppConfig.Database)
	if err != nil {
		return err
	}
	global.DB = db
	global.Logger.Infof("数据库连接成功：driver:%s", AppConfig.Database.Driver)
	return nil
}

// CloseDB 在服务退出后关闭共享连接池
func CloseDB() {
	if global.DB == nil {
		return
	}
	pool, err := global.DB.DB()
	if err == nil {
		err = pool.Close()
	}
	if err != nil {
		global.Logger.Errorf("关闭数据库失败：%v", err)
	}
}

// Database 保存数据库连接配置，DSN 支持环境变量占位
type Database struct {
	Driver       string `json:"driver"`
	DSN          string `json:"dsn"`
	MaxIdleConns int    `json:"max_idle_conns"`
	MaxOpenConns int    `json:"max_open_conns"`
}

// OpenDB 初始化 GORM 和连接池，数据库差异只在此处选择驱动
func OpenDB(config Database) (*gorm.DB, error) {
	dsn := config.DSN
	if config.Driver == "postgres" {
		dsn = os.ExpandEnv(dsn)
	}
	if dsn == "" {
		return nil, errors.New("数据库 DSN 不能为空")
	}
	var dialector gorm.Dialector
	switch config.Driver {
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(dsn), 0o700); err != nil {
			return nil, fmt.Errorf("创建数据库目录：%w", err)
		}
		dialector = gormlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动：%q", config.Driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		// SQL 参数包含玩家状态，连接错误也可能包含 DSN，避免直接记录
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.New("数据库连接失败，请检查连接配置和数据库可用性")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池：%w", err)
	}
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}
