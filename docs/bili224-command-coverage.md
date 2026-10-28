# B224 客户端 request 静态覆盖清单

状态：已验证静态调用点和注册状态；请求实际是否发生、字段语义及实机通过均不能由本表推定

生成：`python -B scripts/scan_client_commands.py`，只读原脚本、配置和代码，不读取玩家存档

客户端 SHA-256：`89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6`；Python SHA-256：`13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25`

Go 受理 50 种请求；静态发现 369 个字面量命令、412 处调用；另有 34 处动态/其他 request 待人工核对

只统计 `.request(请求, 字面量命令, ...)`；不覆盖直接底层发包、动态命令名或全部主动推送

Go“已受理”只表示分发已接入，空态、固定原型模型、错误边界和业务范围须查实现文档

字面量未受理 323 个：编号已知且有 Python 显式分支 57 个、编号已知但未找到显式分支 256 个、未在 B224 协议表找到编号 10 个

“未找到显式分支”是静态扫描结果，仍须追踪动态分发和辅助函数；原型默认成功兜底不作为实现依据

| CMD | 名称 | 客户端 模块.方法:行 | Python 分支行 | Go 受理 |
|---:|---|---|---:|---|
| 101 | MSG_TYPE_AUTH | LoginModel.requestEnterGame:338613 | 8800 | 是 |
| 107 | MSG_CONFIRM_LOG_OFF | PlayerModel.requestMsgConfirmLogOff:383040 | 无显式分支 | 否 |
| 904 | MSG_CLEAR_RED_POINTS | PlayerModel.requestClearRedPoints:382651 | 无显式分支 | 否 |
| 905 | MSG_CLIENT_TRACKING | StatisticModel.requestStatistic:431344 | 9871 | 是 |
| 907 | MSG_REQ_OPEN_RED_GIFT | ActivityModel.requestOpenChatPackage:49873 | 无显式分支 | 否 |
| 1002 | MSG_STAMINA_RECOVERY | PlayerModel.requestPowerResetTime:382418 | 9509 | 是 |
| 1003 | MSG_RANDOM_ROLE_NAME | PlayerModel.requestRandomName:382277 | 8821 | 是 |
| 1004 | MSG_UPDATE_ROLE_NAME | PlayerModel.requestUpdateName:382286 | 8864 | 是 |
| 1005 | MSG_REQ_ROLE_BOARD | PlayerModel.requestBoardInfo:382333 | 9406 | 是 |
| 1006 | MSG_SAVE_BOARD_CHANGE | PlayerModel.requestSaveBoardChange:382318 | 9428 | 是 |
| 1007 | MSG_SAVE_MSETTING_CHANGE | PlayerModel.saveMessetingChange:382436 | 无显式分支 | 否 |
| 1010 | MSG_SAVE_HEAD_MESSAGE | SetModel.requestSaveHead:412685 | 无显式分支 | 否 |
| 1011 | MSG_REQ_FRONT_SETTINGS | CustomDataModel.doReset:200186<br>CustomDataModel.requestGet:200213 | 8875 | 是 |
| 1013 | MSG_SAVE_FRONT_SETTINGS | CustomDataModel.requestSave:200237 | 8886 | 是 |
| 1014 | MSG_GM_CMD | DebugModel.requestDebugOrder:202740 | 10074 | 否 |
| 1015 | MSG_CHOOSE_INIT_LEADER | PlayerModel.requestUpdateLearderId:382302 | 8836 | 是 |
| 1016 | MSG_REQ_DOWNLOAD_REWARD | PlayerModel.requestResourceReward:383007 | 10102 | 是 |
| 1018 | MSG_GET_ROLE_CARD_INFO | PlayerModel.getMyRoleInfo:382775 | 无显式分支 | 否 |
| 1019 | MSG_SET_SHOW_HEROES | PlayerModel.setShowHeros:382794 | 无显式分支 | 否 |
| 1020 | MSG_RECEIVE_STAMINA_BY_POLL | PlayerModel.recieveStaminaPoll:382815 | 无显式分支 | 否 |
| 1021 | MSG_OBTAIN_REMIND_REWARDS | PlayerModel.requestMsgRemindReward:383014 | 无显式分支 | 否 |
| 1022 | MSG_CDKEY_EXCHANGE | PlayerModel.requestMsgCdkeyExchange:383026 | 无显式分支 | 否 |
| 1052 | MSG_CALL_HERO | HeroListModel.requestActiveHero:312035 | 9557 | 否 |
| 1053 | MSG_USED_ITEM_UPGRADE | HeroListModel.requestLevelUp:312049 | 9557 | 否 |
| 1054 | MSG_UPGRADE_QUALITY | HeroListModel.requestQualityUp:312063 | 9557 | 否 |
| 1055 | MSG_UPGRADE_STAR | HeroListModel.requestStarUp:312077 | 9557 | 否 |
| 1057 | MSG_HERO_AWAKEN | HeroListModel.requestAwakeHero:312088 | 9557 | 否 |
| 1058 | MSG_USE_HERO_SURFACE | HeroListModel.requestChangeHero:312099 | 8825 | 否 |
| 1059 | MSG_CHANGE_DEVELOP_REMIND | HeroListModel.requestHeroChangeDe:312136 | 9136 | 否 |
| 1060 | MSG_REQ_HERO_PROFILE | HeroListModel.requesetHeroProfile:312127 | 9704 | 否 |
| 1061 | MSG_GET_HERO_RECEIVED_GIFT | HeroListModel.requestHeroReceivedGift:312148 | 9142 | 否 |
| 1062 | MSG_SEND_HERO_GIFT | HeroListModel.requestHeroGift:312164 | 9557 | 否 |
| 1063 | MSG_OBTAIN_HERO_FAVOR_REWARD | HeroListModel.requestObtainHeroGift:312178 | 9147 | 否 |
| 1065 | MSG_UPGRADE_ONE_HERO_FAVOR | HeroListModel.requestHeroOneUpgrade:312191 | 9147 | 否 |
| 1066 | MSG_OBTAIN_FAVOR_LEVEL_AWARD | HeroListModel.requestFavorUpgradeOneHero:312204 | 9730 | 否 |
| 1067 | MSG_REQ_COLLECT_INFO | FetterModel.requestFetterInfo:257542 | 9629 | 否 |
| 1068 | MSG_REQ_ACTIVATION_FETTER | FetterModel.requestActivationFetter:257551 | 9657 | 否 |
| 1070 | MSG_HERO_TASK_LIST | FetterModel.requsetFetterHeroTask:257559 | 9684 | 否 |
| 1071 | MSG_SAVE_SURFACES_OFFSET | PlayerModel.requestSaveSurfacesOffset:383032 | 8832 | 否 |
| 1073 | MSG_LEADER_COMMON_USE_SKILLS | TeamModel.requestLeaderCommonUseSkills:450031 | 无显式分支 | 否 |
| 1074 | MSG_TAKE_ON_SKILL | HeroListModel.requestTaskOnSkill:312414 | 无显式分支 | 否 |
| 1075 | MSG_SKILL_GROUP_CHANGE_NAME | HeroListModel.requestSkillGroupChangeName:312419 | 无显式分支 | 否 |
| 1076 | MSG_EXPAND_SKILL_GROUP | TeamModel.requestExpandSkillGroup:450047 | 无显式分支 | 否 |
| 1077 | MSG_SELECT_TALENT_TREE | HeroListModel.requestmsgSelectTalentTree:312425 | 无显式分支 | 否 |
| 1078 | MSG_UNLOCK_TALENT_POINT | HeroListModel.requsetUnlockTalentPoint:312435 | 无显式分支 | 否 |
| 1079 | MSG_CHOOSE_UNLOCK_LEADER | HeroListModel.requsetChooseUnlockLeader:312467 | 无显式分支 | 否 |
| 1080 | MSG_RESET_LEADER_TALENT | HeroListModel.requsetResetLeaderTalent:312476 | 无显式分支 | 否 |
| 1081 | MSG_AWAKEN_OPEN | HeroListModel.requsetAwakenOpen:312485 | 无显式分支 | 否 |
| 1082 | MSG_AWAKEN_POINT_UPGRADE | HeroListModel.requsetAwakenPointUpgrade:312494 | 无显式分支 | 否 |
| 1151 | MSG_REQ_RESOURCE_MAP | ResourcesModel.requestBlockData:395623 | 9512 | 是 |
| 1152 | MSG_CHALLENGE_RESOURCE_MAP_POINT | ResourcesModel.requestChallengePoint:395832 | 9520 | 否 |
| 1154 | MSG_SWEEP_RESOURCE_POINT | ResourcesModel.requestSweepPoint:395931 | 无显式分支 | 否 |
| 1202 | MSG_SAVE_TEAM | TeamModel.requestSaveTeam:449942 | 9274 | 否 |
| 1203 | MSG_UPDATE_TEAM_NAME | TeamModel.requestChangeTeamName:450007 | 无显式分支 | 否 |
| 1204 | MSG_USED_TEAM | TeamModel.requestChooseTeam:450021 | 无显式分支 | 否 |
| 1207 | MSG_SAVE_MULTI_TEAM | TeamModel.requestSaveMultiTeam:449978 | 无显式分支 | 否 |
| 1208 | MSG_SET_LEADER_SKILL_GROUP | TeamModel.requestSetLeaderSkillGroup:450038 | 无显式分支 | 否 |
| 1252 | MSG_USE_ITEM | BagModel.requestUseItem:127170 | 9269 | 否 |
| 1254 | MSG_REQ_BAG | BagModel.requestBagData:127193 | 10067 | 是 |
| 1262 | MSG_ENCHANTMENT | BagModel.requestEquipEnchantment:127291 | 无显式分支 | 否 |
| 1263 | MSG_SAVE_ENCHANTMENT | BagModel.requestEquipSaveEnchantment:127302 | 无显式分支 | 否 |
| 1265 | MSG_TRANS_FRAGS_TO_CURRENCY | BagModel.requestTransFrags:127387 | 9321 | 否 |
| 1266 | MSG_COMPOUND_ITEM | BagModel.requestCompoundItem:127394 | 无显式分支 | 否 |
| 1301 | MSG_CHAPTERS | BlockModel.requestBlockData:172604 | 10032 | 是 |
| 1302 | MSG_CHALLENGE_POINT | BlockModel.requestChallengePoint:172749 | 9396 | 否 |
| 1303 | MSG_SWEEP_POINT | BlockModel.requestSweepPoint:172799 | 9026 | 否 |
| 1304 | MSG_GAIN_CHAPTER_REWARD | BlockModel.requestGetStarRewardAll:172846 | 9026 | 否 |
| 1307 | MSG_GET_CHAPTER_AUTO_INFO | BlockModel.requestHangInfo:172858 | 9043 | 是 |
| 1308 | MSG_GAIN_AUTO_CHAPTER_REWARDS | BlockModel.requestHangReward:172864 | 9055 | 否 |
| 1309 | MSG_CHANGE_CHAPTER_AUTO_HERO | BlockModel.requestHangChangeHero:172876 | 9091 | 否 |
| 1310 | MSG_OBTAIN_PASS_REWARDS | BlockModel.requestObtainPassReward:172889 | 9061 | 否 |
| 1311 | MSG_POINT_PRESET_INFO | BlockModel.requsetGetTraingTeam:172898 | 无显式分支 | 否 |
| 1312 | MSG_POINT_TRAIL_HERO | BlockModel.requestLotusBossTrialHeroAttrs:172903 | 无显式分支 | 否 |
| 1351 | MSG_BATTLE_END | ActivityModel.requestBattleFinish:49660<br>ArenaModel.requestBattleFinish:110951<br>BlockModel.requestBattleFinish:172775<br>CardGroupModel.requestBattleFinish:186986<br>FightTogetherModel.requestBattleFinish:261554<br>GuildGveModel.requestBattleFinish:283868<br>HeroListModel.requestBattleFinish:312394<br>ResourcesModel.requestBattleFinish:395887<br>RoguelikeModel.requestBattleFinish:404812 | 9403 | 否 |
| 1354 | MSG_VERIFY_BATTLE | ActivityModel.requestBattleVerify:49646<br>ArenaModel.requestBattleVerify:110938<br>BlockModel.requestBattleVerify:172790<br>CardGroupModel.requestBattleVerify:186972<br>ExploreModel.requestBattleVerify:237144<br>FightTogetherModel.requestBattleVerify:261569<br>GuildGveModel.requestBattleVerify:283855<br>HeroListModel.requestBattleVerify:312409<br>MysteryModel.requestBattleVerify:368525<br>ResourcesModel.requestBattleVerify:395920<br>RoguelikeModel.requestBattleVerify:404828<br>TowerModel.requestBattleVerify:457219 | 9101 | 否 |
| 1361 | MSG_RECORD_CLIENT_LOGIC | BattleModel.sendRecordVerifyFail:151702 | 无显式分支 | 否 |
| 1401 | MSG_REQ_HERO_PLOT | HeroListModel.requestHeroStory:312246 | 无显式分支 | 否 |
| 1402 | MSG_REQ_PLOT_POINT | HeroListModel.requestHeroInterludePoint:312213 | 无显式分支 | 否 |
| 1403 | MSG_REQ_POINT_REWARD | HeroListModel.requestStoryReward:312220 | 无显式分支 | 否 |
| 1404 | MSG_REQ_SYNC_POINT_FINISH | HeroListModel.requestSyncStoryFinish:312233 | 无显式分支 | 否 |
| 1405 | MSG_REQ_UNLOCK_PLOT | HeroListModel.requestUnlockStory:312239 | 无显式分支 | 否 |
| 1406 | MSG_REQ_HERO_PLOTS | HeroListModel.requestHeroStoryList:312261 | 无显式分支 | 否 |
| 1407 | MSG_REQ_PLOT_FORMATION | HeroListModel.requsetPlotFormation:312447 | 无显式分支 | 否 |
| 1408 | MSG_REQ_SWEEP_PLOT_POINT | HeroListModel.requsetPlotPointr:312453 | 无显式分支 | 否 |
| 1452 | MSG_MANOR | ManorModel.requestInitManor:352301 | 无显式分支 | 否 |
| 1453 | MSG_BUILDING | ManorModel.requestConstruct:352321 | 无显式分支 | 否 |
| 1454 | MSG_UPGRATE_BUILDING | ManorModel.requestUpgrade:352327 | 无显式分支 | 否 |
| 1455 | MSG_UPGRATE_BUILDING_TIMEOUT | ManorModel.requestConfirmUpgrade:352333 | 无显式分支 | 否 |
| 1456 | MSG_EDITOR_MANOR | ManorModel.requestEditManor:352308 | 无显式分支 | 否 |
| 1457 | MSG_SAVE_MANOR | ManorModel.requestSaveEdit:352314 | 无显式分支 | 否 |
| 1460 | MSG_COLLECT_RESOURCE | ManorModel.requestCollectResource:352387 | 无显式分支 | 否 |
| 1462 | MSG_PLACE_HERO | ManorModel.requestPlaceHero_old:352469 | 无显式分支 | 否 |
| 1466 | MSG_BUY_DECORATE | ManorModel.requestPurchase_old:352353 | 无显式分支 | 否 |
| 1468 | MSG_RECYCLE_DECORATION | ManorModel.requestRecycle_old:352393 | 无显式分支 | 否 |
| 1469 | MSG_REQ_ROOM | ManorModel.requestBackyardInit:352403 | 无显式分支 | 否 |
| 1470 | MSG_EDITOR_ROOM | ManorModel.requestBackyardEdit_old:352409 | 无显式分支 | 否 |
| 1471 | MSG_SAVE_ROOM_EDITOR | ManorModel.requestBackyardSave_old:352419 | 无显式分支 | 否 |
| 1473 | MSG_ONE_KEY_PUT_DECORATION | ManorModel.requestBackyardDeployAll_old:352436 | 无显式分支 | 否 |
| 1474 | MSG_REPLACE_DECORATION | ManorModel.requestBackyardReplace_old:352445 | 无显式分支 | 否 |
| 1475 | MSG_PUT_ROAD | ManorModel.requestRoadPave:352490 | 无显式分支 | 否 |
| 1476 | MSG_CLEAR_ROAD | ManorModel.requestRoadDemolish:352496 | 无显式分支 | 否 |
| 1478 | MSG_TEMP_SAVE_ROAD | ManorModel.requestRoadSave:352502 | 无显式分支 | 否 |
| 1480 | MSG_MANOR_HOME | ManorModel.requestManorHome:352284 | 无显式分支 | 否 |
| 1481 | MSG_MANOR_DECORATE | ManorModel.requestBackyardSave:352425 | 无显式分支 | 否 |
| 1483 | MSG_MANOR_BUY_DECORATE | ManorModel.requestPurchase:352373 | 无显式分支 | 否 |
| 1484 | MSG_MANOR_ONEKEY_BUY_DECORATE | ManorModel.requestPurchase:352364 | 无显式分支 | 否 |
| 1485 | MSG_MANOR_PUT_HERO | ManorModel.requestPlaceHero:352478 | 无显式分支 | 否 |
| 1486 | MSG_MANOR_SHOP_INFO | ManorModel.requestShopInit_old:352339<br>ManorModel.requestShopInit:352345 | 无显式分支 | 否 |
| 1502 | MSG_REQ_MAILS | MailModel.requestMailList:339741 | 9505 | 是 |
| 1503 | MSG_READ_MAIL | MailModel.readMail:339751 | 无显式分支 | 否 |
| 1504 | MSG_RECEIVE_ATTACHMENT | MailModel.recieveMail:339762 | 无显式分支 | 否 |
| 1505 | MSG_ONEKEY_RECEIVE | MailModel.onekeyRecieve:339784 | 无显式分支 | 否 |
| 1506 | MSG_DEL_READ_MAILS | MailModel.onekeyDeleta:339796 | 无显式分支 | 否 |
| 1552 | MSG_REQ_TASK | TaskModel.requestTask:442825 | 10015 | 是 |
| 1555 | MSG_REQ_TASK_REWARD | TaskModel.requestGetReward:442838 | 10020 | 否 |
| 1556 | MSG_RECEIVE_ACTIVE_REWARD | TaskModel.requestGetActiveReward:442851 | 无显式分支 | 否 |
| 1557 | MSG_ONEKEY_FINISH_TASK | TaskModel.requestOnekeyReward:442863 | 无显式分支 | 否 |
| 1558 | MSG_ACCEPT_TASK | TaskModel.requestAcceptTask:442875 | 无显式分支 | 否 |
| 1559 | MSG_REQ_ONE_KEY_COLLECT_REWARD | FetterModel.requestOneKeyCollectReward:257564 | 9688 | 否 |
| 1601 | MSG_DRAWCARD_SYSTEM | DrawModel.requestDrawData:209047 | 9764 | 是 |
| 1602 | MSG_DRAWCARD | DrawModel.drawCard:209062 | 9770 | 否 |
| 1603 | MSG_CONVERT_COUPON | DrawModel.convertCoupon:209093 | 9767 | 否 |
| 1604 | MSG_CHOOSE_REWARD | DrawModel.selectDrawResult:209105 | 无显式分支 | 否 |
| 1606 | MSG_CHOICE_RESERVE | DrawModel.chosePre:209122 | 无显式分支 | 否 |
| 1607 | MSG_SEE_FRESH | DrawModel.clearFreshPre:209133 | 无显式分支 | 否 |
| 1609 | MSG_REQ_DRAW_CARD_TASK_REWARD | DrawModel.rewardDrawTask:209144 | 无显式分支 | 否 |
| 1610 | MSG_REQ_DRAW_CARD_RECORD | DrawModel.requestDrawCardRecord:209163 | 无显式分支 | 否 |
| 1611 | MSG_REQ_PREPARATION_DRAW_CARD_RECORD | DrawModel.requestPrearationDrawCardRecord:209158 | 无显式分支 | 否 |
| 1651 | MSG_REQ_SHOP | ShopModel.requestShopData:416168<br>ShopModel.refreshShop:416227 | 9571 | 否 |
| 1652 | MSG_REQ_BUY_SHOP_ITEM | ShopModel.bugGoods:416181 | 9158 | 否 |
| 1653 | MSG_REQ_REFLASH_SHOP | ShopModel.handRefreshShop:416216 | 无显式分支 | 否 |
| 1654 | MSG_REQ_RESET_REFLASH | ShopModel.sysHandRefresh:416238 | 无显式分支 | 否 |
| 1656 | MSG_REQ_RESOURCE_SHOP | ExchangeCurrencyModel.requestExchange:216340 | 9576 | 否 |
| 1657 | MSG_REQ_BUY_RESOURCE | ExchangeCurrencyModel.exchangeCurrency:216351 | 9786 | 否 |
| 1658 | MSG_REQ_SURFACE_SHOP | ShopModel.requestShopSkin:416268 | 9586 | 否 |
| 1659 | MSG_REQ_BUY_SURFACE | ShopModel.buyHeroSkin:416281 | 无显式分支 | 否 |
| 1660 | MSG_REQ_SURFACE_PRICE | HeroListModel.requestHeroSkin:312110 | 9154 | 否 |
| 1663 | MSG_REQ_SHOP_MALL | ShopModel.requestMallShop:416301 | 9581 | 否 |
| 1664 | MSG_REQ_MONEY_EXCHANGE | PlayerModel.requestExchange:382449 | 无显式分支 | 否 |
| 1666 | MSG_FAST_BUY_RESOURCE | PlayerModel.requestFastBuyResource:382465 | 无显式分支 | 否 |
| 1667 | MSG_GET_FAST_BUY_RESOURCE_NUM | PlayerModel.requestFastBuyResourceNum:382471 | 无显式分支 | 否 |
| 1668 | MSG_REQ_RECOMMEND_SHOP | ShopModel.requestRecommendShop:416321 | 9591 | 否 |
| 1670 | MSG_REQ_MONTH_CARD | ShopModel.requestMonthCard:416251 | 8973 | 是 |
| 1671 | MSG_GET_PAY_INFO | ShopModel.requestPayInfo:416338 | 9597 | 否 |
| 1672 | MSG_PAY_SUCCESS_CONFIRM | ShopModel.requestPayConfim:416346 | 9620 | 否 |
| 1673 | MSG_PACKAGE_BUY_ITEMS | ShopModel.bugPackageGoods:416204 | 无显式分支 | 否 |
| 1676 | MSG_SELECT_BAG_REWARD | ShopModel.requestSelectReward:416352 | 9243 | 否 |
| 1677 | MSG_DO_PASS_CARD_PAY | ActivityModel.requestDoPassCardPay:49943 | 无显式分支 | 否 |
| 1678 | MSG_REQ_IC_SKIN_EXCHANGE | PlayerModel.requestSkinExchange:382458 | 无显式分支 | 否 |
| 1701 | MSG_REAL_ARENA_BASIC | ArenaRealModel.requestArenaSeasonInfo:116508<br>ArenaRealModel.requestArenaBase:116527 | 10106 | 是 |
| 1702 | MSG_REAL_ARENA_INFO | ArenaRealModel.requestArenaInfo:116535 | 无显式分支 | 是 |
| 1703 | MSG_REAL_ARENA_MATCH | ArenaRealModel.requestArenaMatch:116550 | 无显式分支 | 否 |
| 1704 | MSG_REAL_ARENA_CANCEL_MATCH | ArenaRealModel.requestCancleArenaMatch:116566 | 无显式分支 | 否 |
| 1705 | MSG_REAL_ARENA_DRAW_REWARD | ArenaRealModel.requestArenaBoxReward:116635 | 无显式分支 | 否 |
| 1706 | MSG_REAL_ARENA_RANK | ArenaRealModel.requestArenaRealRank:116614 | 无显式分支 | 否 |
| 1707 | MSG_REAL_ARENA_BATTLE_RECORE | ArenaRealModel.requestArenaRealBattleRecore:116601 | 无显式分支 | 否 |
| 1752 | MSG_REQ_CHECK_BUY_MONTH_CARD | ShopModel.buyMonthCard:416260 | 9008 | 是 |
| 1801 | MSG_ARENA_BATTLE | ArenaModel.requestArenaBattle:110790 | 无显式分支 | 否 |
| 1802 | MSG_ARENA_INFO | ArenaModel.requestArenaInfo:110770 | 10115 | 是 |
| 1803 | MSG_ARENA_REFRESH_OPPONENT | ArenaModel.requestArenaRefresh:110779 | 无显式分支 | 否 |
| 1804 | MSG_ARENA_DRAW_DALIY_REWARD | ArenaModel.requestArenaDrawReward:110884 | 无显式分支 | 否 |
| 1805 | MSG_ARENA_SAVE_DEFEND_FORMATION | ArenaModel.requestSaveDefendTeam:110712<br>ArenaModel.requestArenaSaveFormation:110903<br>TeamModel.requestSaveDefendTeam:449961 | 无显式分支 | 否 |
| 1806 | MSG_ARENA_RANK | ArenaModel.requestArenaRank:110811 | 无显式分支 | 否 |
| 1807 | MSG_ARENA_BATTLE_RECORE | ArenaModel.requestArenaBattleRecore:110798 | 无显式分支 | 否 |
| 1808 | MSG_ARENA_BATTLE_REPLAY | ArenaModel.requestArenaBattleReplay:110895<br>ArenaRealModel.requestArenaBattleReplay:116651<br>PeakvsModel.requestBattleReplay:377040 | 无显式分支 | 否 |
| 1810 | MSG_ARENA_BASIC | ArenaModel.requestArenaBase:110849 | 10115 | 是 |
| 1812 | MSG_REQ_ARENA_TASK_REWARD | ArenaModel.requestTaskReward:110964 | 无显式分支 | 否 |
| 1813 | MSG_CHOOSE_DEFEND_FIGHT_BATTLE | ArenaModel.requestChoseDefendFightBattle:110921 | 无显式分支 | 否 |
| 1814 | MSG_REQ_ARENA_SCORE_REWARD | ArenaModel.requestScoreReward:110972 | 无显式分支 | 否 |
| 1816 | MSG_BATTLE_REPLAY_QUIT | FightTogetherFightModel.battleReplayQuit:260568<br>FriendFightModel.battleReplayQuit:267826 | 无显式分支 | 否 |
| 1817 | MSG_BATTLE_REPLAY_ENTER | FightTogetherFightModel.battleReplayEnter:260563<br>FriendFightModel.battleReplayEnter:267821 | 无显式分支 | 否 |
| 1851 | MSG_WORLD_MAP | ExploreModel.requestExplore:236852 | 9791 | 是 |
| 1852 | MSG_ENTER_SCENE | ExploreModel.requestEnter:236882 | 8900 | 是 |
| 1853 | MSG_SCENE_MOVE | ExploreModel.requestPosition:236910 | 8946 | 是 |
| 1854 | MSG_SCENE_MINI | ExploreModel.requestSceneMini:237195 | 9859 | 是 |
| 1855 | MSG_SCENE_FIGHT | ExploreModel.requestFight:237085 | 9948 | 否 |
| 1857 | MSG_SCENE_FINISH_CASE | ExploreModel.requestEndTrigger:237099 | 9874 | 是 |
| 1860 | MSG_SCENE_UPDATE_RES_NUM | ExploreModel.requestUpdateNum:237150 | 无显式分支 | 否 |
| 1864 | MSG_QUIT_SCENE | ExploreModel.requestQuitScene:237159 | 8957 | 是 |
| 1866 | MSG_SYNC_SCENE_NPC_POS | ExploreModel.requestSyncObjectPos:237169 | 无显式分支 | 否 |
| 1870 | MSG_DROP_BIND_THING | ExploreModel.requestDropThing:237181 | 无显式分支 | 否 |
| 1871 | MSG_SCENE_UNIT_SCAN | ExploreModel.requestSPSearch:237210 | 无显式分支 | 否 |
| 1873 | MSG_GET_PRESETS_BATTLE_TEAM | ExploreModel.requsetGetTraingTeam:237221 | 无显式分支 | 否 |
| 1903 | MSG_REQ_UPDATE_MYSTERY_FORMATION | MysteryModel.requestSaveMysteryBlock:368543 | 无显式分支 | 否 |
| 1906 | MSG_RESET_MYSTERY_POINT | MysteryModel.requestResetMystery:368693 | 无显式分支 | 否 |
| 1951 | MSG_REQ_ACTIVITIES | ActivityModel.requestActivity:49205 | 9775 | 是 |
| 1952 | MSG_REQ_ACTIVITY_REWARD | ActivityModel.requestGetReward:49214 | 无显式分支 | 否 |
| 1955 | MSG_REQ_FACE_NOTICES | ActivityModel.requestFaceNotices:49312 | 9502 | 是 |
| 1956 | MSG_REQ_GET_FREE_STAMINA | ActivityModel.requestFreeStamina:49227 | 无显式分支 | 否 |
| 1957 | MSG_REQ_ACTIVITY_EXCHANGE | ActivityModel.requestExchagne:49237 | 无显式分支 | 否 |
| 1958 | MSG_REQ_CHANGE_EXCHANGE_REMIND | ActivityModel.requestChangeExchange:49246 | 无显式分支 | 否 |
| 1961 | MSG_REQ_QUEST_REWARD | ActivityModel.requestQuestionReward:49257 | 无显式分支 | 否 |
| 1963 | MSG_DRAW_PASS_CARD_DAILY_REWARD | ActivityModel.requesetDrawPassCardDailyReward:49268 | 无显式分支 | 否 |
| 1964 | MSG_DRAW_PASS_CARD_LEVEL_REWARD | ActivityModel.requesetDrawPassCardLevelReward:49277 | 无显式分支 | 否 |
| 1965 | MSG_ONE_KEY_DRAW_PASS_CARD_REWARD | ActivityModel.requestOneLeyDrawPassCardReward:49286 | 无显式分支 | 否 |
| 1966 | MSG_ONE_KEY_FINISH_PASS_CARD_TASK | ActivityModel.requestOneLeyFinishPassCardTask:49295 | 无显式分支 | 否 |
| 1967 | MSG_BUY_PASS_CARD_LEVEL | ActivityModel.requestBuyPassCardLevel:49304 | 无显式分支 | 否 |
| 1968 | MSG_REQ_REWARD_PLAN_REWARDS | ActivityModel.requestGetPlanReward:49322 | 无显式分支 | 否 |
| 1969 | MSG_ACT_CHOOSE_REWARD | ActivityModel.requestChooseReward:49334 | 无显式分支 | 否 |
| 1970 | MSG_HALLOWEEN_MOVE | ActivityModel.requestHalloweenDiceMove:49371 | 无显式分支 | 否 |
| 1972 | MSG_HALLOWEEN_DICE_TRIGGER | ActivityModel.requestHalloweenDiceTrigger:49386 | 无显式分支 | 否 |
| 1973 | MSG_ACTIVITY_FIGHT | ActivityModel.requestActivityFight:49625 | 无显式分支 | 否 |
| 1974 | MSG_HALLOWEEN_TOWER_REWARD | ActivityModel.requestHalloweenTowerReward:49670 | 无显式分支 | 否 |
| 1975 | MSG_MARKET_LEVEL_UP | ActivityModel.requestSwimMarketLevelUp:49691 | 无显式分支 | 否 |
| 1976 | MSG_MARKET_RECEIVE | ActivityModel.requestSwimMarketReceive:49703 | 无显式分支 | 否 |
| 1977 | MSG_RECEIVE_SWIMSUIT | ActivityModel.requestSwimMainReceive:49715 | 无显式分支 | 否 |
| 1979 | MSG_OBTAIN_CG_REWARD | ActivityModel.requestCGReward:49727 | 无显式分支 | 否 |
| 1980 | MSG_ACTIVITY_STORY_REWARD | ActivityModel.requestActivityStoryReward:49739 | 无显式分支 | 否 |
| 1981 | MSG_ONE_KEY_FINISH_ACTIVITY_TASK | ActivityModel.requestOneKeyFinishActivityTaskReward:49756 | 无显式分支 | 否 |
| 1982 | MSG_SAVE_BOSS_CHALLENGE_BUFFS | ActivityModel.requestSaveBossChallengeBuff:49773 | 无显式分支 | 否 |
| 1983 | MSG_LOTTERY | ActivityModel.requestLottery:49782<br>ActivityModel.requestArchaeology:49791 | 9756 | 否 |
| 1984 | MSG_RESET_LOTTERY | ActivityModel.requestResetLottery:49800 | 9756 | 否 |
| 1985 | MSG_GLOBAL_WISH | ActivityModel.requestGlobalWish:49809 | 9756 | 否 |
| 1986 | MSG_LOTTERY_REWARD | ActivityModel.requestLotteryReward:49818 | 9756 | 否 |
| 1988 | MSG_REQ_OPEN_ACT_RED_GIFT | ActivityModel.requestOpenCodePackage:49853 | 无显式分支 | 否 |
| 1989 | MSG_REQ_GRAB_RED_GIFT | ActivityModel.requestGrabRedPackage:49889 | 无显式分支 | 否 |
| 1990 | MSG_REQ_ACT_SEND_RED_GIFT | ActivityModel.sendRedPackage:49863 | 无显式分支 | 否 |
| 1991 | MSG_DICE_SOLUTION | ActivityModel.requestDiceSolution:49830 | 9756 | 否 |
| 1992 | MSG_REQ_ACTIVITY_GIFT_SEND | ActivityModel.requestSendGift:49898 | 无显式分支 | 否 |
| 1993 | MSG_REQ_ACTIVITY_TURNTABLE | ActivityModel.requestRoulette:49916 | 无显式分支 | 否 |
| 1994 | MSG_SAVE_CREATE_ROLE_BOSS | ActivityModel.requestSaveCreateRoleBoss:49907 | 无显式分支 | 否 |
| 1995 | MSG_PASS_CARD_ITEM_TRANS_TO_EXP | ActivityModel.requestPassCardItemTransToExp:49925 | 无显式分支 | 否 |
| 1996 | MSG_EXCHANGE_PASS_CARD_ITEM | ActivityModel.requestExchangePassCardItem:49934 | 无显式分支 | 否 |
| 1998 | MSG_ACTIVITY_RUNE_NEXT_STEP | ActivityModel.requestRuneNextStep:49953 | 无显式分支 | 否 |
| 1998 | MSG_SWEEP_ACTIVITY_FIGHT | ActivityModel.requestSweepActivityFight:49971 | 无显式分支 | 否 |
| 1999 | MSG_RANDOM_ACTIVITY_RUNE_ENTRIES | ActivityModel.requestRuneEntries:49962 | 无显式分支 | 否 |
| 2001 | MSG_PVP_INVITE | FriendFightModel.sendFriendFightInvite:267634 | 无显式分支 | 否 |
| 2003 | MSG_REPLY_PVP_INVITE | FightTogetherFightModel.replyFriendFightInvite:260318<br>FriendFightModel.replyFriendFightInvite:267650 | 无显式分支 | 否 |
| 2005 | MSG_PVP_ROOM_READY | FightTogetherFightModel.readyFriendFight:260373<br>FriendFightModel.readyFriendFight:267664 | 无显式分支 | 否 |
| 2006 | MSG_LEAVE_PVP_ROOM | FightTogetherFightModel.leaveRoom:260382<br>FriendFightModel.leaveRoom:267673 | 无显式分支 | 否 |
| 2007 | MSG_PVP_ROOM_FIGHT | FightTogetherFightModel.starFight:260410<br>FriendFightModel.starFight:267681 | 无显式分支 | 否 |
| 2008 | MSG_GET_PVP_FIGHT_INFO | ArenaRealModel.requestFightInfo:116577<br>FightTogetherFightModel.requestFightInfo:260392<br>FriendFightModel.requestFightInfo:267779 | 无显式分支 | 否 |
| 2009 | MSG_PVP_LOAD_COMPLETE | PvpModel.sendLoadComplete:386166 | 无显式分支 | 否 |
| 2012 | MSG_PVP_OPERATE_ORDER | PvpModel.sendOp:386171 | 无显式分支 | 否 |
| 2013 | MSG_PVP_FINISHED | ArenaRealModel.requestBattleVerify:116681<br>FightTogetherFightModel.requestBattleVerify:260513<br>FriendFightModel.requestBattleVerify:267804 | 无显式分支 | 否 |
| 2014 | MSG_PVP_GET_FRAMES | PvpModel.sendGetFrame:386193 | 无显式分支 | 否 |
| 2016 | MSG_GET_PVP_ROOM | PvpFightModel.requestRoomInfo:386091 | 9023 | 是 |
| 2017 | MSG_GET_PVP_BATTLE_RESULT | PvpModel.sendGetResult:386206 | 无显式分支 | 否 |
| 2051 | MSG_WEAR_EQUIP_NEW | BagModel.requestEquipHandler:127200 | 9122 | 否 |
| 2052 | MSG_EQUIP_UPGRADE_LEVEL_NEW | BagModel.requestEquipUpgrade:127234 | 9122 | 否 |
| 2053 | MSG_EQUIP_UPGRADE_QUALITY_NEW | BagModel.requestEquipQualitygrade:127255 | 9122 | 否 |
| 2054 | MSG_EQUIP_UPGRADE_STAR_NEW | BagModel.requestEquipStargrade:127275 | 9122 | 否 |
| 2055 | MSG_LOCK_EQUIP_NEW | BagModel.requestLockEquip:127215 | 9122 | 否 |
| 2056 | MSG_WEAR_RUNE | BagModel.requestMsgWearRune:127317 | 无显式分支 | 否 |
| 2057 | MSG_ONE_KEY_TAKE_OFF_RUNE | BagModel.requestOneKeyTaseOffRune:127333 | 无显式分支 | 否 |
| 2058 | MSG_RUNE_UPGRADE_LEVEL | BagModel.requestRuneStrength:127349 | 无显式分支 | 否 |
| 2059 | MSG_LOCK_RUNE | BagModel.requestLockRune:127372 | 无显式分支 | 否 |
| 2060 | MSG_TAKE_OFF_EQUIP | BagModel.requestTakeOffEquip:127416 | 无显式分支 | 否 |
| 2061 | MSG_DECOMPOSE_EQUIP | BagModel.requestDecomposeEquip:127400 | 无显式分支 | 否 |
| 2152 | MSG_REQ_MOUNT_ON | MountModel.requestMountOn:364670 | 无显式分支 | 否 |
| 2153 | MSG_REQ_MOUNT_OFF | MountModel.requestMountOff:364692 | 无显式分支 | 否 |
| 2251 | MSG_REQ_RANK_LIST | RankModel.requestRankList:387483 | 无显式分支 | 否 |
| 2252 | MSG_REQ_MILESTONES_INFO | RankModel.requestMilestonesList:387502 | 无显式分支 | 否 |
| 2253 | MSG_REQ_MILESTONES_REWARD | RankModel.requestMilestonesReward:387509 | 无显式分支 | 否 |
| 2301 | MSG_REQ_COLLECT_LIST | MonsterModel.requestEquipAndMonster:364136 | 9265 | 否 |
| 2401 | MSG_GET_SUMMIT_BASE_INFO | PeakvsModel.requestPeakvsBaseInfo:376817 | 9833 | 是 |
| 2402 | MSG_SUMMIT_COURSE_INFO | PeakvsModel.requestPeakvsCourseInfo:376826 | 无显式分支 | 否 |
| 2403 | MSG_SUMMIT_APPLY | PeakvsModel.requestPeakvsApply:376873 | 无显式分支 | 否 |
| 2404 | MSG_GET_SUMMIT_TEAMS | PeakvsModel.requestPeakvsTeams:376883 | 无显式分支 | 否 |
| 2405 | MSG_EDIT_TEAM | PeakvsModel.requestPeakvsSaveFormation:376921 | 无显式分支 | 否 |
| 2406 | MSG_SWAP_TEAM_INDEX | PeakvsModel.requestSwapTeam:376950 | 无显式分支 | 否 |
| 2407 | MSG_SWAP_TEAM_LEADER | PeakvsModel.requestSwapTeamLeader:376978 | 无显式分支 | 否 |
| 2408 | MSG_PLAYER_BOUT_INFO | PeakvsModel.requestPeakvsBoutInfo:376846 | 无显式分支 | 否 |
| 2410 | MSG_SUMMIT_COURSE_GROUP_INFO | PeakvsModel.requestPeakvsGroupInfo:376837 | 无显式分支 | 否 |
| 2411 | MSG_GET_ALL_SUMMIT_BATTLE_RECORD | PeakvsModel.requestPeakvsAllRecord:376855 | 无显式分支 | 否 |
| 2412 | MSG_GET_TARGET_BATTLE_RECORD | PeakvsModel.requestPeakvsTargetRecord:376867 | 无显式分支 | 否 |
| 2415 | MSG_QUIT_SUMMIT | PeakvsModel.requestQuitPeakvs:377004 | 无显式分支 | 否 |
| 2421 | MSG_GET_SUMMIT_BATTLE | PeakvsModel.requestSummitBattle:377009 | 无显式分支 | 否 |
| 2502 | MSG_GET_TRAINING_ALL_INFO | CardGroupModel.requsetGetTrainingAllInfo:186792 | 无显式分支 | 否 |
| 2504 | MSG_ENTER_SPEICAL | CardGroupModel.requsetEnterSpecial:186801 | 无显式分支 | 否 |
| 2505 | MSG_EDIT_TRAINING_TEAM | CardGroupModel.requsetEditTraingTeam:186786 | 无显式分支 | 否 |
| 2506 | MSG_GET_TRAINING_TEAM | CardGroupModel.requsetGetTraingTeam:186781 | 无显式分支 | 否 |
| 2507 | MSG_TRAINING_START_FIGHT | CardGroupModel.requestTrainingStartFight:186951 | 无显式分支 | 否 |
| 2508 | MSG_TRAINING_PICKED_BOX | CardGroupModel.requsetReceiveStarBox:186812 | 无显式分支 | 否 |
| 2551 | MSG_REQUEST_ROGUE_INFO | RoguelikeModel.requestRogue:404620 | 无显式分支 | 是 |
| 2552 | MSG_ENTER_ROUGE | RoguelikeModel.requestEnter:404629 | 无显式分支 | 否 |
| 2553 | MSG_CHOOSE_ROGUE_NODE | RoguelikeModel.requestChoosePoint:404644 | 无显式分支 | 否 |
| 2554 | MSG_CHOOSE_ROGUE_SKILL | RoguelikeModel.requestChooseSkill:404654 | 无显式分支 | 否 |
| 2555 | MSG_CHOOSE_ROGUE_WISH | RoguelikeModel.requestChooseWish:404664 | 无显式分支 | 否 |
| 2556 | MSG_CHOOSE_ROGUE_EVENT | RoguelikeModel.requestChooseEvent:404674 | 无显式分支 | 否 |
| 2557 | MSG_REFUSE_ROGUE_REWARD | RoguelikeModel.requestChooseQuit:404684 | 无显式分支 | 否 |
| 2558 | MSG_ROGUE_SHOP_BUY | RoguelikeModel.requestChooseShopItem:404694 | 无显式分支 | 否 |
| 2559 | MSG_LEAVE_ROGUE_SHOP | RoguelikeModel.requestChooseShopQuit:404704 | 无显式分支 | 否 |
| 2560 | MSG_REFRESH_ROGUE_REWARD | RoguelikeModel.requestRefreshReward:404714 | 无显式分支 | 否 |
| 2561 | MSG_REFRESH_ROGUE_SHOP | RoguelikeModel.requestRefreshShop:404724 | 无显式分支 | 否 |
| 2563 | MSG_ROGUE_SETTLE | RoguelikeModel.requestFinish:404754 | 无显式分支 | 否 |
| 2564 | MSG_REENTER_ROGUE | RoguelikeModel.requestReenter:404765 | 无显式分支 | 否 |
| 2565 | MSG_RESET_ROGUE_SCORE | RoguelikeModel.requestResetScore:404844 | 无显式分支 | 否 |
| 2566 | MSG_DRAW_ROGUE_SCORE_REWARD | RoguelikeModel.requestKeyReward:404834 | 无显式分支 | 否 |
| 2567 | MSG_ACTIVE_TALENT | RoguelikeModel.requestActiveTalent:404855 | 无显式分支 | 否 |
| 2568 | MSG_REQ_BATTLE_INFO | RoguelikeModel.requestChallengePoint:404779 | 无显式分支 | 否 |
| 2569 | MSG_GIVE_UP_ROGUE_FIGHT | RoguelikeModel.requestBattleRestart:404795 | 无显式分支 | 否 |
| 2571 | MSG_ROGUE_EVENT_TROPHY_CHOOSE | RoguelikeModel.requestReophyChose:404734 | 无显式分支 | 否 |
| 2572 | MSG_ROGUE_TROPHY_STRENGTH | RoguelikeModel.requestTrophyStrength:404744 | 无显式分支 | 否 |
| 2573 | MSG_ROGUE_PICTORIAL_PAGE | RoguelikeModel.requestPictorialPage:404872 | 无显式分支 | 否 |
| 2574 | MSG_ROGUE_RECEIVE_PICTORIAL_REWARD | RoguelikeModel.requestReceivePictorialReward:404881 | 无显式分支 | 否 |
| 2575 | MSG_ROGUE_RECEIVE_ALL_PICTORIAL_REWARD | RoguelikeModel.requestReceiveAllPictorialReward:404892 | 无显式分支 | 否 |
| 2601 | MSG_GET_FT_INFO | FightTogetherModel.requestGetFTInfo:261413 | 无显式分支 | 是 |
| 2604 | MSG_BOSS_SINGLE_START_FIGHT | FightTogetherModel.requestChallengePoint:261425 | 无显式分支 | 否 |
| 2606 | MSG_TOGETHER_CREATE_ROOM | FightTogetherFightModel.createTogetherRoom:260344 | 无显式分支 | 否 |
| 2607 | MSG_TOGETHER_PVP_INVITE | FightTogetherFightModel.sendTogetherFightInvite:260337 | 无显式分支 | 否 |
| 2608 | MSG_TOGETHER_MODIFY_ROOM | FightTogetherFightModel.modifyTogetherRoom:260364 | 无显式分支 | 否 |
| 2609 | MSG_TOGETHER_JOIN_ROOM | FightTogetherFightModel.joinTogetherRoom:260354 | 无显式分支 | 否 |
| 2610 | MSG_ENTER_RECOMMEND_BOSS | FightTogetherModel.requestRoomList:261578 | 无显式分支 | 否 |
| 2611 | MSG_SWITCH_RECOMMEND_BOSS | FightTogetherModel.requestRoomRefresh:261610 | 无显式分支 | 否 |
| 2614 | MSG_FT_PVP_ESCAPE | PvpModel.sendRunAway:386179 | 无显式分支 | 否 |
| 2615 | MSG_FT_RECEIVE_TAOFA_REWARD | FightTogetherModel.requestReceiveReward:261627 | 无显式分支 | 否 |
| 2616 | MSG_FT_ASSIST_ROBOT_SET | FightTogetherFightModel.selectRobot:260580 | 无显式分支 | 否 |
| 10101 | MSG_GET_FRIENDS | FriendFightModel.requestFriendList:267763<br>FriendModel.requestFriendList:270699 | 10097 | 是 |
| 10102 | MSG_GET_RECOMMEND_FRIENDS | FriendModel.requestRecommendList:270715 | 9700 | 是 |
| 10103 | MSG_ADD_FRIEND | FriendModel.addFriend:270889 | 无显式分支 | 否 |
| 10104 | MSG_ONEKEY_ADD_FRIEND | FriendModel.addOneKeyFriend:270787 | 无显式分支 | 否 |
| 10105 | MSG_AUDIT_FRIEND_LIST | FriendModel.requestApplyList:270706 | 无显式分支 | 否 |
| 10106 | MSG_AUDIT_FRIEND | FriendModel.auditFirend:270743 | 无显式分支 | 否 |
| 10107 | MSG_ONEKEY_AUDIT_FRIENDS | FriendModel.auditOneKeyApply:270757 | 无显式分支 | 否 |
| 10108 | MSG_DELETE_FRIEND | FriendModel.deletaFriend:270902 | 无显式分支 | 否 |
| 10109 | MSG_SEARCH_FRIEND | FriendModel.requestSearchList:270725 | 无显式分支 | 否 |
| 10110 | MSG_SET_REMARK | FriendModel.setRemark:270778 | 无显式分支 | 否 |
| 10111 | MSG_SEND_CHAT_MSG | ChatModel.sendChatContent:191342<br>FriendModel.sendChatContent:270825 | 无显式分支 | 否 |
| 10113 | MSG_GET_OTHERS_INFO | FriendModel.getPlayerinfo:270864 | 无显式分支 | 否 |
| 10114 | MSG_GET_CHAT_HISTORY_LIST | FriendModel.lookChatHistory:270836<br>FriendModel.lookPrivateChatHistory:270847 | 无显式分支 | 否 |
| 10115 | MSG_GET_CHAT_INFO | ChatModel.requestWorldChannel:191353 | 8961 | 是 |
| 10116 | MSG_CHOOSE_WORLD_CHANNEL_LINE | ChatModel.changeWorldChannel:191364 | 无显式分支 | 否 |
| 10120 | MSG_GET_FRIEND_ROLE_INFO | FriendModel.getFriendRoleInfo:270881<br>PlayerModel.getFriendRoleInfo:382783 | 无显式分支 | 否 |
| 10121 | MSG_GET_FRIEND_HERO_INFO | FriendModel.getFriendHeroInfo:270923 | 无显式分支 | 否 |
| 10122 | MSG_SHARE_BATTLE_RECORD | ChatModel.shareBattle:191386 | 无显式分支 | 否 |
| 10124 | MSG_REQ_BLACKLIST | FriendModel.requestBlackList:270853 | 8970 | 是 |
| 10125 | MSG_ADD_BLACKLIST | FriendModel.addBlackList:270937 | 无显式分支 | 否 |
| 10126 | MSG_DELETE_BLACKLIST | FriendModel.removeBlackList:270962 | 无显式分支 | 否 |
| 10127 | MSG_REQ_REPORT | FriendModel.reportPlayer:270928 | 无显式分支 | 否 |
| 10128 | MSG_SYNC_CHAT_READ_LIST | FriendModel.syncChatRead:270815 | 无显式分支 | 否 |
| 10129 | MSG_GET_CHAT_LIST | FriendModel.requestPrivateFriendList:270733 | 8967 | 是 |
| 10131 | MSG_CLAEAR_CHAT_LIST | FriendModel.clearChatList:270802 | 无显式分支 | 否 |
| 10133 | MSG_SHARE_ROOM | FightTogetherFightModel.shareRoom:260573 | 无显式分支 | 否 |
| 10151 | MSG_GET_CLUB | GuildModel.refreshList:288979 | 10000 | 是 |
| 10152 | MSG_GET_RECOMMEND_CLUBS | GuildApplyListModel.refreshList:280634 | 10000 | 是 |
| 10153 | MSG_CREATE_CLUB | GuildModel.createGuild:288991 | 无显式分支 | 否 |
| 10154 | MSG_GET_CLUB_MEMBERS | FriendFightModel.requestMemberList:267772<br>GuildModel.requestMemberList:289092 | 无显式分支 | 否 |
| 10155 | MSG_APPLY_CLUB | GuildApplyListModel.applyGuild:280701 | 无显式分支 | 否 |
| 10156 | MSG_INVITE_JOIN_CLUB | FriendModel.inviteGuild:270914 | 无显式分支 | 否 |
| 10157 | MSG_GET_CLUB_INVITES | GuildApplyListModel.requestInviteMsg:280678 | 10000 | 是 |
| 10158 | MSG_REPLY_CLUB_INVITE | GuildApplyListModel.replyInviteMsg:280687 | 无显式分支 | 否 |
| 10159 | MSG_CLUB_AUDIT_LIST | GuildModel.requestAduitApply:289046 | 无显式分支 | 否 |
| 10160 | MSG_CLUB_AUDIT | GuildModel.aduitApply:289008 | 无显式分支 | 否 |
| 10161 | MSG_CLUB_SETTING | GuildModel.changeGuildSet:289053 | 无显式分支 | 否 |
| 10162 | MSG_UPDATE_CLUB_NAME | GuildModel.requestUpdateName:289067 | 无显式分支 | 否 |
| 10163 | MSG_CLUB_POST_SETTING | GuildModel.updateGuildPost:289106 | 无显式分支 | 否 |
| 10164 | MSG_CLUB_DEL_MEMBER | GuildModel.leaveGuild:289121 | 无显式分支 | 否 |
| 10165 | MSG_QUIT_CLUB | GuildModel.exitGuild:289079 | 无显式分支 | 否 |
| 10166 | MSG_CLUB_LOG | GuildModel.requestLogList:289155 | 无显式分支 | 否 |
| 10167 | MSG_SEARCH_CLUB | GuildApplyListModel.requestSearchList:280646 | 10000 | 是 |
| 10168 | MSG_CLUB_ONKEY_APPLY | GuildApplyListModel.keyApply:280670 | 无显式分支 | 否 |
| 10171 | MSG_CLUB_SETTING_WECLOME | GuildModel.setGuildWelcome:289138 | 无显式分支 | 否 |
| 10172 | MSG_CLUB_AUDIT_ALL | GuildModel.aduitKeyApply:289026 | 无显式分支 | 否 |
| 10174 | MSG_CLUB_INFO | GuildApplyListModel.requestClubInfo:280717 | 无显式分支 | 否 |
| 10175 | MSG_LAST_RECOMMEND_CLUBS_REQ | GuildApplyListModel.resetGuildList:280660 | 10000 | 是 |
| 10176 | MSG_CLUB_SIGNIN | GuildModel.requestSign:288971 | 无显式分支 | 否 |
| 10177 | MSG_REQ_CLUB_RED_GIFTS | ActivityModel.requestRedGifts:49838 | 无显式分支 | 否 |
| 10178 | MSG_SEND_CLUB_MAIL | GuildModel.sendGuildMail:289167 | 无显式分支 | 否 |
| 未知 | MSG_ENTER_EXPLORE_SCENE | MysteryModel.requestEntetMystery:368470 | 无显式分支 | 否 |
| 未知 | MSG_EXPLORE_FINISH_CASE | MysteryModel.requestEndTrigger:368514 | 无显式分支 | 否 |
| 未知 | MSG_EXPLORE_MOVE | MysteryModel.requestMove:368489 | 无显式分支 | 否 |
| 未知 | MSG_EXPLORE_SCENE_FIGHT | MysteryModel.requestFight:368682 | 无显式分支 | 否 |
| 未知 | MSG_EXPLORE_SPEEDUP | MysteryModel.requestChangeMoveSpeed:368707 | 无显式分支 | 否 |
| 未知 | MSG_EXPLORE_STOP_MOVE | MysteryModel.requestStand:368497 | 无显式分支 | 否 |
| 未知 | MSG_EXPLORE_TRIGGER_CASE | MysteryModel.requestStartTrigger:368504 | 无显式分支 | 否 |
| 未知 | MSG_QUIT_EXPLORE | MysteryModel.requestFinishExplore:368480 | 无显式分支 | 否 |
| 未知 | MSG_REQ_EDIT_AUTO_FIGHT_SETTING | TeamModel.requestSaveTeamAutoFightSetting:449994 | 无显式分支 | 否 |
| 未知 | MSG_REQ_MOUNT_INFO | MountModel.requestMount:364658 | 无显式分支 | 否 |

## 动态调用及其他 request

下列位置未从第二参数取到字面量消息名，必须人工追踪，不能推断已经覆盖

- `ActivityModel.requesSpringStoryPointEnd:49682`
- `ActivityModel.requestSelectFightMap:49984`
- `ActivityModel.requestResetToPoint:49993`
- `ActivityModel.requestResetRound:50002`
- `ActivityModel.requestLotusBossTrialHeroAttrs:50010`
- `ActivityModel.rewardStory:50016`
- `ActivityModel.requestRunResult:50030`
- `ActivityModel.redBlueVote:50054`
- `ActivityModel.redGuessSelect:50068`
- `ActivityModel.redGuessRecords:50076`
- `ActivityModel.redViewGuessBattles:50082`
- `ActivityModel.guessLastPopUp:50088`
- `FairPlayModel.requestArenaInfo:254874`
- `FairPlayModel.requestFairPlayCard:254887`
- `FairPlayModel.selectCardHero:254893`
- `FairPlayModel.clearLostTime:254903`
- `FairPlayModel.requestArenaMatch:254917`
- `FairPlayModel.requestCancleArenaMatch:254933`
- `FairPlayModel.requestFightInfo:254944`
- `FairPlayModel.requestArenaRealBattleRecore:254966`
- `FairPlayModel.requestArenaBoxReward:254978`
- `FairPlayModel.requestArenaBattleReplay:254991`
- `FairPlayModel.requestBattleVerify:255020`
- `GiftPushModel._requestMsg:276674`
- `GuildGveModel._requestMsg:284004`
- `GuildModel.lookClubChatHistory:288872`
- `Model.request:361692`
- `MysteryModel.requestMysteryInfo:368459`
- `NetManager.request:370225`
- `NetNode.requestUnique:370674`
- `NetNode.requestHeartBeat:370749`
- `NetNode.requestResumeResponse:370763`
- `SetModel.requestHeadList:412676`
- `TowerModel._requestTowerMsg:456961`

## Go 已受理但本扫描未找到字面量调用

常见原因是协议框架直接发包或动态命令名；不能据此删除实现

- `901 MSG_HEARTBEAT`
- `1863 MSG_SCENE_SPEEDUP`
- `2101 MSG_REQ_TOWER_INFO`
- `2351 MSG_FAIR_ARENA_INFO`
