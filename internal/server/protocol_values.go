package server

import (
	"fmt"

	"github.com/GXJY-Project/gxjy-server/internal/assets"
	"github.com/GXJY-Project/gxjy-server/internal/game"
	"github.com/GXJY-Project/gxjy-server/internal/game/hero"
	"github.com/GXJY-Project/gxjy-server/internal/protocol"
)

const (
	commandAuth              uint16 = game.CommandAuth
	commandHeartbeat         uint16 = game.CommandHeartbeat
	commandRedPoints         uint16 = game.CommandRedPoints
	commandPushReminds       uint16 = game.CommandPushReminds
	commandRoleInfo          uint16 = game.CommandRoleInfo
	commandRandomRoleName    uint16 = game.CommandRandomRoleName
	commandUpdateRoleName    uint16 = game.CommandUpdateRoleName
	commandFrontSettings     uint16 = game.CommandFrontSettings
	commandSystemUnlock      uint16 = game.CommandSystemUnlock
	commandSaveFrontSettings uint16 = game.CommandSaveFrontSettings
	commandChooseLeader      uint16 = game.CommandChooseLeader
	commandDownloadReward    uint16 = game.CommandDownloadReward
	commandLoginInfo         uint16 = game.CommandLoginInfo
	commandHeroes            uint16 = game.CommandHeroes
	commandTeams             uint16 = game.CommandTeams
	commandBagRequest        uint16 = game.CommandBagRequest
	commandBagResponse       uint16 = game.CommandBagResponse
	commandChapters          uint16 = game.CommandChapters
	commandWorldMap          uint16 = game.CommandWorldMap
	commandActivities        uint16 = game.CommandActivities
	commandSummitBase        uint16 = game.CommandSummitBase
	defaultRoleID                   = "10001"
	defaultRoleName                 = "测试玩家"
)

// GameState 将业务状态接到网络层
type GameState struct{ runtime *game.Runtime }

func NewGameState(loader *assets.TableLoader) (*GameState, error) {
	return NewGameStateWithStore(loader, nil)
}
func NewGameStateWithStore(loader *assets.TableLoader, store game.Store) (*GameState, error) {
	if loader == nil {
		return nil, hero.ErrTableSourceRequired
	}
	return NewGameStateFromSource(loader, store, game.Options{})
}

// NewGameStateFromSource 允许差分测试注入同一份配置、时钟和随机输入
func NewGameStateFromSource(source hero.TableSource, store game.Store, options game.Options) (*GameState, error) {
	runtime, err := game.New(source, store, options)
	if err != nil {
		return nil, err
	}
	return &GameState{runtime: runtime}, nil
}
func (state *GameState) handle(command uint16, message protocol.Value) ([]game.Message, error) {
	if state == nil || state.runtime == nil {
		return nil, hero.ErrTableSourceRequired
	}
	data, err := protocolValueJSON(message)
	if err != nil {
		return nil, err
	}
	value, err := decodeRequestJSON(data)
	if err != nil {
		return nil, fmt.Errorf("解码请求: %w", err)
	}
	return state.runtime.Handle(command, value)
}
func (state *GameState) updateRole(command uint16, message protocol.Value) error {
	_, err := state.handle(command, message)
	return err
}

func object(fields ...protocol.Field) protocol.Value {
	return protocol.Object(fields...)
}

func field(name string, value protocol.Value) protocol.Field {
	return protocol.Named(name, value)
}

func trueMap(keys []string) protocol.Value {
	fields := make([]protocol.Field, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, field(key, protocol.Bool(true)))
	}
	return protocol.Object(fields...)
}

func result(commandName string) protocol.Value {
	return object(
		field("result", protocol.Uint(0)),
		field("cmd", protocol.String(commandName)),
	)
}

func unwrapEnvelope(message protocol.Value) protocol.Value {
	if message.Kind != protocol.KindObject {
		return message
	}
	_, hasSequence := valueField(message, "seq")
	inner, hasMessage := valueField(message, "msg")
	if hasSequence && hasMessage {
		return inner
	}
	return message
}

func stringField(value protocol.Value, name string) (string, bool) {
	found, ok := valueField(value, name)
	return found.Text, ok && found.Kind == protocol.KindString
}

func valueField(value protocol.Value, name string) (protocol.Value, bool) {
	if value.Kind != protocol.KindObject {
		return protocol.Value{}, false
	}
	for _, item := range value.Fields {
		if item.Key.Kind == protocol.KindString && item.Key.Text == name {
			return item.Value, true
		}
	}
	return protocol.Value{}, false
}
