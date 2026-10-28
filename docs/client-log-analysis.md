# client-log.txt 日志分析报告（供错误处理）

## 一、日志概况

| 项目 | 内容 |
|---|---|
| 文件 | client-log.txt（约 4.97 MB，21024 行） |
| 格式 | Android logcat（含 `system` 与 `main` 两个 buffer） |
| 时间范围 | 2026-10-25 10:42:34 ~ 10:49:21（约 6.5 分钟） |
| 设备标识 | REDMI 2407FRK8EC（device=unicorn, brand=REDMI） |
| 分辨率 | 1600×900（横屏） |
| GPU | Adreno（加载 /vendor/lib/egl/libEGL_adreno.so，GLES 3.1，hostGLVersion 46） |
| 运行环境 | 疑似虚拟化/模拟环境：存在 virtio 虚拟磁盘（pci0000:00/…/virtio1/block/sda）、HostConnection、EGL_emulation 等特征；但游戏侧 SDK 反检测均未命中（Cocos2dxActivity 判定 isEmulator=false，bluestacks/mumu 检测均返回 0，乐变 runningOnVm=false） |
| 系统服务 | 无 Google Play 服务（GMS 缺失） |

## 二、核心进程

| PID | 进程 | 角色 |
|---|---|---|
| 343 | system_server | 系统核心服务 |
| 1861 / 2353 | com.komoegame.gxjytw | 游戏主进程（两次启动） |
| 1878 / 2372 | com.komoegame.gxjytw:lebian.base | 乐变（Lebian）基础进程 |
| 1921 / 2413 | com.komoegame.gxjytw:lebian.dns | 乐变后台服务/DNS 进程 |
| 2069 / 2550 | com.google.android.webview:sandboxed_process0 | WebView 沙箱进程 |
| 886 | com.android.launcher3 | 桌面启动器 |
| 558 | systemui 相关（WindowManagerShell） | 系统 UI |

**应用特征**：com.komoegame.gxjytw 是 Cocos2d-x JS 游戏（`org.cocos2dx.javascript.AppActivity`），集成了：
- 乐变热更新 SDK（`USE_LEBIAN=true`，从 `.lebiansdk/` 加载 assetsonly.jar / lbmust.jar / lbui.jar）
- CRIWARE 音频中间件（CRIWARE for CC）
- AIHelp 客服 SDK、Firebase Messaging
- 原生登录/协议页 `com.gs.android.menu.PreLoginContractsActivity`

日志中游戏主进程启动了两次（10:43:59 的 PID 1861、10:49:00 的 PID 2353），两次都伴随 lebian.base、lebian.dns、webview 沙箱进程一并拉起。

## 三、问题清单（按严重度排序）

### 问题 1（核心，E 级 18668 条）：乐变热更新缓存资源全部加载失败

- 现象：游戏启动后的资源加载阶段，引擎对乐变缓存目录内几乎所有资源文件报 `No such file or directory`。
- 分布：第一次启动（10:44 分钟段）8563 条；第二次启动（10:49 分钟段）10105 条。两次均在启动加载阶段爆发。
- 原文样例：
  ```
  10-25 10:44:28.810 E/omoegame.gxjytw( 1861): Failed to open file '/data/user/0/com.komoegame.gxjytw/lebian/cacheFiles/com.komoegame.gxjytw-3279/assets/assets/resources/native/5d/5df1211d-ce59-450a-8ef1-ea02f5f75abb.skill': No such file or directory
  10-25 10:44:28.810 E/omoegame.gxjytw( 1861): Failed to open file '/data/user/0/com.komoegame.gxjytw/lebian/cacheFiles/com.komoegame.gxjytw-3279/assets/assets/resources/import/5b/5bb154ca-00db-4576-b81c-b5a725c635cf.json': No such file or directory
  ```
- 受影响文件类型：`project.json`、`project.manifest`、`remote.json`、`assets/assets/resources/import/<xx>/<uuid>.json`、`assets/assets/resources/native/<xx>/<uuid>.skill` 等，全部指向缓存目录 `com.komoegame.gxjytw-3279`。
- 疑点：
  - 缓存目录 `lebian/cacheFiles/com.komoegame.gxjytw-3279/` 是否存在/是否完整（热更新包可能未下载完成或已损坏）；
  - 乐变 VFS 资源重定向是否生效（若未生效，引擎会逐文件回退到 APK 内 assets，启动变慢且刷屏报错）；
  - 缓存版本号 3279 是否与服务端下发版本一致。

### 问题 2（P1，启动链路异常）：project.json 读取失败 → SDKWrapper 抛 JSONException

- 时间：10:49:02.900（第二次启动 onCreate 阶段）。
- 现象：先报 `Failed to open file '.../com.komoegame.gxjytw-3279/assets/project.json': No such file or directory`，随后 SDKWrapper 解析配置抛异常：
  ```
  10-25 10:49:02.900 W/System.err( 2353): org.json.JSONException: No value for serviceClassPath
  10-25 10:49:02.902 W/System.err( 2353): 	at org.json.JSONObject.get(JSONObject.java:398)
  10-25 10:49:02.902 W/System.err( 2353): 	at org.json.JSONObject.getJSONArray(JSONObject.java:593)
  10-25 10:49:02.902 W/System.err( 2353): 	at org.cocos2dx.javascript.SDKWrapper.loadSDKClass(Unknown Source:20)
  10-25 10:49:02.902 W/System.err( 2353): 	at org.cocos2dx.javascript.SDKWrapper.setGLSurfaceView(Unknown Source:2)
  10-25 10:49:02.902 W/System.err( 2353): 	at org.cocos2dx.javascript.AppActivity.onCreateView(Unknown Source:21)
  ...
  ```
- 影响：SDK 类列表未能按配置加载（可能影响登录、支付、统计等第三方 SDK 的初始化）。该异常为 W 级、未直接崩溃，游戏继续运行并显示了登录页（`PreLoginContractsActivity` +3s308ms）。
- 判断：与问题 1 同源——project.json 在缓存路径读不到，配置内容不完整导致 SDKWrapper 取 `serviceClassPath` 失败。

### 问题 3（P2，量少，疑似环境兼容噪音）：图形/显示层错误

| 时间 | 内容 | 说明 |
|---|---|---|
| 10:44:39 / 10:49:01 | `E/EGL_emulation: eglSurfaceAttrib(1504): error 0x3009 (EGL_BAD_MATCH)` + `E/OpenGLRenderer: Unable to match the desired swap behavior` | 发生在 launcher（PID 886）上，均与游戏前后台切换时刻重合，swap behavior 不匹配 |
| 10:44:39 / 10:49:02 | `E/libEGL: denglibo Loader::init_api redirect ... failed`（约 18 条） | 小米图形驱动加载器（denglibo）对 APPLE/EXT 扩展函数的重定向失败，system_server 与游戏进程均有 |
| 10:44:39 | `E/HostConnection: /data/local/cfg-hnegi/gl_force format error!` | 小米 GL 配置（gl_force）解析异常 |
| 10:45 前后 | `E/HWComposer: getSupportedContentTypes failed for display 0: Unsupported (8)` | 虚拟化环境下 HWC 能力缺失 |
| 10:45 前后 | `E/KernelCpuSpeedReader: Failed to read cpu-freq .../time_in_state: ENOENT` | 虚拟化环境无 cpufreq 统计 |
| — | `E/eglCodecCommon: glUtilsParamSize: unknow param 0x0000xxxx` | 模拟器 GL 栈常见噪音 |

这些错误与虚拟化 GPU 环境强相关，多数可视为噪音；但若出现黑屏、闪退、花屏，需回到这里排查。

### 问题 4（P3，可忽略噪音）：系统与第三方

- `ResilientAtomicFile: Failed to verity-protect package restrictions` / `Failed to enable fs-verity: Operation not supported on transport endpoint`（28 条）：文件系统不支持 fs-verity，系统常见提示，低风险。
- `VoiceInteractionManager: no available voice recognition services`：无语音识别服务，正常。
- `BatteryExternalStatsWorker: error reading Bluetooth stats: 11`：蓝牙统计读取失败，低风险。
- `wificond: Scan already started`：WiFi 扫描重入，正常噪音。
- GMS 缺失系列：`DynamiteModule: Failed to load IDynamiteLoader from GmsCore`、`FirebaseMessaging: Google Play services missing`、`ActivityThread: Failed to find provider info for com.google.android.gms.chimera`：设备无 GMS，若游戏依赖 GMS 推送/登录需注意。
- `dplm`（16 条，乐变下载管理模块）：`fileset count = 0`、`fileset count start =cfg.json / lbresinfo0 / lbstream ...`：乐变资源集统计日志，建议结合问题 1 核对。
- `CRIWARE for CC: Can not get hardware property`：音频中间件性能提示，非致命。
- `CellBroadcastUtils: no package found`、`BpTransactionCompletedListener: Failed to transact (-32)`、`ConnectivityService: ignoring attempt to change owner`：系统常规噪音。

## 四、关键事件时间线

| 时间 | 事件 |
|---|---|
| 10:43:59.367 | 游戏第一次启动：主进程 1861（next-top-activity）+ lebian.base 1878 + lebian.dns 1921 |
| 10:44:01~04 | webview 沙箱 2069、webview_service 2191 拉起 |
| 10:44:11.778 | lebian.dns (1921) 被系统回收（cch+5 CEM，缓存进程正常回收） |
| 10:44:28~39 | 资源加载错误爆发（8563 条，全部为缓存路径文件不存在） |
| 10:44:39.062 | 用户按 Home 键回桌面（launcher START，LAUNCH_SINGLE_TASK） |
| 10:44:39.077 / .176 | 游戏 onPause、失去焦点（hasFocus=false） |
| 10:44:41.385 | `ActivityManager: Killing 1861 ... (adj 900): remove task` —— 用户从最近任务列表移除游戏，进程被回收（非崩溃） |
| 10:44:41~10:49:00 | 间隙期：仅系统 WiFi/网络等杂项日志，游戏无活动 |
| 10:49:00.878 | 游戏第二次启动：主进程 2353 + lebian.base 2372 + lebian.dns 2413 + webview 2550 |
| 10:49:02.762 | AppActivity onCreate |
| 10:49:02.900 | project.json 打开失败 → JSONException: No value for serviceClassPath（SDKWrapper） |
| 10:49:02.907 | lebian.dns 收到 `com.excelliance.open.action.gameverchk`（游戏版本校验动作） |
| 10:49:02.996 → 03.517 → 04.597 | onResume → 短暂 onPause → 再次 onResume（疑似系统弹窗或登录页覆盖） |
| 10:49:04.636 | 登录/协议页 `PreLoginContractsActivity` 完成首帧（+3s308ms） |
| 10:49:13.150 / .183 | lebian.dns (2413) onDestroy 后被回收（cch+5 CEM，正常） |
| 10:49:21.632 | 日志结束（第二次启动的资源错误爆发仍在进行，共 10105 条） |

## 五、明确排除项（本日志中未发生）

- **无崩溃**：无 FATAL EXCEPTION / AndroidRuntime 崩溃栈。
- **无 ANR**：无 ANR 记录。
- **无内存问题**：无 lmkd / OOM 杀进程记录（`OomAdjuster: Not killing cached processes` 反复出现属正常）。
- **无 SELinux 拒绝**：avc denied 均为 `permissive=1`，不阻断。
- **进程结束均属正常**：lebian.dns 两次均为缓存回收；游戏主进程 1861 是被用户从最近任务移除（remove task）而结束，不是异常退出。

## 六、给处理 AI 的排查建议

1. **优先核对乐变热更新缓存完整性**：
   - 检查 `/data/user/0/com.komoegame.gxjytw/lebian/cacheFiles/` 下是否存在 `com.komoegame.gxjytw-3279/` 目录及 `assets/` 内容；
   - 若缓存缺失/不完整，重新触发资源下载或清缓存重试，观察 `dplm`/lebian.dns 的 `gameverchk` 版本校验结果（10:49:02.907 的 `com.excelliance.open.action.gameverchk` 是版本校验入口）；
   - 确认乐变 VFS 重定向是否生效——若未生效，引擎会对每个资源先查缓存再回退 APK，导致启动缓慢 + 刷屏 E 日志。

2. **处理 project.json 与 SDKWrapper 异常**：`serviceClassPath` 缺失根因是 project.json 在缓存路径读不到，修复问题 1 后应同步消失；若仍复现，检查 SDK 配置文件（assets/project.json 或服务端下发配置）中 `serviceClassPath` 字段是否存在。

3. **图形层错误按环境评估**：EGL_BAD_MATCH / denglibo / gl_force / HWComposer 错误与虚拟化 GPU 环境相关，建议在真机复现对比；若目标环境就是模拟器/云手机且无黑屏闪退，可暂不处理。

4. **GMS 缺失**：设备无 Google Play 服务，若游戏功能（推送/广告/登录）依赖 GMS，需确认是否走国内渠道包逻辑。

---

**验证说明**：以上统计基于对 client-log.txt 的逐行正则统计（级别分布、标签/PID 聚合、时间分段），关键结论均附日志原文行；未使用任何统计图表。
