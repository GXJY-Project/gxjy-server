# B 服启动“无网络”与连接日志排查

日期：2026-10-25

最终状态：维护者授权修改自有本地测试客户端后，已部署本地登录地址及匹配公钥的脚本补丁，维护者确认登录成功
下文最初排查结论保留为历史阶段，实际修复与更新后回退原因见文末

## 结论与范围

本轮 B 服客户端仍访问原官方登录域名，DNS 解析失败，未切换到本地 HTTP 登录入口
模拟器通过现有 ADB reverse 能访问本机 8080 与 8765，当前失败首先发生在客户端资源发现请求
服务端资源和游戏协议的离线对齐不会修改客户端的 `G_Config.url`

状态：已验证当前请求目标、DNS 失败及本地端口可达性
网络错误界面与该失败路径一致，未逐帧核对界面状态
完整 B 服 SDK 登录、RSA 握手、资源更新及业务兼容仍待实机验证

本轮只修改 Go 请求日志和回归测试，未修改客户端、APK、模拟器配置、正式服务配置或玩家存档
未向官方登录地址、SDK 或 CDN 发起测试请求

## 证据

客户端包名为 `com.bilibili.gxjy`，包版本 `1.1.224.1`，当前进程可见
读取该进程日志后只保留错误类型、目标域名、计数和时间，未导出原始日志、请求正文、账号或令牌

| 检查 | 实际结果 |
|---|---|
| 本机 GET `/healthz` | HTTP 200，`status=ok` |
| `adb -s emulator-5554 reverse --list` | 已存在 `tcp:8080 → tcp:8080` 与 `tcp:8765 → tcp:8765` |
| 模拟器 GET `http://127.0.0.1:8080/healthz` | 返回 `status=ok` |
| 模拟器普通 HTTP GET `http://127.0.0.1:8765/game` | HTTP 400，与未带 WebSocket 升级头的现有处理一致，只证明端口可达 |
| B 服进程日志，22:54:25–22:54:41 | 两个官方登录域名各出现 8 次 `UnknownHostException`，并有 `Response failed, error buffer: connect failed` |
| 模拟器 hosts 与全局代理 | hosts 中未发现 B 服登录域名映射，全局 `http_proxy=null` |

日志中的两个目标域名为：

```text
l1-ob-bili-login-klight.bilibiligame.net
l2-ob-bili-login-klight.bilibiligame.net
```

源码依据沿用已授权且固定 SHA-256 的更新脚本 `data/assets/bili/analysis/verified/index.js`
来源与解码过程见 [B 服脚本与协议证据](bili-script-analysis.md)

| 位置 | 已确认行为 |
|---|---|
| `devConfig`，第 476747 行 | `G_Config.url` 是上述两个官方域名的 HTTPS 根地址 |
| `LoginModel.getUrl`，第 341449 行 | 从 `G_Config.url` 取得根地址，再拼接请求路径 |
| `LoginModel.requestLogin`，第 341989 行 | POST `player/getResource`，当前根地址失败后依次重试；全部失败后调用 `checkNetStatus` |
| `LoginModel.checkNetStatus`，第 341691 行 | 打开使用 `Common_UI_NetWorkTip`、`Common_UI_NetWorkError` 的网络错误弹窗 |

当前 B 服进程的实际失败域名与源码的默认地址一致
未假设已安装客户端的脚本内容与本地证据文件逐字相同，也未从模拟器导出客户端脚本进行哈希核对

## 请求日志改动

之前 `App.HTTPHandler` 直接返回 `LoginHandler`，没有 HTTP 请求到达日志
游戏连接只在升级失败、读取首条消息或会话结束时记录，成功升级后尚未收到消息时也看不到建连日志
因此旧版本“无连接日志”本身不能证明所有请求均未到达

现在 HTTP 入口在读取正文前记录方法和路径，WebSocket 入口在路由判断前记录方法和路径，升级成功后另记连接已建立
日志不记录查询参数、正文、请求头、账号或设备标识，路径使用带转义的引号格式避免控制字符污染日志
登录回包、状态码、升级流程及游戏消息处理保持原有行为
重新编译并重启服务后生效

回归测试 `internal/server/request_log_test.go` 覆盖资源请求、带查询参数的网关请求、超限正文、错误 WebSocket 路径与缺少升级头
同时检查原状态码和认证信息不进入日志

## 待授权的客户端修改方案

针对维护者自有的本地 B 服测试客户端，将实际生效的 `devConfig` 登录根地址替换为：

```javascript
G_Config.url = [ "http://127.0.0.1:8080/" ];
```

这与当前模拟器已经存在的 8080 reverse 映射一致
三个登录请求仍使用原 `LoginModel` 构造的路径、方法和字段
仅把官方域名映射到本机不能完成本方案，因为原根地址是 HTTPS，而本地登录服务是 8080 上的 HTTP

排查开始时正式配置的 `gate_host=127.0.0.1`，结束检查时已变为 `0.0.0.0`
这处变更不是本轮 Agent 写入，修改来源及意图尚未确认，因此保留现场并向维护者协调
`LoginHandler.response` 将 `gate_host` 返回为顶层 `gate_ip`，B 服 `LoginServer.synchronize` 用它拼接 WebSocket 地址
按本轮已验证的 loopback reverse 方案，客户端目标应为 `127.0.0.1`，监听配置 `http_host`、`ws_host` 可以继续为 `0.0.0.0`
该配置问题发生在取得网关后的阶段，不改变本轮已经确认的官方域名解析失败原因

更新目录先于 APK 搜索，需确认实际加载脚本，避免只改 APK 而被更新目录的原脚本覆盖
已有资源基线记录不等于已修改或安装这份测试客户端

地址修复后还需验证渠道 SDK 登录
进入网关后，既有证据表明客户端 RSA 公钥与本地服务的两份密钥均不匹配；后续应使用与本地已授权私钥匹配的测试公钥，保留 RSA 和 CC2P 校验
公钥匹配问题与本轮 DNS 失败是两个阶段，不能把改地址描述为已经完成游戏登录

根据根目录 `AGENTS.md` 的第三方材料边界，以及既有 B 服接入记录的客户端改动授权要求，本轮没有执行客户端地址或公钥修改
正式执行需要维护者授权自有测试客户端的修改和安装范围，原始备份保留

## 实际验证

修改前与修改后均执行 `go test ./internal/server -count=1`，通过
修改后首次测试因沙箱不能访问工作区外 Go 构建缓存而失败，按授权重跑后通过

| 命令或检查 | 结果 |
|---|---|
| 对 `rg --files -g '*.go'` 找到的 Go 源码执行 `gofmt -l` | 通过，无未格式化文件 |
| `scripts/check.ps1` | 失败，枚举旧的 `data/assets/bili/analysis/runtime-validation/run-bakqdz_h` 时被目录权限拒绝，未执行到完整检查 |
| `go test ./...` | 失败，匹配目录时遇到同一访问限制 |
| `go test ./cmd/... ./internal/...` | 全部 Go 源码包通过，未使用 `-count=1` 强制重跑的包可能采用 Go 测试缓存 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go build ./cmd/... ./internal/...` | 执行授权被拒绝，未运行 |

源码包目录依据仓库实际 Go 文件位置，未修改旧验证目录权限或忽略有效测试
本轮没有修改并发逻辑，未运行 race

本轮没有启动第二份游戏服务、触发真实玩家登录或改动存档
客户端地址、公钥修改和完整 B 服实机登录仍未执行

## 授权后的修复与实机结果

维护者明确授权修改并安装本地 B 服测试客户端，以及恢复 `gate_host=127.0.0.1`
恢复检查时配置已是 `127.0.0.1`，运行中的 GET `/player/enter` 也返回 `gate_ip=127.0.0.1、port=8765、type=ws`，无需再次写入配置

实际安装客户端的 APK 主脚本与本地授权底包副本 SHA-256 一致：

```text
65fec07161fa8c7a73b0dc973d4bb728bdc57ebb620b4ec696c9aa030dc50520
```

更新目录当时没有主脚本，因此本次以实际安装底包的 `1.1.224.1` 脚本生成补丁，未混入本地 281 更新主脚本或其他资源
补丁只修改登录根地址为 `http://127.0.0.1:8080/`、添加该地址的启动标记，以及替换与运行配置私钥匹配的测试公钥
RSA 位宽、PKCS#1 v1.5 padding、CC2P、消息编号、请求字段及业务流程保持原脚本行为

工具和验证：

- `scripts/patch_bili_test_client.py` 固定原生库、底包 JSC、原型算法和原公钥版本，仅复用原型既有编解码函数
- 解密后只有上述替换点变化，反向替换可恢复原始源码
- 压缩及 XXTEA 重新编码后再解码，结果与补丁源码逐字相同，`node --check` 通过
- `scripts/verify_bili_test_client.cjs` 离线执行原 `devConfig`、`LoginModel.getUrl` 与 `data_pack`，验证三个登录接口的本地路径、测试公钥、AUTH 101 帧和 CC2P 标签
- 原客户端实际生成的合成 RSA 区块可由运行配置中的私钥解密，密钥内容未写入报告或日志
- `scripts/deploy_bili_test_client.py` 在写入前核对实际安装脚本、补丁哈希、目标不存在及应用所有者，写入后再次核对设备文件哈希

补丁生成、认证验证和部署记录只保存在已忽略的 `data/assets/bili/analysis/local-client/base224-local/`
原始 APK、原始备份和已有应用数据均保留，未重签 APK 或卸载应用
实际部署目标为：

```text
/data/user/0/com.bilibili.gxjy/files/remote-asset/assets/main/index.jsc
```

### 0.01 MB 更新为什么导致再次“无网络”

维护者观察到补丁首次启动已识别本地服务，但提示更新约 0.01 MB，更新后再次出现网络错误
本轮设备检查证实：首次补丁运行后，目标 JSC 被更新流程删除，更新目录只剩 `project.manifest` 等元数据
落地 project 的 `version=1.0.0、assets={}` 与服务端 `LoginHandler.response` 的 manifest 分支一致，另有 `updating=true`
原 `Updater.initAssetManager` 的比较器对任意不同版本都返回 `-1`，不是按版本大小区分升级与降级
因此不能根据服务端 `1.0.0` 版本更低就断定客户端不会更新

删除更新主脚本后，后续启动回退到 APK 原脚本，再次请求官方域名并出现 DNS 失败
这条现场证据修正了最初“只需改地址”的范围：补丁还必须在更新流程后保留下来
本轮没有修改 Go 的 manifest 响应，也没有禁用客户端更新逻辑
在本次空清单更新已落地后，恢复同一份已验证补丁并重启，当前本地和服务端清单版本一致
恢复后日志确认本地登录启动标记，目标 JSC 哈希仍为：

```text
4a8dc775ee4093f6c76142c379bc80409f21c9395e05cf39ecc8247f39fc8c53
```

维护者随后明确确认“登录成功了”
状态：本次客户端登录已获维护者实机确认；完整 SDK 行为、全部业务与长期资源更新兼容未验收

### 使用边界和恢复

当前是模拟器应用数据目录中的测试补丁，原 APK 未改变
清除应用数据、重装原 APK、修复缓存或后续资源更新可能删除／覆盖该补丁，不能将本次成功视为可重装的完整发行包
服务端空 manifest 的长期兼容方案仍待单独核实和维护者决定，未将临时部署描述为正式更新服务
部署工具不会覆盖来源不明的更新主脚本，重复运行已有部署会拒绝写入
`--restore-after-update` 只在首次部署记录、设备和补丁哈希一致且目标确实已消失时允许恢复一次

Windows computer-use 助手因 native pipe 不可用而无法操作，按规定重试及重置后仍失败
本轮改用正常可用的 ADB 安装脚本补丁、重启客户端、只读日志和 Android 截图确认网络错误界面，不执行账号登录界面的输入
最终登录结果采用维护者确认，没有自动填写真实账号凭据

本轮未提交、推送或发布，也未改写原有资源、私钥、协议表或玩家存档
