package models

// Player 保存当前业务的完整玩家状态，数据库模型不参与协议回包
type Player struct {
	RoleID int64  `gorm:"primaryKey;autoIncrement:false"`
	State  string `gorm:"type:text;not null"`
}
