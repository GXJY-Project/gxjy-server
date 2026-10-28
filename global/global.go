package global

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 启动阶段初始化，服务运行期间统一复用
var (
	Logger *logrus.Logger
	DB     *gorm.DB
)
