# B 服备份分析与接入方案

分析日期：2026-10-25

本报告记录首次只读分析，后续授权的资源落地执行情况见 [B 服资源基线落地记录](bili-resource-baseline.md)

后续已完成更新 JSC 解码及协议离线验证，见 [B 服脚本与协议证据](bili-script-analysis.md)
下文保留首次分析时的未知项，当前解码状态以新报告为准

## 本次基准决定

维护者明确要求后续以 B 服为准，不继续以台服资源恢复为目标
本次 B 服指 `com.bilibili.gxjy`，来源为根目录 `20261019_213556/光隙解语(com.bilibili.gxjy).bak`
维护者确认这份备份在备份前已经能够正常进入游戏

现有 Go 实现和 Python 原型可作为已有实现参考，原台服差分结果不能直接作为这版 B 服的兼容性证明
原型 `2-SERVER/protocol/read_config_table.py` 的历史 `bili` 分支注释指向 `com.bilibili.klight.bili`、`1.9.1687.1`
该历史分支与本次备份的包名和资源版本不同，不能仅设置 `GXJY_BUILD=bili` 就认为已经切换到本次基线

## 已验证的容器和内容

本轮读取文件头、完整 TAR 目录、APK ZIP 目录及必要的配置内容，未将 APK、媒体、脚本、数据库或账号缓存导出到文件系统

| 项目 | 本地观察 |
|---|---|
| 备份大小 | 8,107,358,794 字节，约 7.55 GiB |
| 外层 | `MIUI BACKUP`，版本 2 |
| 内层 | `ANDROID BACKUP`，版本 5，压缩标志 0，加密字段 `none` |
| TAR 起点 | 本文件的第 74 字节偏移，不能将这个偏移当作其他备份的固定规则 |
| TAR 内容 | 15,160 个条目，其中 13,321 个普通文件 |
| APK | `apps/com.bilibili.gxjy/a/base.apk`，3,107,808,246 字节 |
| APK 内部 | 55,013 个文件，未压缩内容合计 3,242,843,149 字节 |
| 备份说明 | `descript.xml` 中包名吻合，`error=0`，`completedSize` 和 `bakFileSize` 均等于实际文件大小 |

备份内路径以下列前缀为基准：`apps/com.bilibili.gxjy/`

| 路径 | 实际内容与用途 |
|---|---|
| `a/base.apk` | B 服客户端、基础资源、DEX、各架构原生库 |
| `r/lebian/cacheFiles/com.bilibili.gxjy-101/` | 乐变缓存，6,123 个普通文件 |
| `r/lebian/cacheFiles/com.bilibili.gxjy-102/` | 乐变缓存，452 个普通文件 |
| `f/remote-asset/` | 主资源更新、更新后的资源索引和主脚本 |
| `f/subBundles/heroVideo/` | 角色视频子包 |
| `f/subBundles/live2d/` | Live2D 子包 |
| `f/subBundles/CVSound/` | 语音子包 |
| `db/`、`sp/`、`r/app_webview/` 等 | 应用数据库、偏好及浏览器缓存区域，不作为 Go 玩家存档导入来源 |

本次没有发现可直接作为服务端工程使用的 `.go`、`.py`、`.sql` 或 `.proto` 外层备份文件
客户端资源、配置和脚本可以提供兼容证据，但服务端业务实现仍需要逐项对齐

## 资源完整性：清单内 59,599 项全部通过

以各 `project.manifest` 的相对路径、`size` 和 `md5` 为依据逐项读取内容校验
对主资源同时检索 APK、两层乐变缓存及 `remote-asset`，每一项必须找到大小和 MD5 同时匹配的实际文件
APK 条目读取时还会执行 ZIP 库的相应条目校验

| 清单 | 清单版本 | 条目数 | 大小及 MD5 一致 | 缺失／不一致 |
|---|---|---:|---:|---:|
| `f/remote-asset/project.manifest` | `1.1.281.1.1` | 55,997 | 55,997 | 0 |
| `f/subBundles/heroVideo/project.manifest` | `1.1.281.1.1` | 130 | 130 | 0 |
| `f/subBundles/live2d/project.manifest` | `1.1.281.1.1` | 2,806 | 2,806 | 0 |
| `f/subBundles/CVSound/project.manifest` | `1.1.281.1.1` | 666 | 666 | 0 |
| 合计 | | 59,599 | 59,599 | 0 |

清单内内容合计 6,338,442,854 字节，不包含整个 APK、额外缓存、私有数据库和清单本身
这是本地清单范围内的完整性验证，不是对所有历史活动资源、清单外资源或所有客户端运行场景的证明
MD5 用于与本地清单比较，不用于证明材料来源或发布者真实性

本轮主资源核验采用的匹配来源如下：

| 来源 | 采用的匹配项数 |
|---|---:|
| APK | 46,619 |
| `remote-asset` | 3,038 |
| 乐变缓存 `-102` | 438 |
| 乐变缓存 `-101` | 5,902 |

部分条目可能在多处存在
核验依次尝试 remote、cache102、cache101、APK，表格仅表示该次选中的匹配来源，不代表已经验证客户端运行时采用相同搜索顺序
重建资源时应按目标清单的哈希选择文件，不能直接整目录覆盖，也不能只取 APK 或只取 `remote-asset`

## 版本状态需要原样保留

| 来源 | 观察结果 |
|---|---|
| APK 内 `assets/assets/project.manifest` | `1.1.224.1.1`，53,993 个资源条目 |
| `f/remote-asset/project.manifest` | `1.1.281.1.1`，`updating=true` |
| `f/remote-asset/version.manifest` | 解码后为 `1.1.279.1.1` |
| 三个子包的 project/version manifest | 均为 `1.1.281.1.1`，project 的 `updating=true` |
| `f/remote-asset/project.manifest.temp.tmp` | 0 字节 |

主清单只有 2 个条目带 `downloadState=2`，其余没有该字段
本轮没有推断这个状态值的枚举含义
资源实物全部符合 281 清单，维护者也确认备份前能进入游戏，因此不能仅凭上述字段将备份判为损坏或未下载完成
这些观察也不能证明更新状态机为何保留 279 或 `updating=true`

后续应以此可运行 B 服快照为基线，同时保留原始元数据
整理资源时可使用 281 project 清单进行内容校验；实际版本响应、manifest 改写及运行时搜索优先级要以 B 服启动／更新逻辑核实后再决定
本轮未把 279 改为 281，也未清除更新标志

## 配置解码与 Go 接入条件

复用本地 Python 原型 `gxjy_decrypt.py:xxtea_decrypt_raw`，在内存中按既有 REF、GZip 和 Cocos 容器规则解析
密钥只从既有本地资源密钥文件读取，不写入本报告或新文件
未调用会生成配置缓存的原型 `load_table` 路径

| 资源索引 | paths 数 | UUID 数 | `config/data/` 索引项数 |
|---|---:|---:|---:|
| APK 基础索引 | 28,510 | 41,276 | 1,753 |
| 281 主清单对应的更新索引 | 29,810 | 42,680 | 1,838 |

`config/data/` 项数包含子表，不能直接称为独立业务表数量

对 `internal/game/runtime.go:requiredTables` 的 33 张必需表均完成实际解码
读取时按主清单校验对应文件，按原型规则合并 `_subJsonCount` 和 `_subFiles`
共读取 433 个不同配置文件，未发现子表缺失或合并错误

| 表 | 合并后的行数 |
|---|---:|
| HeroBase | 117 |
| HeroData | 385 |
| SkillBase | 8,533 |
| LeaderBase | 9 |
| UnlockSystem | 188 |
| PlayerLevel | 80 |
| RewardConfig | 44,305 |
| FightPoint | 1,198 |

这些结果说明现有资源解析思路可以读取 B 服配置
本轮没有让 Go Runtime 加载这些表，没有验证字段消费、角色初始化、战斗公式或登录回包是否与 B 服一致
不能把“33 张表可读”表述为“现有 Go 服务已适配 B 服”

## 主脚本与协议仍待核实

- APK 主脚本 `assets/assets/main/index.jsc`：2,921,504 字节
- 更新主脚本 `f/remote-asset/assets/main/index.jsc`：2,940,380 字节
- 两者均未带本轮配置使用的 `REF:` 文件头
- 直接用现有资源密钥调用既有 XXTEA 原语，主脚本、Global 和 settings 都没有得到有效文本
- APK 的 `assets/main.js` 已通过 REF、XXTEA、GZip 在内存中解码，包含以 writable path 加 `remote-asset` 定位更新目录的代码
- APK 含 `libcocos2djs.so` 的 arm64-v8a、armeabi-v7a、x86、x86_64 版本，可作为后续核对脚本加载方式的本地证据

目前不能确认 `.jsc` 的正确解码参数和完整加载链路
资源密钥可读配置，不等于它就是脚本密钥，也不等于网络会话密钥已知
应从这份 B 服 APK 的加载实现和更新后的主脚本继续核实 HTTP 登录、渠道认证、网关发现、命令编号及帧格式
不能直接沿用历史 bili 分支的明文 MessagePack 结论，或台服的 RSA／CC2P 结论

## 建议的实际使用顺序

1. 建立独立的 B 服本地资源目录，例如 `gxjy-server/data/assets/bili/`
   该位置已被现有 `/gxjy-server/data/assets/` 规则忽略
   首次分析时根目录 `20261019_213556/` 未被忽略，后续资源落地工作已补充对应忽略规则
2. 使用现成归档工具按清单导出，仅取 APK 和已确认的资源路径
   每个文件记录来源、目标相对路径、大小、哈希；拒绝越界路径，对已有目标不做无条件覆盖
   完整客户端需要基础 APK 以及匹配的缓存／更新／子包，单独安装原 APK不会自动得到快照里的更新资源
3. 将主清单路径导出到 `bili/resolved/` 下，保留 `assets/resources/...` 层级
   例如 APK 的 `assets/assets/resources/...` 对应 `resolved/assets/resources/...`
   三个子包分别存放并保留其原始清单，不混入主资源索引
4. Go 的资源接入点继续使用现有参数，不需要因此引入数据库或新解析依赖
   `asset_root` 指向 `data/assets/bili/resolved/assets`
   `bundle_config` 指向 `data/assets/bili/resolved/assets/resources/config.json`
   使用独立的 `save_dir`，避免把台服存档当成 B 服初始状态
   这些是资源侧规划值，尚不是可直接运行并完成 B 服登录的配置
5. 核实 B 服脚本加载、协议与渠道登录后，确定本地测试客户端和本地服务的对接方式
   重建 B 服协议映射及最小登录证据，逐项对比现有 Go 输入、回包和状态变化
   原有 Python／Go 测试仍用于旧路径回归，新 B 服预期需来自 B 服证据
6. 客户端确需通过 HTTP 获取本地资源时，再按 B 服实际请求路径接入静态资源服务
   现有 `internal/server/login.go` 的 getResource 仅返回 `/gameres/` 地址，manifest 分支返回空清单
   `internal/server/server.go:HTTPHandler` 将 HTTP 请求交给 LoginHandler，未提供实际媒体文件服务
   单改资源目录不能替代这部分客户端对接工作

## 证据定位与实际验证

| 结论 | 状态 | 来源与验证 |
|---|---|---|
| 后续以此次 B 服为准 | 维护者决定 | 2026-10-25 对话确认 |
| 备份前可以正常进入游戏 | 维护者确认 | 本轮追问答复，未在本机复现 |
| 容器可读且有 APK、缓存和子包 | 已验证 | 文件头、全部 TAR 目录、嵌套 APK ZIP 目录 |
| 四份 project 清单内容完整 | 已验证 | 59,599 项逐项读取，大小及 MD5 全部一致 |
| 版本元数据不一致 | 已验证，原因待验证 | APK、主更新及子包各 manifest 的内存解析 |
| Go 必需配置可解析 | 已验证 | 33 张表、433 个配置文件，无子表合并错误 |
| B 服登录／游戏协议兼容 | 待验证 | 未解出更新主脚本，未执行客户端联调 |

关键原始文件 SHA-256：

```text
f/remote-asset/project.manifest
5ac5a2a61570a5c6a823830b659e91b114ff5aeb81c947a929aae6edd509c60c

f/remote-asset/assets/resources/config.json
0bae810fdd13117ce1501bc1f4dc01b96cec9b86833022e466b810acd32195b2

APK: assets/assets/resources/config.json
06ba27ae21334175e88cb308dbd350b3af4163b08c3fefeb480b787be69c50c6
```

执行工具为工作区运行时自带的 Python，以 `-B -` 运行只读内存脚本，使用 tarfile、zipfile、hashlib、json 和既有解码原语
TAR/ZIP 未整体展开；没有安装第三方工具或新增依赖，没有连接游戏正式服务，没有读取私有数据库或偏好文件的内容
本轮仅新增这份分析文档，未修改原始备份、Go 生产代码、运行配置、现有存档或旧报告
未运行 Go 构建、测试、vet、race 或重新生成 Python 差分：本轮没有修改程序，也没有完成 B 服运行时接入
