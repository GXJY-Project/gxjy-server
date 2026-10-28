# B 服更新主脚本解码与协议证据

日期：2026-10-25

状态：**“解码 B 服主脚本”已完成离线验收**

更新主脚本、登录请求构造、消息编号、RSA/CC2P 流程及启动加载顺序已有可复现证据
原客户端登录现有 Go 服务仍未完成实机验收，现有两份服务端密钥均与客户端公钥不匹配

## 范围与来源

本次继续使用已授权的 `com.bilibili.gxjy` 备份及其 APK、更新资源
来源、授权边界与全量资源完整性见 [备份分析](bili-backup-analysis.md) 和 [资源基线](bili-resource-baseline.md)
没有连接官方 HTTP、CDN、SDK 或游戏服务，没有修改 APK、正式运行配置或玩家存档

相对于 `gxjy-server/` 的本地证据目录：

| 内容 | 路径或标识 |
|---|---|
| 原始 APK | `data/assets/bili/source/backup/a/base.apk` |
| APK SHA-256 | `d5cc33f3406d10198d08b0de7a2086f2aa68af9bb6a537f9ab80133c3c2333ee` |
| 原生加载实现 | `data/assets/bili/source/apk/lib/x86_64/libcocos2djs.so` |
| 原生库 SHA-256 | `30c643281dc4241e6d1c55428529372c5ac9751f2d2b34cc59ae2f9d49b35020` |
| 更新资源 | `data/assets/bili/resolved/`，主 project 版本 `1.1.281.1.1` |
| Python XXTEA 原语 | 本地发布包 `3-CLIENT-SRC/patch_cvguard.py` 的 `_s2l`、`_l2s`、`xxtea_dec` |
| 原型 SHA-256 | `53b3874bad499d7491b1cdf90d1ca38f9432ee247ec4524c38b0c9c66793fae8` |
| 解码与提取结果 | `data/assets/bili/analysis/verified/` |

客户端材料、密钥和完整编号表均保留在现有 Git 忽略目录
本报告只记录协议结论、函数定位、完整性标识和合成验证方式
原型复核通过 AST 仅执行三个既有算法函数，不导入旧工具或执行其客户端补丁入口

## JSC 解码结论

**结论：该快照的 `.jsc` 是使用独立脚本密钥加密的 GZip JavaScript，解码顺序为 XXTEA → GZip → UTF-8**

状态：已验证，仅针对上表标识的 APK 与更新快照

来源与调用链：

1. x86_64 `AppDelegate::applicationDidFinishLaunching` 起点为 `0x7ab500`
2. `0x7ab5a4` 与 `0x7ab5b6` 构造 12 字节脚本密钥，`0x7ab5d1` 调用 `jsb_set_xxtea_key`
3. `0x7ab5ed` 调用 `jsb_init_file_operation_delegate` 注册文件读取代理
4. 读取代理优先寻找对应 `.jsc`，不存在时才走脚本文本读取路径
5. 两个读取分支在 `0x821ae7`／`0x8221b3` 调用 `xxtea_decrypt`，随后分别在 `0x821b00`／`0x8221cb` 检测 GZip，在 `0x821b1a`／`0x8221e2` 解压
6. 解密输出不带 GZip 标识时，原生实现另有直接使用输出的分支；本次三个更新脚本均命中 GZip

原生地址是 ELF 虚拟地址，不能直接当文件偏移
`scripts/decode_bili_scripts.py` 先固定原生库 SHA-256，再通过 ELF 节地址转换提取参数并核对指令
该工具不输出密钥，遇到其他原生库版本会拒绝套用这些地址

容器细节：

- 本次 JSC 没有 REF 头、额外签名或 REF 内层长度前缀，整个文件传入 XXTEA
- XXTEA 使用小端 32 位字、长度尾字语义，12 字节密钥补零至 16 字节
- GZip 长度与 JSC 长度不同，不能把解密后的填充或长度尾字交给 JavaScript
- APK `assets/main.js` 是另一条 REF／资源密钥路径；不能将其参数套用到 JSC
- 资源密钥、脚本密钥、RSA 密钥及 CC2P 会话密钥属于不同用途

验证结果：工具逐项核对更新清单的大小与 MD5、解密后的 GZip 标识及 UTF-8，然后重跑核对输出字节

| 更新文件 | JSC 字节数 | 解密 GZip 字节数 | JavaScript 字节数 |
|---|---:|---:|---:|
| `assets/main/index.jsc` | 2,940,380 | 2,940,376 | 15,803,233 |
| `src/settings.jsc` | 308 | 303 | 600 |
| `src/assets/Script/Global.jsc` | 1,964 | 1,960 | 4,733 |

更新主脚本明文 SHA-256：

```text
a8a75655aafdf51a7b0cc397e64b9d3efe789367311cc9ae80b69a4b9901cda0
```

其余输入、输出哈希及协议表资源定位见本地 `analysis/verified/evidence.json`
解码工具遇到已有输出内容不同会拒绝覆盖

工具选择复查过 Cocos 原生加载实现及 `ifduyue/xxtea` 的库说明
后者标明 BSD-2-Clause，默认采用非标准四字节 PKCS#7 填充；不能将默认库行为视为本快照的长度尾字布局
本次没有引入新依赖，也没有复制网上解密器，继续使用已确认的原型原语
外部实现只供比较，B 服结论依据仍是本地 APK 与实际解码结果
参考：`https://github.com/cocos/engine-native/blob/v2.4.6/cocos/scripting/js-bindings/manual/jsb_global.cpp`、`https://github.com/ifduyue/xxtea`
本轮网页检索未确认这两条参考满足半年内维护条件，不据此宣称其最新版本已适用于本快照

## 登录接口与字段

**结论：登录链路使用两个 POST JSON 接口和一个 GET 查询接口，连接 WebSocket 后发送 `MSG_TYPE_AUTH`**

状态：请求构造与响应字段消费已验证；SDK 服务端认证语义、真实响应及账号策略待验证

来源：更新主脚本的 `LoginModel`、`Http.easyHttpRequest`、`LoginServer.synchronize`、`SDKManager` 和渠道 delegate

| 顺序 | 方法与路径 | 请求内容 | 客户端消费内容 |
|---|---|---|---|
| 资源发现 | POST `player/getResource` | `initBaseInfo()` 加 `network_type`，JSON 正文 | `result`、`notice`、`resources.cdn`、`resources.version`、`resources.remoteCfg`、布尔 `maintain` |
| 渠道登录后取区服 | POST `player/login` | 基础信息加 `openid`、`access_key`、`channel_token`、`push_token`、`channel_id`、`network_type` | `result`、`recommend`、`sec_infos`、`role.info_map`、`role.openid`、可选实名认证状态、`token` |
| 选择区服获取网关 | GET `player/enter` | 查询参数见下文，正文为空 | 顶层 `result`、`gate_ip`、`port`、`type` |
| 建立网关连接 | WebSocket `<type>://<gate_ip>:<port>/game` | 二进制认证帧 | AUTH 响应的 `result`、可选 `login_queue` |

`requestGate` 将方法参数传为 `null`，`easyHttpRequest` 用 `i || "GET"` 选择 GET，并逐字段 `encodeURIComponent`
因此不能把 `player/enter` 记作第三个 POST 接口
POST 请求显式设置 `Content-Type: application/json`

`initBaseInfo` 包含 `platform`、`agent`、`local`、`environment`、`device_os`、`version`、`base_version`、`device_id`、`device_type`、`channel_id`、`ad_channel_id`、`sdk_name`、`lb_small_pkg`、`ratio`、`os_version`
更新主脚本的 `devConfig` 设置 `publishPlatform = "bagent_bili_cn_ob"`，`initBaseInfo` 按四段拆为 agent/platform/local/environment
`base_version` 来自原生注入的 `window._BASEPACKV`，不能与更新版本混为一项

`player/enter` 查询字段是：`openid`、`sec_id`、`token`、`client_version`、`version`、`platform`、`agent`、`local`、`environment`、`device_os`、`base_version`、`sdk_name`、`ad_channel_id`
`LoginModel.getVersion` 将超过三段的更新版本截为前三段；当前 project 对应 `1.1.281`
`LoginServer.synchronize` 从响应顶层拼接 `/game`，不优先使用额外的嵌套 gate 对象或 `url` 字段

渠道链路通过原生 SDK 回调提供 `uid`、`accessToken`、`channelToken` 和 `pushToken`，再交给 `requestServerList`
`SDKManager.getSDKName` 从原生桥接选择 `bilibili`、`bilibiliUO` 等 delegate
脚本中的渠道字段流向已确认，原生 SDK 当前实际分支及外部认证流程未进行联网验证
测试只使用 `synthetic-*` 值，未使用真实账号或会话

WebSocket `getEnterParams` 除 `is_reconnected`、`openid`、`token` 外，还带设备、网络、区服、渠道和版本元数据
`requestEnterGame` 调用 `request(..., "MSG_TYPE_AUTH", ...)`；表中存在 `MSG_TYPE_LOGIN` 不代表它是本版本实际发送的认证命令

## 消息编号与分表

**结论：编号取自更新资源 `SERVER_Protocol` 及 `SERVER_Protocol_sub1`，必须合并分表**

状态：表内容、合并规则与 Go 读取结果已验证；每个业务命令的完整语义不在本次验收范围

来源：`NetManager.getProtocolCode` → `DataReader` → `JsonReader._createJsonObj`
主表 `_subJsonCount = 1`，客户端删除表元数据后以 `Object.assign` 合并，后表同名行覆盖前表
离线执行原 `JsonReader` 已用合成同名行验证覆盖规则，没有将协议行按编号去重

| 表 | 资源定位 |
|---|---|
| `SERVER_Protocol` | `assets/resources/import/14/14455dfa-8cc5-4a7b-acf8-1104c183b503.json` |
| `SERVER_Protocol_sub1` | `assets/resources/import/89/89a438ff-ef29-4779-97eb-a197cf3cb43e.json` |

主表有 530 个非元数据行，合并后共 661 个名称映射
与现有 `data/protocol/protocol_map.json` 的 554 项比对：原有项全部存在且编号一致，新增 107 项，没有删除或改号
新表写入独立的 `analysis/verified/protocol_map.json`，未覆盖当前运行映射

| 名称 | 编号 | 本次可确认的用途 |
|---|---:|---|
| `MSG_TYPE_AUTH` | 101 | `requestEnterGame` 的实际认证命令 |
| `MSG_TYPE_LOGIN_QUEUE_RES` | 106 | 客户端注册的排队响应 |
| `MSG_TYPE_RESUME_RESPONSE` | 108 | `NetNode` 重连续传响应处理 |
| `MSG_HEARTBEAT` | 901 | 心跳命令 |
| `MSG_ROLE_INFO_RES` | 1001 | 角色信息映射，完整业务内容未在本轮验收 |
| `MSG_LOGIN_INFO_RES` | 1017 | 登录信息映射，完整业务内容未在本轮验收 |

有 19 组重复编号，全部保存在 `evidence.json` 的 `protocol_aliases` 中
多组包含名称带 `GAME2WORLD`、`GAME2GLOBAL` 等标识的行，不能把全表当作单一无歧义的客户端 dispatch 表
其中 `MSG_ACTIVITY_RUNE_NEXT_STEP` 与 `MSG_SWEEP_ACTIVITY_FIGHT` 同为 **1998**，业务区分仍待确认
没有凭名称猜测改号、删除行或选择覆盖编号
全表没有超出 u16 的值，但这不能证明每项都属于客户端可收发命令

## RSA、CC2P 与帧格式

**结论：本版本使用 RSA PKCS#1 v1.5 传递会话密钥，消息采用 CC2P，不符合历史 bili 分支的明文假设**

状态：原脚本构造帧、独立解密、下行解析及 Go 字节回归已验证；真实私钥握手与实机时序待验证

来源：`data_pack.packMsg`、`unpackMsg`、`resetUseKeyData`，`NetNode.request` 和 `LoginModel.reconnectGateServer`

- `data_pack` 初始化随机生成 32 字节会话密钥，用内嵌 RSA 公钥和 `RSA_PKCS1_PADDING` 加密原始字节
- 首包带 128 字节 RSA 区块，后续包 RSA 长度为 0
- `resetUseKeyData` 复位“已发送密钥”标记，重连再次携带同一初始化密钥的 RSA 区块，没有在该函数重新生成密钥
- CC2P 是 ChaCha20-Poly1305，本次帧可被 Node 标准加密库独立解密，AAD 为空，标签 16 字节，nonce 12 字节
- `NetNode.request` 封装 `{msg: 业务对象, seq: 请求序号}`；二进制头中的消息序号使用另一计数器

原脚本生成的布局：

```text
上行：0x01 | varint(RSA 字节数) | RSA 区块 | 密文 | tag[16] | nonce[12]
明文：消息序号 u32 大端 | 命令号 u16 大端 | MessagePack({msg, seq})

下行：varint(1 + 加密载荷字节数) | 0x01 | 密文 | tag[16] | nonce[12]
明文：消息序号 u32 大端 | 命令号 u16 大端 | MessagePack(响应对象)
```

上行没有额外的整个消息长度前缀；下行没有 RSA 长度字段
Go `internal/protocol/frame.go` 对这三类合成上行及一类下行的字节结果均与客户端证据一致
重连样本特意令二进制头序号为 1、MessagePack `seq` 为 3，验证两个字段不会被混为同一个值

原 `unpackMsg` 用 `e.slice(t)` 返回剩余数据，未将长度前缀本身计入偏移
本次单帧下行解析后 `remainData.byteLength = 1`，原脚本标签篡改后返回 `cmdId = null`
这两项已留在离线验证断言中，没有顺手修正客户端行为
多帧合包、长长度前缀及实际重连补发仍需后续专项验证

### 当前密钥阻碍

客户端公钥为 1024 位 RSA，SPKI DER SHA-256 为：

```text
2e742265a6e64446c029abdc6bc055aa1756b32d71ecc208001af6fc520233b9
```

`scripts/check_bili_wire.cjs` 拦截原客户端加密调用，仅检查 padding、合成明文和公钥指纹，然后仍执行原加密函数
与本地 `data/keys/gate.pem` 的公钥及 `data/keys/server_rsa.json` 的 N/E 比对均不相同
因此当前服务端密钥无法按该客户端公钥恢复握手会话密钥
帧兼容通过不等于真实 RSA 握手通过

后续适配需维护者决定匹配的已授权测试密钥来源，或明确批准自有测试客户端的公钥和服务地址修改方案
本任务未更改客户端公钥、关闭加密校验或引入明文回退

## 资源加载顺序

**结论：更新目录先于 APK 搜索，基础脚本与基础 bundle 汇合后才加载主 bundle；主资源更新与可选子包另有流程**

状态：原生调用链与 APK 启动脚本已验证；手机上实际资源请求、配置合并结果及子包选择状态未实机验证

1. `AppDelegate` 使用 writable path 拼接 `remote-asset`，在 `0x7abf30` 调用 `FileUtils.addSearchPath(..., true)` 将其加入搜索路径前端
2. APK `assets/main.js` 启动时检查 `remote-asset_temp/`；目录存在且没有 `project.manifest.temp` 时执行收尾搬移，再清理临时目录
3. 通常的 JSB 分支依次 require `src/settings.js`、`src/cocos2d-jsb.js`，按物理开关选择 `src/physics.js`，最后 require `jsb-adapter/jsb-engine.js`
4. `cc.assetManager.init` 消费 settings 的 `bundleVers`、`remoteBundles` 和 `server`
5. 同时提交 `loadScript(settings.jsList)`、`loadBundle("internal")`、`loadBundle("resources")`
6. 三个完成回调全部成功到达后，才 `loadBundle("main")`，随后 `cc.game.run` 和场景 `db://assets/Scene/main.fire`

当前 settings 的脚本请求列表依次包含 `Global.js`、Cubism core 完整版和 min 版
离线验证改变完成回调顺序两次，均确认 main 必须等待全部基础项
脚本请求数组顺序与回调完成顺序不能混为一谈
存在 `loadRuntime` 的另一原生分支加载 `cocos2d-runtime.js` 与 `jsb-adapter/engine/index.js`，本轮执行验证覆盖通常的 JSB 分支

`GameManager.onLoad` 等待 `loadSubBundles` 后才建立 SDK、音频、Updater 和 UI 管理对象，再进入 `OnStart`
`GameManager.loadSubBundles` 顺序为 **CVSound → live2d → heroVideo → remote**，每项等待前一个 Promise 完成
原生分支优先检查 `writable/subBundles/<name>/config.json`，存在则加载本地路径；否则检查 APK 内 `assets/<name>/config.json`，存在才加载底包中的该 bundle，两处都不存在则跳过
遍历完成后从 `G_Config.subBundles` 移除 `remote`，留下 **CVSound → live2d → heroVideo** 的初始下载调度顺序
这套本地优先、底包回退、缺失跳过及 remote 移除规则已用原方法离线执行验证

`LoginModel.requestLogin` 获取 CDN/remoteCfg 后，更新界面通过主 `Updater` 检查版本
主更新器本地 manifest 为 `assets/project.manifest`，落地 `writable/remote-asset`
远端主 manifest 根按 `<cdn><语言>/<resLevel>/main/` 拼接，`resLevel` 默认 high
更新完成后允许回调接管，否则执行 `GameRestart` 使新脚本进入后续加载链

`SubBundleUpdater` 落地 `writable/subBundles/<name>`，远端根按 `<cdn><语言>/<resLevel>/<name>/` 拼接
已落地的子包为 `heroVideo`、`live2d`、`CVSound`
下载调度遍历运行时 `G_Config.subBundles`，逐项检查选择状态、完成状态和大小，再下载第一个符合条件的子包
初始数组已核实；后续 `Event_Merge_Config` 仍可能覆盖非 startup 参数，手机最终配置及选择状态未实机验证
`LoginModel` 的 `UPDATEPROGRESS_FINISH` 监听器移除同名旧 bundle，调用 `CommonUtils.loadSubBundleByPromise` 加载新本地目录，再推进后续下载
实际下载也受平台、`isMultBundle`、skip 开关和 `noUpdate` 影响

原 project/version 元数据的 281／279 差异仍按资源基线原样保留，本任务未通过改写 manifest 消除差异

## 复现与验收

在 `gxjy-server/` 执行，Python 使用本机现有解释器，不需新安装包：

```powershell
& 'C:\Users\WANGYU\.cache\codex-runtimes\codex-primary-runtime\dependencies\python\python.exe' -B scripts/decode_bili_scripts.py
node scripts/check_bili_wire.cjs
node scripts/check_bili_bootstrap.cjs
```

- 解码器固定原型和原生库版本、校验资源清单，写入三个解码脚本、协议映射和证据索引
- wire 检查使用原 bundle 的网络模块，禁用 bundle 游戏入口，固定随机源，只构造合成账号帧
- bootstrap 检查只执行原 APK 启动脚本与选定模块 factory，文件操作、SDK 元数据和 HTTP 请求都由内存替身接收
- 两个 Node 工具在写入前核对 Git 忽略规则，输出与既有文件不同则拒绝覆盖
- wire 检查逐字节核对 Go 固定样本，结果另存 `wire-check.json`
- 启动与请求构造结果另存 `bootstrap-check.json`，子包加载结果另存 `subbundle-check.json`

验证记录：

| 实际检查 | 结果 |
|---|---|
| 更新脚本解码复跑 | 3 个脚本、661 项映射，输出完全一致 |
| 客户端 wire 离线复跑 | 首包、后续包、重连包通过，下行可读，篡改标签被拒绝 |
| 原脚本 bootstrap／登录构造 | 两种完成顺序通过，POST／POST／GET、remoteCfg、顶层网关字段及分表覆盖通过 |
| 原 `GameManager.loadSubBundles` | 顺序等待、本地优先、底包回退、缺失跳过及 remote 移除通过 |
| `go test ./internal/protocol ./internal/crypto ./internal/assets` | 通过 |
| `TestActualTablesMatchPython` 指向 B 服真实配置与本轮 table-fixture | `SERVER_Protocol` 全内容及键顺序通过 |
| `scripts/check.ps1` | gofmt、`go test ./...`、`go vet ./...`、`go build ./...` 通过 |

真实资源差分仅对 `internal/assets` 运行，注入独立 B 服资源路径与本地资源密钥
没有把仅含协议表的 fixture 传给全量业务测试，不能据此宣称所有 B 服业务已对齐

在独立 PowerShell 会话中复现该项差分，避免影响正常启动环境：

```powershell
$env:GXJY_DIFF_FIXTURE = Join-Path (Get-Location).Path 'data/assets/bili/analysis/verified/table-fixture.json'
$env:GXJY_DIFF_ASSET_ROOT = Join-Path (Get-Location).Path 'data/assets/bili/resolved/assets'
$env:GXJY_DIFF_BUNDLE = Join-Path (Get-Location).Path 'data/assets/bili/resolved/assets/resources/config.json'
$env:GXJY_ASSET_KEY = (Get-Content -Raw -LiteralPath data/keys/asset.key).Trim()
go test ./internal/assets -run '^TestActualTablesMatchPython$' -count=1
```

本轮未运行 race：没有修改并发逻辑
未执行官方客户端／SDK／真实密钥握手、实机更新下载或完整游戏登录

## 交付与后续边界

交付包含解码工具、两类原脚本离线检查、合成 wire fixture、Go 回归测试及本报告
之前的“JSC 解码未知”阻碍已解除，当前接入问题转为 RSA 密钥匹配、客户端访问地址以及 B 服业务响应的逐项对齐
重复编号 1998、下行多帧缓冲、原生 SDK 实际分支与子包运行时配置仍有明确待验证项
改动留在工作区，原有未提交工作保留，未提交或推送
