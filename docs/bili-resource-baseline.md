# B 服资源基线落地记录

日期：2026-10-25

状态：第一步资源整理已完成，导出后的资源和原始版本元数据均通过独立校验

后续的更新主脚本解码、登录接口、编号分表、网络加密及加载顺序已完成离线分析，见 [B 服脚本与协议证据](bili-script-analysis.md)
本文的“下一步”保留资源整理完成时的阶段状态

## 范围和工具

维护者已授权开始第一步资源整理，并要求优先使用现成工具

本次使用本机既有工具完成工作，没有新增自定义解包程序或项目依赖：

| 工具 | 本机版本 | 实际用途 |
|---|---|---|
| GNU tail | coreutils 8.32 | 从已确认的第 74 字节偏移读取未压缩 TAR 流 |
| Windows tar | bsdtar / libarchive 3.5.2 | 检查目录、按白名单导出备份、展开 APK |
| PowerShell | 7.6.5 | ConvertFrom-Json 读取原清单，Get-FileHash 选择内容匹配的文件，Copy-Item 复制，Export-Csv 记录来源 |
| GNU md5sum | coreutils 8.32 | 独立读取目标目录，复核标准校验清单 |
| dos2unix | 7.5.7 | 将生成的校验清单转换为 LF 换行 |

tail、tar 和 md5sum 负责现有格式处理；游戏清单的文件选择由一次性 PowerShell 命令串联
没有安装 ABE、7-Zip 或其他下载工具，也没有编写新的 Python／Go 解包器

工具选择参考了 libarchive 官方站点和源码文档、GNU coreutils 文档及 Microsoft PowerShell 字节管道文档
本机工具已能处理这份未压缩、未加密的备份，因此本次未更新工具版本

## 来源和目录

B 服包名为 `com.bilibili.gxjy`，目标为原始 281 project 清单对应的资源内容
原始备份保持原状，工作目录为 `gxjy-server/data/assets/bili/`

```text
bili/
├── README.md
├── source/
│   ├── backup/
│   │   ├── a/base.apk
│   │   ├── r/lebian/cacheFiles/
│   │   └── f/
│   │       ├── remote-asset/
│   │       └── subBundles/
│   └── apk/
├── resolved/
│   ├── assets/
│   ├── subBundles/
│   ├── project.manifest
│   └── version.manifest
└── reports/
```

备份 SHA-256：

```text
43c52a31c65629f13fe78733668bdf57ab786da03bccec84b273e8ad056888a5
```

导出的原始 APK SHA-256：

```text
d5cc33f3406d10198d08b0de7a2086f2aa68af9bb6a537f9ab80133c3c2333ee
```

按白名单展开的备份源目录有 13,226 个文件、8,074,338,662 字节
APK 展开后有 55,013 个文件、3,242,843,149 字节
两个 tar 操作均成功退出，错误日志均为空

## 导出依据

1. 重新核对备份大小、文件头和 TAR 偏移，再用 `tail -c +75` 将二进制流交给 tar
2. tar 白名单只包含原始 APK、两层乐变缓存、主更新目录及三个子包
3. 导出前检查所选 TAR 和 APK 条目，未发现符号链接、硬链接或上级目录跳转路径
4. 使用新建目录和 tar 的 `-k`，保留原始备份路径关系，未覆盖已有工作区资源
5. 读取四份原始 project 清单，确认版本、59,599 个唯一目标路径及 6,338,442,854 字节的总量
6. 每个源文件必须同时符合清单的大小和 MD5，再复制到 `resolved`
7. 主资源候选检索顺序为 remote、cache102、cache101、APK；该顺序用于寻找匹配内容，不作为客户端搜索顺序的协议结论
8. 额外保留主资源和三个子包的 8 份 project/version 清单原始字节

没有导出数据库、偏好或 WebView 缓存区域
没有更改主清单的 `1.1.281.1.1`、版本清单的 `1.1.279.1.1` 或 `updating` 字段

## 验收记录

第一次独立检查返回退出码 1：本机 GNU md5sum 把校验清单的 CRLF 中的 CR 当成文件名的一部分，日志显示文件名末尾的 `\r` 导致打开失败
已使用本机 dos2unix 将两个生成的 `.md5` 清单转换为 LF，再重跑全量检查
该修正只涉及校验清单的换行，未改变资源文件或原始版本清单的内容

最终检查结果：

| 范围 | 文件数 | 字节数 | md5sum 退出码 |
|---|---:|---:|---:|
| 清单内资源 | 59,599 | 6,338,442,854 | 0 |
| 原始版本元数据 | 8 | 10,317,725 | 0 |
| resolved 全目录 | 59,607 | 6,348,760,579 | 两组均通过 |

逐项 MD5 通过后，再次枚举目标目录确认文件总数和总字节数与预期一致
没有缺失项或内容不一致项，最终校验的标准输出和错误输出日志均为空
第一次换行格式失败的日志也保留在 reports 中，没有将该次失败记为通过
本地机器可读结果已保存到 `data/assets/bili/reports/verification.json`，状态为 `verified`

每项资源选中的匹配来源：

| 来源 | 文件数 |
|---|---:|
| APK | 46,619 |
| remote-asset | 3,038 |
| 乐变缓存 -102 | 438 |
| 乐变缓存 -101 | 5,902 |
| heroVideo | 130 |
| live2d | 2,806 |
| CVSound | 666 |

复核入口位于本地 `data/assets/bili/README.md`
使用已保存的标准清单，无需重新实现哈希检查：

```powershell
Set-Location 'E:\Go Project\gxjy\gxjy-server\data\assets\bili\resolved'
& 'G:\Git\usr\bin\md5sum.exe' --check --strict --quiet '..\reports\resolved.md5'
& 'G:\Git\usr\bin\md5sum.exe' --check --strict --quiet '..\reports\metadata.md5'
```

两条命令分别检查资源和版本元数据，均应以退出码 0 结束

`resource-sources.csv` 记录每项资源的清单分组、目标路径、源层、源路径、大小和 MD5
`metadata-sources.csv` 记录原始清单的来源与校验和
它们位于已忽略的本地 reports 目录，不将完整资源索引加入 Git

## 下一步和边界

资源加载器后续应使用：

```text
asset_root:    data/assets/bili/resolved/assets
bundle_config: data/assets/bili/resolved/assets/resources/config.json
```

这些路径尚未写入现有运行配置
本次没有启动服务、创建玩家存档、恢复手机备份、修改客户端或执行 Go 运行时测试
文件完整性不能代替 B 服主脚本、协议和登录流程的验证
下一步仍是核实 B 服更新主脚本的加载／解码方式

Git 变更仅涉及本次记录及原始备份的忽略规则
新增 `/20261019_213556/` 忽略项，保留原有规则；全部导出内容沿用已有 `/gxjy-server/data/assets/` 忽略项
没有提交或推送，本次未修改 Go 运行配置、存档或其他任务文件
