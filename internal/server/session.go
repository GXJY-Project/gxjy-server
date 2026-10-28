package server

import (
	"errors"
	"fmt"
	"io"

	"github.com/GXJY-Project/gxjy-server/global"

	gxjycrypto "github.com/GXJY-Project/gxjy-server/internal/crypto"
	"github.com/GXJY-Project/gxjy-server/internal/game"
	"github.com/GXJY-Project/gxjy-server/internal/protocol"
)

var errSessionKeyMissing = errors.New("加密消息缺少会话密钥")

type gameSession struct {
	connection *webSocketConn
	rsaKey     *gxjycrypto.RSAKey
	state      *GameState
	names      map[uint16]string

	key   []byte
	seq   uint32
	plain bool
}

func (session *gameSession) run() error {
	first := true
	for {
		data, err := session.connection.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if first {
			session.plain = protocol.LooksLikePlain(data)
			first = false
			global.Logger.Infof("识别游戏协议：plain:%t", session.plain)
		}

		decoded, err := session.decode(data)
		if err != nil {
			return err
		}
		if decoded.UsedTextFallback {
			return fmt.Errorf("游戏消息 MessagePack 解码失败：command=%d", decoded.Command)
		}
		global.Logger.Infof("收到游戏消息：command:%d name:%s", decoded.Command, session.name(decoded.Command))
		if err := session.dispatch(decoded.Command, decoded.Message); err != nil {
			return err
		}
	}
}

func (session *gameSession) decode(data []byte) (protocol.DecodedBody, error) {
	if session.plain {
		decoded, _, err := protocol.ParsePlainFrame(data)
		return decoded, err
	}

	packet, err := protocol.ParseClientPacket(data)
	if err != nil {
		return protocol.DecodedBody{}, err
	}
	if len(packet.RSA) > 0 {
		if session.rsaKey == nil {
			return protocol.DecodedBody{}, errors.New("服务端未配置 RSA 私钥")
		}
		session.key, err = gxjycrypto.RecoverSessionKey(packet.RSA, session.rsaKey)
		if err != nil {
			return protocol.DecodedBody{}, fmt.Errorf("恢复会话密钥：%w", err)
		}
	}
	if packet.EncryptionType == 0 || packet.EncryptionType == 2 {
		return protocol.DecodeBody(packet.Ciphertext)
	}
	if len(session.key) == 0 {
		return protocol.DecodedBody{}, errSessionKeyMissing
	}
	return protocol.DecodeClientBody(packet.Ciphertext, session.key)
}

func (session *gameSession) dispatch(command uint16, message protocol.Value) error {
	if _, supported := game.CommandName(command); !supported {
		return fmt.Errorf("不支持游戏命令：command=%d name=%s", command, session.name(command))
	}
	messages, err := session.state.handle(command, message)
	if err != nil {
		return err
	}
	for _, out := range messages {
		value, err := anyProtocolValue(out.Payload)
		if err != nil {
			return err
		}
		if err = session.send(out.Command, value); err != nil {
			return err
		}
	}
	return nil
}

func (session *gameSession) send(command uint16, message protocol.Value) error {
	session.seq++
	var (
		frame []byte
		err   error
	)
	if session.plain {
		frame, err = protocol.BuildPlainFrame(command, message, session.seq)
	} else {
		if len(session.key) == 0 {
			return errSessionKeyMissing
		}
		frame, err = protocol.BuildServerFrame(command, message, session.seq, session.key, 1, nil)
	}
	if err != nil {
		return err
	}
	if err := session.connection.writeBinary(frame); err != nil {
		return err
	}
	global.Logger.Infof("发送游戏消息：command:%d name:%s sequence:%d", command, session.name(command), session.seq)
	return nil
}

func (session *gameSession) name(command uint16) string {
	if name := session.names[command]; name != "" {
		return name
	}
	if name := builtInCommandNames[command]; name != "" {
		return name
	}
	if name, supported := game.CommandName(command); supported {
		return name
	}
	return fmt.Sprintf("MSG_%d", command)
}

var builtInCommandNames = map[uint16]string{
	commandAuth:                   "MSG_TYPE_AUTH",
	commandHeartbeat:              "MSG_HEARTBEAT",
	commandRedPoints:              "MSG_RED_POINTS_RES",
	commandPushReminds:            "MSG_PUSH_REMINDS",
	commandRoleInfo:               "MSG_ROLE_INFO_RES",
	commandRandomRoleName:         "MSG_RANDOM_ROLE_NAME",
	commandUpdateRoleName:         "MSG_UPDATE_ROLE_NAME",
	game.CommandRoleBoard:         "MSG_REQ_ROLE_BOARD",
	game.CommandSaveBoardChange:   "MSG_SAVE_BOARD_CHANGE",
	commandFrontSettings:          "MSG_REQ_FRONT_SETTINGS",
	commandSaveFrontSettings:      "MSG_SAVE_FRONT_SETTINGS",
	commandSystemUnlock:           "MSG_SYSTEM_UNLOCK_RES",
	commandChooseLeader:           "MSG_CHOOSE_INIT_LEADER",
	commandDownloadReward:         "MSG_REQ_DOWNLOAD_REWARD",
	commandLoginInfo:              "MSG_LOGIN_INFO_RES",
	commandHeroes:                 "MSG_HEROS_RES",
	commandTeams:                  "MSG_TEAMS_RES",
	commandBagRequest:             "MSG_REQ_BAG",
	commandBagResponse:            "MSG_BAG_RES",
	commandChapters:               "MSG_CHAPTERS",
	game.CommandMonthCard:         "MSG_REQ_MONTH_CARD",
	game.CommandTasks:             "MSG_REQ_TASK",
	game.CommandChatInfo:          "MSG_GET_CHAT_INFO",
	game.CommandFriends:           "MSG_GET_FRIENDS",
	game.CommandBlacklist:         "MSG_REQ_BLACKLIST",
	game.CommandChatList:          "MSG_GET_CHAT_LIST",
	game.CommandPVPRoom:           "MSG_GET_PVP_ROOM",
	game.CommandGuild:             "MSG_GET_CLUB",
	game.CommandTowerInfo:         "MSG_REQ_TOWER_INFO",
	game.CommandResourceMap:       "MSG_REQ_RESOURCE_MAP",
	game.CommandDrawSystem:        "MSG_DRAWCARD_SYSTEM",
	game.CommandRealArenaBasic:    "MSG_REAL_ARENA_BASIC",
	game.CommandRealArenaInfo:     "MSG_REAL_ARENA_INFO",
	game.CommandArenaBasic:        "MSG_ARENA_BASIC",
	game.CommandArenaInfo:         "MSG_ARENA_INFO",
	game.CommandFairArenaInfo:     "MSG_FAIR_ARENA_INFO",
	game.CommandClientTracking:    "MSG_CLIENT_TRACKING",
	game.CommandFightTogetherInfo: "MSG_GET_FT_INFO",
	game.CommandRogueInfo:         "MSG_REQUEST_ROGUE_INFO",
	game.CommandExploreDP:         "MSG_SCENE_EXPLORE_DP",
	commandWorldMap:               "MSG_WORLD_MAP",
	commandActivities:             "MSG_REQ_ACTIVITIES",
	commandSummitBase:             "MSG_SYNC_BASE_INFO",
}
