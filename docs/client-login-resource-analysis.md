# 2026-10-25 登录加载与本地资源核对

本记录补充根目录 `client-log-analysis.md`，保留原报告不变
本轮只做日志、代码、资源索引和 APK 目录核对，没有修改游戏协议、客户端、存档或资源

## 结论与边界

- 维护者确认：10:44:39 是因加载／登录停滞而手动退出，不是游戏自动返回桌面
- 服务端 10:44:38 收到 101，并依次写出 11 个登录消息；10:44:40 才读到连接异常结束
- 这段记录不能证明 1006 是加载问题的起因，也不能证明客户端已成功解密和处理全部下行消息
- 原始日志确有资源下载失败：120 条 `retCode=404`，13 个不同文件最终出现 `auto dload failed, skip`
- 13 个文件的同 UUID 描述 JSON 均存在，但对应原生文件未在本次搜索范围找到
- APK 中存在同系列背景、音频等资源，尚无证据证明它们可以直接替换缺失文件
- 尚未进行同一客户端的 Python／Go 实机对照；“Python 应该能进入主界面”是维护者预期，不是本轮验收结果

## 可追溯证据

原始日志：`gxjy-release-20261019/client-log.txt`，继续保留在已忽略的发布目录内

| 证据位置 | 观察 | 状态 |
|---|---|---|
| 日志第 370–371 行 | 10:44:28.950 已有下载 404，请求主机为 `www.g.cn`，端口 80，路径前缀为 `/gameres/trialres/801012/3279/` | 已验证 |
| 第 8237–8239 行 | 10:44:38.463，图片下载 404 后记录 `auto dload failed, skip` | 已验证 |
| 第 9015、9018、9025 行 | 10:44:39.062 返回桌面，随后 onPause、失去焦点 | 已验证，手动操作由维护者确认 |
| 第 9132 行 | 10:44:39.701 出现 `memoryWarning` | 已验证，但不能据此认定 OOM |
| 第 9199 行 | 10:44:41.387，进程因 `remove task` 被移除 | 已验证，发生在服务端 EOF 之后 |
| 第 9703–9704 行 | 第二次启动时缓存路径的 project.json 打开失败，并出现 serviceClassPath 缺失异常 | 已验证，两者因果仍需区分 |
| APK 的 `assets/project.json` | 文件存在，可解析为 JSON，但本身不含 serviceClassPath | 已验证，不能承诺修复缓存就会消除该异常 |

按游戏主进程分别统计：

| PID | 文件打开失败 | 下载 404 | 最终跳过的文件 | memoryWarning |
|---|---:|---:|---:|---:|
| 1861 | 8563 | 56 | 6 | 1 |
| 2353 | 10100 | 64 | 7 | 0 |

这里只统计这两个 PID，不将其他进程的错误混入
120 条下载 URL 均指向上述主机；没有根据域名推断实际服务器或 DNS／代理配置，也没有向该地址发起请求

## 搜索范围与资源映射

搜索了工作区文件名、标准 UUID 和 Cocos 压缩 UUID，并检查以下压缩文件目录：

- `gxjy-release-20261019.zip`
- `gxjy-release-20261019/gxjy-release-20261019/1-APK/gxjy_probe29.apk`
- `gxjy-release-20261019/gxjy-release-20261019/3-CLIENT-SRC/unsigned_probe30.apk`

两个 APK 的 `assets/assets/resources/native/` 中各有 4552 个文件
Go 的 `data/assets/resources/` 和 Python 的 `2-SERVER/apk/assets/assets/resources/` 都只有 config.json 和 import 描述目录，本次均未发现 native 目录
因此服务端配置表完整，不等于客户端媒体资源完整

以下逻辑名来自本地 `resources/config.json` 的 paths／uuids 映射
容器读取复用了 Python 原型 `read_config_table.unpack_ref` 与 `decode_uuid`，只在内存解析描述，不导出媒体或密钥

| 缺失的原生文件 | 资源逻辑名 |
|---|---|
| `01c08001-4e31-43fd-8b6a-6fd1ca1be744.png` | `spineEffect/battle/E903_Special/E903_Special` |
| `0161b271-9e4d-4648-8631-2cb8957d736e.png` | `spineEffect/battle/E903_Special/E903_Special2` |
| `52755549-069f-4427-a930-32cdb474a3ec.png` | `spineEffect/battle/E903_Special/E903_Special3` |
| `775579a5-fba2-4dba-8716-3fef38e9c1fe.bin` | `spineEffect/battle/E903_Special/E903_Special` |
| `2bead7df-aa9a-43fd-af91-a8c971154ef0.png` | `spine/EBoss03a_Model/EBoss03a_Model` |
| `f444981d-9e48-48ea-8ee3-106acb54c770.bin` | `spine/EBoss03a_Model/EBoss03a_Model` |
| `1b6f417d-e0b7-427b-981a-c7b668b05a60.jpg` | `scene/UpData004` |
| `232849aa-1a63-40bd-b549-be5e0ada731d.jpg` | `scene/storybg_cg_01_2` |
| `6d1c95a5-a873-4ea1-b246-aa270a3f6710.jpg` | `scene/cg-02` |
| `49244ce1-a1bf-43f0-9d05-32cfc6126b82.jpg` | `scene/exploreWorldMap/MapPic_1_2_3/MapPic_1_2_3` |
| `8982ffde-2693-40af-96b1-a6da58c20b4b.png` | `ui/exploreNote/res/ENote_NC1_005_Bgp` |
| `39759035-bbbb-4be9-9ccd-07a7d26559a2.acb` | `sound/Mus_Theme_Awaken` |
| `e60320e4-6633-4b40-85ba-197c679993ef.usm` | `video/JQ_huaping` |

这些都是日志中最终跳过的样本，不代表全部缺失资源清单
图像描述是 cc.Texture2D；两个 .bin 的描述是 sp.SkeletonData，并引用表中的配套纹理
13 个同 UUID 的 import JSON 在两个服务端资源副本里各有一份，不能把描述 JSON 改后缀作为原生资源使用

## 已找到的相近文件

下表文件均确认存在于 probe29 APK 中
原生路径统一为 `assets/assets/resources/native/<UUID 前两位>/<文件名>`
“相近”仅指同系列逻辑名／资源类型，没有验证视觉、尺寸、音频 cue 或动画兼容性

| 缺失项／系列 | 已有逻辑名 | 包内文件名 | 字节数 |
|---|---|---|---:|
| UpData004 背景 | `scene/UpData003` | `67f2eb0b-c20f-4b7e-b211-9af927395687.jpg` | 462484 |
| UpData004 背景 | `scene/UpData006` | `08a08edc-fac5-4d32-bfad-f7d4bd1e3b2a.jpg` | 669628 |
| storybg_cg_01_2 背景 | `scene/storybg_cg_01_4` | `600aa3d0-7cac-4d50-95de-e0b037c2278d.jpg` | 669628 |
| ENote_NC1_005_Bgp 背景 | `ui/exploreNote/res/ENote_NC1_006_Bgp` | `fed2c7ef-1bfa-4c5b-a05e-7d60a7cdc0b9.png` | 88732 |
| ENote_NC1_005_Bgp 背景 | `ui/exploreNote/res/ENote_NC1_007_Bgp` | `96b63a22-280c-4a61-be3f-85a880008314.png` | 88732 |
| ENote_NC1_005_Bgp 背景 | `ui/exploreNote/res/ENote_NC1_008_Bgp` | `4e4bd81c-ab46-4f54-ac96-61da92f9f123.png` | 88732 |
| Mus_Theme_Awaken 音频 | `sound/Mus_Theme_Awaken_Musiclist` | `01e8a481-71cf-4bd8-b078-036d163d2877.acb` | 1934272 |
| Mus_Theme_Awaken 音频 | `sound/Mus_Theme_Awaken_03_Musiclist` | `9059f29b-cc06-4eb4-b865-591fac01d183.acb` | 923552 |
| E903_Special 的图集描述 | `spineEffect/battle/E903_Special/E903_Special` | `2dfc4fc0-7a03-413d-8f4f-7324d93e1881.atlas` | 6312 |
| EBoss03a_Model 的图集描述 | `spine/EBoss03a_Model/EBoss03a_Model` | `89ced331-8d0b-4cba-a38f-a5c7724cc52f.atlas` | 4492 |

图集描述仍然需要对应纹理和骨骼数据，不能作为缺失 .png／.bin 的替代品
同系列音频包的存在不能证明包含原包使用的 cue，直接重命名尚无兼容依据
本轮没有进行任何替换，也未将“相近文件存在”标记为问题已修复

## 服务端与加载链路核对

- `internal/server/session.go:29`：101 解包后进入业务分派；读取下一条 WebSocket 消息时可以返回 EOF 类错误
- `internal/server/session.go:104`：发送日志在 writeBinary 成功后记录，只证明该次写入未报错
- 本地 gorilla/websocket v1.5.3 的 conn.go:178、370 将底层 EOF 表示为带 1006 的 unexpected EOF，不能据此定位游戏业务错误
- `internal/game/runtime.go:222`：登录推送顺序与原型 `gxjy_server.py` 的 Session.dispatch、send_login_info、send_login_pushes 一致
- `internal/server/login.go:57`：getResource 回显版本并返回本地 `/gameres/` 地址，没有静态资源处理器
- 原型 `gxjy_server.py:122`、`:243` 同样使用版本回显、空 manifest 和 JSON 响应，没有发现本次范围内遗漏迁移的媒体服务
- 客户端基线 `3-CLIENT-SRC/index.probe13.dec.js:250331` 的登录完成依赖英雄、登录快照、背包、客户端设置、技能脚本加载共五类事件，权重合计 9
- 同文件 `:250945` 的 requestEntranceInfo 会请求后续业务并加载技能脚本；`:139911` 的 loadAllSkillScript 加载的是 config/skills，不能直接把缺少战斗纹理等同于该事件失败

目前提供的服务端片段只有 101 入站，没有后续请求的客户端处理证据
已验证 Go 的现有路径不代表后续公会、聊天等尚未迁移请求均受支持，也不代表实机已完成登录

## 建议验证与修复次序

1. 同一客户端先保持前台完成一次观察，分别记录点击开始、认证、进度变化和第一条脚本异常／资源失败的时间
   发布包 README 的历史记录为点击开始后约 40 秒进主城，本次认证后约 1 秒就回到桌面；可先观察至少 60 秒，若进度仍变化则继续记录，不能把固定等待时间当作通过条件
2. 使用独立测试存档对照 Python 和 Go，同一时刻只连接一个测试服务，避免改动当前存档
   两端都卡时优先定位共同客户端／资源问题；只有 Go 卡时，抓取首个不同请求或响应再修协议
3. 核对乐变资源请求的地址来源和有效文件来源
   日志请求的是端口 80 的主机，而 Go getResource 返回的是配置的游戏 HTTP 端口；仅给 Go 添加静态路由不能证明这些请求会到达 Go
4. 正式资源修复需要已获授权且与版本匹配的原文件／完整资源副本，并确认下载路径、类型和内容均正确
   清缓存不会自动补回本地和下载源都缺失的文件，暂不建议把清缓存作为第一步
5. 若要用相近背景做诊断占位，需要先确定具体映射和回退方式，再由维护者确认这种临时客户端行为变化
   骨骼、纹理和音频还需要验证配套引用，不能根据文件扩展名直接替换

## 本轮实际验证

已执行，全部通过：

```text
go test ./internal/server ./internal/game ./internal/protocol ./internal/crypto -count=1
```

使用仓库默认合成 fixture；包含现有 Python fixture 的业务、HTTP、社区明文及官方 RSA/CC2P WebSocket 比较
没有重新启动原 Python 服务、连接真实客户端或加载玩家存档
另外完成日志统计、资源索引解析、三个压缩文件目录核对、相近文件大小及 SHA-256 计算
Python 元数据核对使用工作区自带解释器，禁用字节码缓存，未输出资源密钥

本轮只新增排查文档，没有改生产代码，因此未重新运行全仓格式化、构建、vet 或 race，也未重新生成真实资源差分
原始报告、发布包、密钥、资源、玩家存档及现有未提交修改均保留
