# 运行数据目录

此目录只存放本地运行所需的数据。

```text
data/
├── assets/       # 登录业务必需的授权配置，也可以指向外部目录
├── keys/         # 可选：与测试客户端匹配的 RSA 私钥
├── protocol/     # 可选：完整 protocol_map.json
├── db/           # 自动创建：SQLite 数据库
└── save/         # 历史 JSON 存档，仅供已有离线工具使用
```

当前本机配置使用本目录下的资源、RSA 和协议表本地副本。

资源来自本地发布包 `2-SERVER/apk/assets/assets`，RSA 文件来自 `2-SERVER/tools/gxjy-gate-key.pem` 和 `2-SERVER/protocol/server_rsa.json`，协议表来自 `2-SERVER/protocol/protocol_map.json`。

本次仅复制配置引用的数据，原发布包保留；`assets/`、`keys/` 和 `protocol/` 均由 Git 忽略规则排除。

`keys/asset.key` 保存从本地 `2-SERVER/gxjy_decrypt.py` 的 `KEY` 常量迁移的资源密钥，不加入 Git。Go 启动入口在未设置 `asset_key_env` 指定的环境变量时读取 `asset_key_file` 指定的文件，默认分别为 `GXJY_ASSET_KEY` 和 `data/keys/asset.key`，可在项目根目录使用 `go run .` 启动。

官方加密客户端需要匹配的 RSA 私钥。

真实配置表需要已获授权的资源，以及环境变量或本地密钥文件提供的资源密钥。

本地真实差分结果保存在 `assets/diff/`，由现有忽略规则排除，不能加入 Git。

不要提交私钥、资源密钥、APK 或真实玩家存档。
