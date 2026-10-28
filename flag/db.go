package flag

import (
	"github.com/GXJY-Project/gxjy-server/global"
	"github.com/GXJY-Project/gxjy-server/models"
)

// Run 执行数据库命令，done=true 时入口直接退出，不启动游戏服务
func Run() (done bool, err error) {
	if !flagOptions.DB {
		return false, nil
	}
	if err := global.DB.AutoMigrate(&models.Player{}); err != nil {
		return true, err
	}
	global.Logger.Info("数据库模型同步成功")
	return true, nil
}
