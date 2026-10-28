package storage

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/GXJY-Project/gxjy-server/models"
	playerrepo "github.com/GXJY-Project/gxjy-server/repositories/player"
	"gorm.io/gorm"
)

// PlayerStore 将现有游戏状态的序列化接到玩家 repo
type PlayerStore struct {
	repo   *playerrepo.Repo
	roleID int64
}

// NewPlayerStore 绑定当前运行角色
func NewPlayerStore(repo *playerrepo.Repo, roleID int64) *PlayerStore {
	return &PlayerStore{repo: repo, roleID: roleID}
}

// Load 读取角色状态，保留业务使用的大整数和自定义解码
func (s *PlayerStore) Load(target any) (bool, error) {
	player, err := s.repo.GetByRoleID(s.roleID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	decoder := json.NewDecoder(strings.NewReader(player.State))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return false, err
	}
	return true, nil
}

// Save 将业务状态写入当前角色
func (s *PlayerStore) Save(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.repo.Save(&models.Player{RoleID: s.roleID, State: string(data)})
}
