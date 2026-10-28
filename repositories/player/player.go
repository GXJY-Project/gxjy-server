package player

import (
	"github.com/GXJY-Project/gxjy-server/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repo 负责玩家数据的数据库读写
type Repo struct {
	db *gorm.DB
}

// NewRepo 复用已有连接池，事务内可传入事务句柄
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// GetByRoleID 查询角色，不存在时返回 gorm.ErrRecordNotFound
func (r *Repo) GetByRoleID(roleID int64) (models.Player, error) {
	var player models.Player
	err := r.db.Where("role_id = ?", roleID).First(&player).Error
	return player, err
}

// Save 创建或更新角色，更新时包含零值
func (r *Repo) Save(player *models.Player) error {
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(player).Error
}
