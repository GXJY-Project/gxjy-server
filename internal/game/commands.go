package game

// 命令编号来自原 protocol_map.json，回包用途由 Session.dispatch 和 B224 消费点确认
const (
	CommandAuth              uint16 = 101
	CommandHeartbeat         uint16 = 901
	CommandRedPoints         uint16 = 903
	CommandClientTracking    uint16 = 905
	CommandPushReminds       uint16 = 906
	CommandRoleInfo          uint16 = 1001
	CommandStaminaRecovery   uint16 = 1002
	CommandRandomRoleName    uint16 = 1003
	CommandUpdateRoleName    uint16 = 1004
	CommandRoleBoard         uint16 = 1005
	CommandSaveBoardChange   uint16 = 1006
	CommandFrontSettings     uint16 = 1011
	CommandSystemUnlock      uint16 = 1012
	CommandSaveFrontSettings uint16 = 1013
	CommandChooseLeader      uint16 = 1015
	CommandDownloadReward    uint16 = 1016
	CommandLoginInfo         uint16 = 1017
	CommandHeroes            uint16 = 1051
	CommandResourceMap       uint16 = 1151
	CommandTeams             uint16 = 1201
	CommandBagResponse       uint16 = 1251
	CommandBagRequest        uint16 = 1254
	CommandChapters          uint16 = 1301
	CommandChapterAutoInfo   uint16 = 1307
	CommandMails             uint16 = 1502
	CommandTasks             uint16 = 1552
	CommandDrawSystem        uint16 = 1601
	CommandMonthCard         uint16 = 1670
	CommandRealArenaBasic    uint16 = 1701
	CommandRealArenaInfo     uint16 = 1702
	CommandCheckMonthCard    uint16 = 1752
	CommandArenaInfo         uint16 = 1802
	CommandArenaBasic        uint16 = 1810
	CommandWorldMap          uint16 = 1851
	CommandEnterScene        uint16 = 1852
	CommandSceneMove         uint16 = 1853
	CommandSceneMini         uint16 = 1854
	CommandExploreDP         uint16 = 1856
	CommandFinishSceneCase   uint16 = 1857
	CommandSceneSpeedup      uint16 = 1863
	CommandQuitScene         uint16 = 1864
	CommandActivities        uint16 = 1951
	CommandFaceNotices       uint16 = 1955
	CommandPVPRoom           uint16 = 2016
	CommandTowerInfo         uint16 = 2101
	CommandFairArenaInfo     uint16 = 2351
	CommandSummitRequest     uint16 = 2401
	CommandSummitBase        uint16 = 2419
	// CommandRogueInfo 来自 B224 SERVER_Protocol，旧原型没有对应分支
	CommandRogueInfo uint16 = 2551
	// CommandFightTogetherInfo 来自 B224 SERVER_Protocol，旧原型编号表没有此项
	CommandFightTogetherInfo uint16 = 2601
	CommandFriends           uint16 = 10101
	CommandRecommendFriends  uint16 = 10102
	CommandChatInfo          uint16 = 10115
	CommandBlacklist         uint16 = 10124
	CommandChatList          uint16 = 10129
	CommandGuild             uint16 = 10151
	CommandRecommendGuilds   uint16 = 10152
	CommandGuildInvites      uint16 = 10157
	CommandSearchGuild       uint16 = 10167
	CommandLastGuilds        uint16 = 10175
)

// CommandName 只登记已经迁移的客户端请求
func CommandName(command uint16) (string, bool) {
	switch command {
	case CommandAuth:
		return "MSG_TYPE_AUTH", true
	case CommandHeartbeat:
		return "MSG_HEARTBEAT", true
	case CommandStaminaRecovery:
		return "MSG_STAMINA_RECOVERY", true
	case CommandClientTracking:
		return "MSG_CLIENT_TRACKING", true
	case CommandRandomRoleName:
		return "MSG_RANDOM_ROLE_NAME", true
	case CommandUpdateRoleName:
		return "MSG_UPDATE_ROLE_NAME", true
	case CommandRoleBoard:
		return "MSG_REQ_ROLE_BOARD", true
	case CommandSaveBoardChange:
		return "MSG_SAVE_BOARD_CHANGE", true
	case CommandFrontSettings:
		return "MSG_REQ_FRONT_SETTINGS", true
	case CommandSaveFrontSettings:
		return "MSG_SAVE_FRONT_SETTINGS", true
	case CommandChooseLeader:
		return "MSG_CHOOSE_INIT_LEADER", true
	case CommandDownloadReward:
		return "MSG_REQ_DOWNLOAD_REWARD", true
	case CommandBagRequest:
		return "MSG_REQ_BAG", true
	case CommandChapters:
		return "MSG_CHAPTERS", true
	case CommandChapterAutoInfo:
		return "MSG_GET_CHAPTER_AUTO_INFO", true
	case CommandMails:
		return "MSG_REQ_MAILS", true
	case CommandCheckMonthCard:
		return "MSG_REQ_CHECK_BUY_MONTH_CARD", true
	case CommandFaceNotices:
		return "MSG_REQ_FACE_NOTICES", true
	case CommandSummitRequest:
		return "MSG_GET_SUMMIT_BASE_INFO", true
	case CommandMonthCard:
		return "MSG_REQ_MONTH_CARD", true
	case CommandTasks:
		return "MSG_REQ_TASK", true
	case CommandResourceMap:
		return "MSG_REQ_RESOURCE_MAP", true
	case CommandDrawSystem:
		return "MSG_DRAWCARD_SYSTEM", true
	case CommandRealArenaBasic:
		return "MSG_REAL_ARENA_BASIC", true
	case CommandRealArenaInfo:
		return "MSG_REAL_ARENA_INFO", true
	case CommandArenaInfo:
		return "MSG_ARENA_INFO", true
	case CommandArenaBasic:
		return "MSG_ARENA_BASIC", true
	case CommandPVPRoom:
		return "MSG_GET_PVP_ROOM", true
	case CommandTowerInfo:
		return "MSG_REQ_TOWER_INFO", true
	case CommandFairArenaInfo:
		return "MSG_FAIR_ARENA_INFO", true
	case CommandActivities:
		return "MSG_REQ_ACTIVITIES", true
	case CommandFriends:
		return "MSG_GET_FRIENDS", true
	case CommandRecommendFriends:
		return "MSG_GET_RECOMMEND_FRIENDS", true
	case CommandChatInfo:
		return "MSG_GET_CHAT_INFO", true
	case CommandBlacklist:
		return "MSG_REQ_BLACKLIST", true
	case CommandChatList:
		return "MSG_GET_CHAT_LIST", true
	case CommandGuild:
		return "MSG_GET_CLUB", true
	case CommandRecommendGuilds:
		return "MSG_GET_RECOMMEND_CLUBS", true
	case CommandGuildInvites:
		return "MSG_GET_CLUB_INVITES", true
	case CommandSearchGuild:
		return "MSG_SEARCH_CLUB", true
	case CommandLastGuilds:
		return "MSG_LAST_RECOMMEND_CLUBS_REQ", true
	case CommandFightTogetherInfo:
		return "MSG_GET_FT_INFO", true
	case CommandRogueInfo:
		return "MSG_REQUEST_ROGUE_INFO", true
	case CommandWorldMap:
		return "MSG_WORLD_MAP", true
	case CommandEnterScene:
		return "MSG_ENTER_SCENE", true
	case CommandSceneMove:
		return "MSG_SCENE_MOVE", true
	case CommandSceneMini:
		return "MSG_SCENE_MINI", true
	case CommandFinishSceneCase:
		return "MSG_SCENE_FINISH_CASE", true
	case CommandSceneSpeedup:
		return "MSG_SCENE_SPEEDUP", true
	case CommandQuitScene:
		return "MSG_QUIT_SCENE", true
	default:
		return "", false
	}
}
