# Python 项目迁移分析

状态：第一阶段完成，尚未开始大规模 Go 迁移。

## 1. 分析边界与证据

本报告以发布包中的代码为当前基线：

- `../gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_server.py`
- `../gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_protocol.py`
- `../gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/read_config_table.py`
- `../gxjy-release-20261019/gxjy-release-20261019/2-SERVER/gxjy_decrypt.py`
- `../gxjy-release-20261019/gxjy-release-20261019/3-CLIENT-SRC/rebuild_apk.py`
- `../gxjy-release-20261019/gxjy-release-20261019/3-CLIENT-SRC/patch_cvguard.py`

同时阅读了发布说明、协议规格、最新交接文档、战斗交接文档和用户需求汇总。文档只作为证据索引。文档与代码冲突时，以当前代码为准。

当前仓库在 `main` 分支。Git 只跟踪根 `README.md`。`.kiro/`、压缩包和整个发布目录均未跟踪。因此，后续不能直接执行 `git add .`。大型 APK、原始资源、缓存、私钥、签名文件和存档必须排除。

当前机器有 Python 3.15.1。`py -3` 可用。`python` 命令异常退出。Go 1.27.3 位于 `C:\Program Files\Go\bin\go.exe`，但未加入 PATH。

## 2. 目录与资产分类

发布目录约有 32,531 个文件。

| 目录或文件 | 文件数 | 大小 | 分类 | 迁移处理 |
|---|---:|---:|---|---|
| `1-APK/` | 1 | 966,919,714 B | 外部二进制 | 不迁入 Go 源码 |
| `2-SERVER/` | 32,503 | 31,372,982 B | Python 服务端与数据 | 主要迁移源 |
| `3-CLIENT-SRC/` | 7 | 11,586,669 B | 客户端补丁工具 | 默认不迁移业务，仅保留兼容工具 |
| `4-DOCS/` | 12 | 142,761 B | 研究文档 | 证据来源，需标记新旧 |
| `MANIFEST.sha256` | 1 | 5,029,506 B | 完整性清单 | 保留在发布包外 |
| `save/` | 1 | 121 B | 运行时目录 | Go 版使用独立数据目录 |
| 根批处理 | 2 | 2,401 B | Windows 启动工具 | 后期替换或兼容 |

按后缀统计：30,358 个 JSON、2,138 个 GZip 缓存、14 个 Markdown、6 个 Python、1 个反编译 JS、1 个 APK、2 个 PEM、1 个 keystore。

### 2.1 源码

- `gxjy_server.py`：10,226 行，191 个顶层函数，1 个 `Session` 类，9 个方法。
- `gxjy_protocol.py`：443 行，协议、MessagePack、ChaCha20-Poly1305 和自测。
- `read_config_table.py`：262 行，Cocos 配置表定位、解密、合并和缓存。
- `gxjy_decrypt.py`：101 行，REF 容器和 XXTEA 解密。
- `rebuild_apk.py`：261 行，客户端补丁、JSC 打包、APK 签名和安装。
- `patch_cvguard.py`：184 行，JSC 的 XXTEA/GZip 读写与旧补丁工具。

### 2.2 数据与生成物

- `2-SERVER/apk/assets/assets/`：原始 Cocos 配置资源，运行时必需。
- `2-SERVER/protocol/table_cache/`：2,138 个已解密 GZip 缓存，可再生成。
- `protocol_map.json`：消息名到编号映射。
- `SERVER_Protocol.json`：协议数据表快照。
- `mall_info_payload.json`：商店回包数据。
- `index.probe13.dec.js`：358,055 行反编译客户端基线。
- APK、JSC、未签名 APK：构建产物，不属于 Go 服务端源码。

### 2.3 运行时文件

- `save_localplayer.json`：单玩家状态。
- `save_localplayer.prev.json`：前一次状态。
- `save_backup_*.json`：最多十份差异备份。
- `server_live.log`：追加日志。

这些文件不能进入版本库。

### 2.4 敏感资产

当前发布包含网关 RSA 私钥、回退私钥参数、APK keystore 和明文口令。Go 版不得把它们编译进源码，也不得自动提交。运行时应通过配置路径加载。测试只能使用专用测试密钥。

## 3. 外部依赖

### 3.1 Python 服务端

| 依赖 | 用途 | 当前状态 |
|---|---|---|
| Python 3.10+ | 服务端运行 | 3.15.1 可用 |
| `cryptography>=41` | PEM 私钥和 RSA PKCS#1 v1.5 解密 | 当前环境未安装 |
| Python 标准库 | HTTP、TCP、线程、JSON、压缩、随机数 | 已使用 |

`cryptography` 缺失时，代码会使用 `server_rsa.json` 中的 `N/D/E` 做手写模幂解密。该回退路径已能加载入口，但本次没有启动完整服务。

### 3.2 客户端构建

| 工具 | 用途 | 必需性 |
|---|---|---|
| Node.js | `node --check` 语法检查 | 可选 |
| JDK `jarsigner` | APK 签名 | 构建 APK 必需 |
| ADB | 安装、反向端口、拉起游戏 | 实机验证必需 |
| Android 模拟器 | 客户端验收 | 集成验证必需 |

### 3.3 Go 依赖建议

HTTP、JSON、压缩和加密优先使用标准库。WebSocket 使用 `github.com/gorilla/websocket`，UUID 使用 `github.com/google/uuid`。

MessagePack 建议先实现当前 Python 支持的最小子集。这样能复制整数宽度和错误行为，并避免引入大库。

ChaCha20-Poly1305 可选 `golang.org/x/crypto/chacha20poly1305`。引入前必须固定版本，并用 RFC 向量和 Python 差分测试确认。若其封装妨碍精确帧测试，再迁移当前纯实现。

WebSocket 使用成熟库承载握手、分片、关闭帧和错误处理。游戏二进制协议仍由差分测试固定。

## 4. 当前执行链

```text
HTTP 登录请求
  -> make_login_response
  -> 返回资源、区服、角色与网关地址

WebSocket Upgrade
  -> ws_handshake
  -> Session.run
  -> 首包识别 official/plain
  -> RSA 恢复会话密钥
  -> cc2p 解密
  -> MessagePack 解码
  -> Session.dispatch
  -> 游戏逻辑修改全局玩家状态
  -> MessagePack 编码
  -> cc2p 加密
  -> WebSocket 回包
```

启动顺序如下：

1. 解析命令行参数。
2. `load_player()` 加载单玩家存档。
3. `warm_tables()` 预热重表。
4. 发放起始装备。
5. 后台线程启动 HTTP 服务。
6. 主线程运行 WebSocket 服务。

此顺序是兼容行为。Go 版不能在配置未就绪时开始接收游戏请求。

## 5. 模块依赖关系

```text
cmd/server
  -> config/runtime
  -> config/table
       -> asset/ref
       -> crypto/xxtea
       -> gzip/json/cache
  -> storage/player
  -> transport/http
  -> transport/websocket
  -> protocol/frame
       -> protocol/varint
       -> protocol/msgpack
       -> crypto/cc2p
       -> crypto/rsa
  -> session
  -> dispatcher
       -> game/player
       -> game/hero
       -> game/team
       -> game/reward
       -> game/battle
       -> game/equipment
       -> game/inventory
       -> game/gacha
       -> game/task
       -> game/explore
       -> game/shop
```

业务模块依赖配置表和玩家状态。协议模块不应依赖业务模块。存储模块不应依赖网络模块。

## 6. 功能清单

“理解度”表示本轮基于代码和现有证据的理解，不代表已与真实客户端验证。

| 功能 | Python 文件与函数 | 输入 | 输出/副作用 | 主要依赖 | 二进制 | 难点 | 理解度 | 计划 Go 位置 | 验证方式 |
|---|---|---|---|---|---|---|---|---|---|
| 日志 | `gxjy_server.py:80 log` | 任意消息 | 控制台与追加文件 | 时间、文件 | 否 | 并发顺序 | 高 | `internal/observability` | 捕获日志测试 |
| HTTP 登录 | `:122 make_login_response` | 路径、请求体、WS 端口 | JSON 登录数据 | 时间、主机配置 | 否 | 缺字段与空值 | 高 | `internal/login` | Golden JSON |
| HTTP 服务 | `:232 http_server`, `:243 _handle_http` | TCP 请求 | HTTP/1.1 响应 | socket | 是 | 手写解析语义 | 高 | `internal/transport/httpserver` | 原始 TCP 测试 |
| HTTP GM | `:278 /gm`, `:8393 gm_exec` | 查询参数 | JSON 与玩家变更 | 全局玩家状态 | 否 | 当前无认证 | 高 | `internal/admin` | 临时存档集成测试 |
| WS 握手 | `:325 ws_handshake` | HTTP Upgrade | 101 响应 | SHA-1、Base64 | 是 | 超时与头解析 | 高 | `internal/transport/ws` | 握手字节测试 |
| WS 帧 | `:362 ws_recv`, `:390 ws_send` | 网络帧 | payload/网络写入 | socket | 是 | 掩码、长度、关闭帧 | 高 | `internal/transport/ws` | 帧向量测试 |
| Plain 帧 | `:406 build_plain_frame`, `:443 looks_like_plain`, `:455 parse_plain_frame` | cmd、对象、字节 | 帧或解析值 | MessagePack | 是 | 官方/社区双协议 | 高 | `internal/protocol/plain` | Python 差分 |
| Varint | `gxjy_protocol.py:14-35` | 非负整数、字节 | LEB128 字节、值 | 无 | 是 | 越界与短输入 | 高 | `internal/protocol/varint` | 现有向量差分 |
| MessagePack | `:40 mp_pack`, `:97 mp_unpack` | Python 值、字节 | MessagePack | struct | 是 | 数值宽度、映射顺序 | 高 | `internal/protocol/msgpack` | 全类型字节差分 |
| ChaCha20 | `:206-235` | key、nonce、数据 | 密文/明文 | 32 位运算 | 是 | 溢出必须一致 | 高 | `internal/crypto/cc2p` | RFC 7539 向量 |
| Poly1305 | `:255-272` | AAD、密文、密钥 | 16 字节标签 | 大整数 | 是 | clamp 与填充 | 高 | `internal/crypto/cc2p` | RFC 7539 向量 |
| CC2P | `:275-292` | key、nonce、数据、tag | 密文、明文或失败 | ChaCha/Poly1305 | 是 | 标签验证顺序 | 高 | `internal/crypto/cc2p` | 篡改标签测试 |
| 游戏帧 | `:297 build_server_frame`, `:317 parse_client_packet`, `:337 decode_client_body` | cmd、seq、payload | 完整线帧 | Varint、CC2P、MessagePack | 是 | 大端序、首包布局 | 高 | `internal/protocol/frame` | 原始字节差分 |
| RSA 握手 | `gxjy_server.py:105 rsa_decrypt_pkcs1`, `:8758 recover_session_key` | 128 字节块 | 32 字节密钥或空 | PEM、N/D/E | 是 | PKCS#1 v1.5 | 高 | `internal/crypto/handshake` | 固定测试密钥差分 |
| REF 解密 | `gxjy_decrypt.py:15-52`, `:55 decode_file` | REF 文件 | 解密字节或错误文本 | XXTEA | 是 | 小端 32 位溢出 | 高 | `internal/assets/ref` | 固定样本哈希 |
| Cocos UUID | `read_config_table.py:95 decode_uuid` | 22 字符 UUID | 标准 UUID | Base64 表 | 否 | 非 22 字符原样返回 | 高 | `internal/assets/cocos` | 表驱动测试 |
| 配置定位 | `:112 bundle_paths`, `:126 table_path` | 表名 | 资源路径 | bundle config | 否 | 索引和 UUID | 高 | `internal/config/table` | 已知表路径测试 |
| 配置加载 | `:137 load_table`, `:175 _load_one`, `:217 unpack_ref` | 表名 | 合并后的字典 | REF、GZip、JSON | 是 | `_subN` 与 `_subFiles` | 高 | `internal/config/table` | 行数与哈希差分 |
| 配置缓存 | `:175 _load_one` | 源文件元数据 | GZip JSON 缓存 | mtime、size | 是 | 缓存失效规则 | 高 | `internal/config/cache` | 冷热加载测试 |
| 存档加载 | `gxjy_server.py:679 load_player` | JSON 文件、环境开关 | 全局玩家状态 | 默认值、配置表 | 否 | 缺失键合并和旧档兼容 | 高 | `internal/storage/player` | 存档 fixture 差分 |
| 存档保存 | `:757 save_player`, `:790 _rotate_backups` | 玩家状态 | 主档与备份 | 文件系统 | 否 | 防覆盖、非原子写 | 高 | `internal/storage/player` | 临时目录故障测试 |
| 角色快照 | `:1088 hero_record`, `:1498 owned_heroes`, `:5925 make_role_info` | 玩家与配置 | 协议对象 | 多张英雄表 | 否 | 大量可选字段 | 中高 | `internal/game/hero` | Golden MessagePack |
| 好感与语音 | `:1265-1495`, `:4238 favor_gift` | 英雄、礼物 | 好感、解锁列表 | HeroFavor、Sound | 否 | 条件链与缺资源 | 中 | `internal/game/favor` | 表驱动和客户端验收 |
| 探索场景 | `:1527-1973 scene_*`, `:3228-3359 explore_battle_*` | 场景、case、位置 | units、battle_info | Explore* 表 | 否 | 状态机和首触发 | 中 | `internal/game/explore` | Python 差分加模拟器 |
| 编队 | `:1998-2229 team_*` | 玩家、模式 | teams、expand_teams | Team、ResourcesMap | 否 | 下标从 0/1 混用 | 中高 | `internal/game/team` | 编队 fixture 与实机 |
| 奖励展开 | `:2252-2688 reward_*`, `apply_rewards` | 奖励 ID/结构 | 扁平奖励与状态变更 | RewardConfig | 否 | 递归、权重、动态形状 | 中高 | `internal/game/reward` | 固定随机源差分 |
| 战斗准备 | `:2691-3578` | 关卡、队伍 | battle_info | Fight*、Tower | 否 | 数组/字符串混用 | 中高 | `internal/game/battle` | 1302/1152/2102 fixture |
| 战斗结算 | `:3641-3697`, `:4038-4528` | battle result | 奖励、进度、存档 | 奖励、任务、关卡 | 否 | 首通、失败退款、推送顺序 | 中 | `internal/game/battle` | 1351 差分与模拟器 |
| 装备操作 | `:3700-3961 equip_command` | 装备命令与 payload | 装备状态、材料扣除 | Equip* 表 | 否 | 命令共享但分支多 | 中高 | `internal/game/equipment` | 每命令 fixture |
| 系统解锁 | `:4552-4695` | 玩家进度 | unlock IDs | UnlockSystem | 否 | 客户端加载顺序 | 中高 | `internal/game/unlock` | 登录推送顺序测试 |
| 抽卡 | `:4701-4914`, `:7541-7931` | 池、次数、随机数 | 英雄/装备/碎片 | DrawCard、Activity | 否 | 权重、保底、随机一致性 | 中 | `internal/game/gacha` | 注入 RNG 差分 |
| 体力 | `:4917-4981` | 时间、玩家状态 | 当前体力和恢复时间 | BaseConfig | 否 | 秒/毫秒与取整 | 中高 | `internal/game/stamina` | 注入时钟差分 |
| 商店支付 | `:4991-5262`, `:7945-8209` | 商品、购买次数 | 奖励、货币、邮件推送 | Shop*、RewardConfig | 否 | 本地派生购买语义 | 中 | `internal/game/shop` | 商品 fixture 差分 |
| 英雄养成 | `:5265-5708 hero_develop` | 命令、材料、模式 | 英雄状态与扣料 | HeroLevel/Quality/Star/Awaken | 否 | 类型夹取和目标行计价 | 中高 | `internal/game/progression` | 1052-1062 表驱动 |
| 巡逻试炼 | `:5711-5893` | 关卡与时间 | 奖励、auto_info | Resources* | 否 | 嵌套奖励与次数 | 中 | `internal/game/resource` | fixture 差分 |
| 背包 | `:6019-6649` | 道具与装备 | items、equipment | ItemConfig、Equip* | 否 | 实例 ID 与配置 ID | 中高 | `internal/game/inventory` | 存档前后差分 |
| 章节地图 | `:6654-7047` | 玩家进度 | chapters、探索度 | FightMap、Task、Explore | 否 | 空字典具有特殊含义 | 中高 | `internal/game/chapter` | 1301 字段和实机 |
| 任务 | `:7071-7536` | 玩家状态、任务 ID | tasks、rewards | TaskBase | 否 | 条件形状与自动领取 | 中 | `internal/game/task` | 1552/1555 fixture |
| 世界地图 | `:8077-8172` | 条件与进度 | 解锁映射 | ExploreWorld* | 否 | 发送顺序影响界面 | 中高 | `internal/game/world` | 1851 顺序测试 |
| 会话发送 | `:8253 send` | cmd、消息 | 加密帧与日志 | 协议层 | 是 | seq 和同步写 | 高 | `internal/session` | net.Pipe 测试 |
| 登录推送 | `:8287-8389` | 玩家状态 | 多条主动推送 | 多业务模块 | 是 | 顺序严格 | 中高 | `internal/session/login` | 命令序列 Golden |
| GM 命令 | `:8393-8647` | 名称、参数 | 玩家变更与文本 | 几乎全部业务 | 否 | 部分只确认不执行 | 中 | `internal/game/admin` | 每命令状态差分 |
| 会话循环 | `:8649-8785` | WebSocket 数据 | 解码、分发、错误回包 | 网络、协议 | 是 | 首包含认证消息 | 高 | `internal/session` | 完整握手集成测试 |
| 消息分发 | `:8787-10135 dispatch` | cmd、msg | 回包、推送、状态变更 | 所有业务 | 部分 | 约 90 个显式消息 | 中 | `internal/dispatcher` | 每消息契约 fixture |
| 配置预热 | `:10161 warm_tables` | 无 | 缓存与日志 | 38 张重表 | 否 | 失败后仍继续 | 高 | `internal/config/bootstrap` | 预热结果报告 |
| APK 补丁 | `rebuild_apk.py` 全文件 | 基线 JS、补丁表 | 新 JS/JSC/APK | Node、JDK、ADB | 是 | 精确文本锚点 | 中高 | 保留 Python 工具 | 哈希、回读、签名验证 |
| JSC 容器 | `patch_cvguard.py:49-119` | JS/JSC | GZip+XXTEA | 固定容器格式 | 是 | 填充与压缩差异 | 高 | 可选 `tools/jsc` | Python 字节差分 |

## 7. 协议事实

### 7.1 官方客户端上行

```text
[encType:1]
[varint(rsaBlockLen)]
[rsaBlock]
[ciphertext]
[tag:16]
[nonce:12]
```

首包的 RSA 区块和加密认证消息位于同一帧。不能只处理密钥后丢弃余下密文。

解密明文：

```text
[seq:u32 big-endian]
[cmd:u16 big-endian]
[msgpack body]
```

### 7.2 服务端下行

```text
[varint(1 + payloadLen)]
[encType]
[ciphertext]
[tag:16]
[nonce:12]
```

下行明文同样使用大端 `seq` 和 `cmd`。

### 7.3 社区客户端

代码还支持 plain MessagePack 帧。首包用启发式判断。Go 版必须保留双协议，除非后续有证据删除。

### 7.4 MessagePack 子集

当前实现支持：nil、bool、有符号和无符号整数、float32 解码、float64 编解码、字符串、二进制、数组、映射。

编码会选择最小整数宽度。Python `dict` 保留插入顺序。Go `map` 不保证顺序。客户端通常按语义读取映射，但字节级差分会受影响。计划对协议 Golden 使用有序对象；业务语义比较允许映射顺序不同，并明确记录。

## 8. 分发覆盖

当前 `Session.dispatch` 明确匹配约 90 个消息名称。协议映射共 554 条。其余条目包含服务端推送、内部服务消息、当前客户端不发送的命令和真正未实现命令，不能直接把差值称为缺失功能。

显式处理按组如下：

- 登录角色：101、1003、1015、1004、1011、1013、1005、1006。
- 场景探索：1851、1852、1853、1854、1855、1857、1863、1864。
- 章节战斗：1301、1302、1303、1306、1307、1308、1309、1310、1351、1354、1355、1356、1359。
- 试炼塔：1151、1152、2101、2102、2401。
- 英雄养成：1052、1053、1054、1055、1057、1058、1059、1060、1061、1062、1063、1065、1066。
- 羁绊：1067、1068、1070、1559。
- 背包装备：1252、1254、1265、1676、2051-2055。
- 抽卡活动：1601、1602、1603、1951、1983-1986、1991。
- 商店支付：1651、1652、1656-1658、1660、1663、1668、1670-1672、1752。
- 任务：1552、1555。
- 社交空契约：好友、聊天、黑名单、公会、邮件、竞技场。
- 管理维护：901、905、1014。

未处理命令会记录 `UNHANDLED`，随后返回 `{result:0, cmd:name, data:{}}`。该兜底本身属于兼容行为。迁移时不能默默改成错误码。

## 9. 迁移风险

### 9.1 动态类型

Python 代码大量使用 `dict.get(...) or {}`、`or []`、`setdefault`、`isinstance` 和宽容的 `int(...)`。这会把 `None`、空字符串、零、空容器统一视为假值。Go 的零值和类型断言不会自动复制此语义。

计划：

- 配置表先用受控的 `Value`/`Object` 层。
- 稳定业务边界再转换为明确结构体。
- 为 `PyTruthy`、宽容整数转换和列表归一化建立小型兼容函数。
- 不允许业务层到处散落 `map[string]any` 强转。

### 9.2 JSON

Python JSON 整数保持任意精度。Go `encoding/json` 解到 `any` 时默认变成 `float64`。这会破坏大整数、协议编号和时间戳。

计划使用 `Decoder.UseNumber()`。配置转换必须区分：字段缺失、显式 null、空对象、空数组、零和空字符串。存档输出默认带缩进并保留非 ASCII 字符；Go 需关闭 HTML 转义，并固定换行与缩进。

### 9.3 数值

ChaCha20 和 XXTEA 依赖 32 位无符号溢出。Go 必须使用 `uint32`。MessagePack 需检查 `int` 平台差异，内部使用 `int64`/`uint64`。

Python 大整数用于 Poly1305 和 RSA。Go 使用 `math/big` 或已验证库。

### 9.4 容器

- Python `dict` 有插入顺序；Go `map` 无顺序。
- Python `set` 去重后顺序不稳定；当前代码常再排序。
- Python tuple 在线上编码为数组。
- `bytes` 与 `bytearray` 都按 binary 编码。
- Python 允许容器里混合类型。

计划在配置读取层保留动态值，在业务输出层逐步类型化。需要有序字节输出的地方使用 `OrderedObject`。

### 9.5 错误行为

Python 有 169 处异常处理，很多捕获所有 `Exception`。部分错误被吞掉并继续，部分错误会发送 `result:1`，部分错误会回退到默认对象。

Go 迁移不能统一改成“返回 error”。每个调用点必须记录原行为：

- 记录后继续。
- 返回默认值。
- 丢弃当前帧。
- 发送失败回包。
- 终止连接。
- 终止进程。

### 9.6 并发

Python 为每个 HTTP 和 WebSocket 连接建立线程。所有连接共享 `_player`、`_current_point`、`_current_explore_case` 和缓存。GIL 不能保证跨函数事务安全。

Go goroutine 会让竞态更明显。计划使用单玩家 `StateStore` 锁。一次命令的读取、修改、存档和相关推送应在明确事务边界内执行。网络写入使用每会话发送锁。

### 9.7 加密

- RSA 使用 PKCS#1 v1.5，不是 OAEP。
- 会话密钥可能是原始 32 字节，也可能是十六进制文本。
- CC2P 是 RFC 7539 ChaCha20-Poly1305。
- nonce 为 12 字节，tag 为 16 字节。
- AAD 当前为空。
- 解密必须先验 tag，再解密。
- REF 与 JSC 使用不同 XXTEA key 和容器布局。

这些路径必须先完成字节级差分测试。

### 9.8 时间与随机数

代码混用秒级时间戳、毫秒时间戳、本地时间日志和备份时间字符串。必须逐字段确认单位。

Python `random` 与 Go `math/rand` 算法不同。抽卡、奖励权重和随机名字不能用固定种子直接期待相同序列。计划注入 RNG：差分测试使用预设抽样结果，生产实现只保证已证实的分布。协议 nonce 使用 `crypto/rand`，不能使用游戏 RNG。

### 9.9 排序

备份按文件名排序。英雄、表行、消息映射中同时存在配置顺序、字典插入顺序和显式 `sorted`。迁移时必须逐处确认，不能全局改为按键排序。

### 9.10 网络库语义

改用 `net/http` 会自动改变头部、状态行、路径解析和连接管理。客户端可能不关心，但必须用原始 TCP Golden 验证。WebSocket 当前实现只覆盖客户端实际使用的帧子集。不要提前扩展后改变关闭或错误行为。

## 10. 已知业务不确定项

1. 探索宝箱、1855 路上战斗和剧情正向状态机仍待实机验收。
2. 试炼星星已有代码和协议证据，但最新文档仍标记实机待验。
3. 好感度剧情文本尚未完整实现。
4. 部分社交、公会、邮件和竞技场仅返回安全空结构。
5. GM 中部分命令只确认收到，没有真实状态实现。
6. 需求文档中的 P1-P5 是派生规则，不是官方配置原义。
7. 当前发布包已经修改客户端，早期“绝不改 APK”要求已与实际产物冲突。
8. 当前包没有文档提到的三套合同测试脚本。
9. 音频资源缺失时只能避免崩溃，不能恢复真实语音。
10. 协议表存在重复编号名称，反向映射会过滤内部消息后取首项。

这些项目不能标记为“与原版一致”。

## 11. 安全与协作结论

当前服务监听 `0.0.0.0`。HTTP `/gm` 无认证。登录 token 固定。该行为适合受控本地实验，不适合公网。

Go 版应默认监听 `127.0.0.1`。如果为了严格复现保留全网卡模式，必须通过显式配置开启。管理端点必须独立开关，并从环境或配置读取随机凭据。

安全加固不得混入协议兼容提交。兼容模式与安全默认值应分别记录。

## 12. 本阶段验证结果

实际执行：

```powershell
py -3 gxjy_protocol.py
```

结果：8 组 MessagePack、Varint、Plain 帧、RFC 7539、官方帧、上下行解析和 554 条协议映射全部通过。

实际执行：

```powershell
py -3 gxjy_server.py --help
```

结果：入口可加载。由于 `cryptography` 未安装，当前走 `server_rsa.json` 回退。

未执行：

- 完整服务启动。
- 配置表全量预热。
- 模拟器登录。
- APK 重建。
- 真实客户端协议回放。
- Go 构建和测试。

原因：第一阶段只做分析；Go 工具尚不可用；实机流程耗时且会修改运行状态。
