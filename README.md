# 4vpx

`4vpx` 是一个面向多用户使用的 `VLESS + TCP(raw) + REALITY + xtls-rprx-vision` 管理面板。

首版定位：

- 单节点
- 单管理员
- 按时间续费
- 用户通过专属只读链接查看各设备位配置
- 通过多个独立 `UUID` 实现设备位限制
- 业务数据使用 `SQLite` 单文件存储

## 功能概览

- 管理员后台管理用户、设备位、续费和系统参数
- 用户只读门户页展示到期时间和每设备位独立配置
- 自动渲染并发布本机 `Xray` 配置
- 支持 `JSON` 导出和导入备份

## 生产环境

最终生产环境建议按 `Rocky 9` 部署。

说明：

- `Go + SQLite + systemd` 在 `Rocky 9` 上可正常运行
- 首版默认通过 `systemd` 直接运行 `4vpx`
- 若让 `4vpx` 直接写入 `Xray` 配置并执行重载，建议先使用 root 运行 `4vpx` 服务，避免权限链过于复杂
- `Rocky 9` 自带仓库中的 `Go` 版本可能低于项目要求；一键脚本会自动安装可用的 `Go 1.23+`

当前推荐的生产入口方案为 `B`：

- `Xray` 继续占用公网 `443`
- `4vpx` 单独监听 `8443`，或仅监听本地 `127.0.0.1:8080`

## 协议固定项

本项目固定使用：

- `VLESS`
- `TCP(raw)`
- `REALITY`
- `xtls-rprx-vision`

不包含：

- `WS`
- `gRPC`
- `XHTTP`
- 自动支付
- 多节点调度
- 不可绕过的硬件指纹绑定

## 本地运行

本地默认按你的使用方式处理：

- 本地机器是 `macOS`
- 本地不安装 `Xray`
- 本地只运行 `4vpx` 面板和 `SQLite`
- `Xray` 相关校验与重载在本地默认跳过

```bash
cp .env.example .env.local
export $(grep -v '^#' .env.local | xargs)
go run ./cmd/4vpx
```

说明：

- `.env.example` 默认把 `XRAY_BIN` 留空，因此本地不会执行 `xray run -test`
- 本地仍会生成 `./generated/xray-config.json` 一类输出文件，方便你查看渲染结果
- 真正的 `Xray` 安装、配置写入和重载，统一放在 `Rocky 9` 生产机完成

后台地址：

- `/login`

用户只读地址：

- `/u/{token}`

## Rocky 9 部署

推荐只保留这一套部署口径：

- 系统：`Rocky 9`
- 协议入口：`Xray` 占用公网 `443`
- 后台入口：优先 `127.0.0.1:8080 + SSH` 隧道
- 不使用 `Nginx`

### 一键部署

```bash
sudo dnf install -y git
sudo git clone https://github.com/qiufeihai/4vpx.git /opt/4vpx
cd /opt/4vpx
sudo ./scripts/install-rocky9.sh
```

如果你只是想把源码放在家目录，也可以这样执行：

```bash
sudo dnf install -y git
git clone https://github.com/qiufeihai/4vpx.git ~/4vpx
cd ~/4vpx
sudo ./scripts/install-rocky9.sh
```

脚本会自动完成：

- 安装基础依赖
- 安装或升级 `Go`
- 安装或校验 `Xray`
- 交互生成 `/opt/4vpx/.env.local`
- 构建 `4vpx`
- 注册并启动 `xray` 与 `4vpx`
- 开放 `443/tcp` 与 `8443/tcp`

如果系统里已经有可用的 `Xray`，脚本会直接复用它；默认要求服务名为 `xray.service`。如果你的服务名不同，可以在执行时指定：

```bash
sudo XRAY_SERVICE_UNIT=your-xray.service ./scripts/install-rocky9.sh
```

如果现有 `XRAY_CONFIG_PATH` 指向的 `Xray` 配置本身已经包含 `REALITY` 入站，脚本会优先读取其中的 `dest`、`serverNames[0]`、`privateKey`、`shortIds[0]`，并自动推导 `publicKey` 作为交互默认值。

### 后续更新

首次部署完成后，日常更新建议使用轻量脚本，而不是重复执行首次安装脚本：

```bash
git pull
sudo ./scripts/update-rocky9.sh
```

这个更新脚本会：

- 同步项目文件到 `/opt/4vpx`
- 重新编译 `4vpx`
- 覆盖最新 `4vpx.service`
- 重启 `4vpx`

这个更新脚本不会：

- 重写 `/opt/4vpx/.env.local`
- 重新安装 `Go`
- 重新安装 `Xray`
- 重新生成 `REALITY` 参数
- 修改防火墙

如果这次更新涉及 `Xray` 配置渲染或你就是想顺手重载 `Xray`，可以这样执行：

```bash
sudo ./scripts/update-rocky9.sh --with-xray-reload
```

### 重置管理员

如果忘记后台账号密码，或者你就是想强制重置管理员凭据，可以直接执行：

如果服务器还没安装 `sqlite3` 命令行工具，先执行：

```bash
sudo dnf install -y sqlite
```

然后再执行：

```bash
sudo ./scripts/reset-admin-rocky9.sh <username> <password>
```

这个脚本会：

- 把新的 `ADMIN_USERNAME` 和 `ADMIN_PASSWORD` 写入 `/opt/4vpx/.env.local`
- 备份当前 SQLite 数据库
- 清空 `admins` 和 `admin_sessions`
- 重启 `4vpx`

这个脚本不会影响用户、设备位、续费记录等业务数据，但会让现有管理员登录态全部失效。

### 推荐填写

更安全的后台方式：

```env
APP_ADDR=127.0.0.1:8080
APP_BASE_URL=http://127.0.0.1:8080
SESSION_SECURE=false
```

如果你确实想直接开放后台：

```env
APP_ADDR=0.0.0.0:8443
APP_BASE_URL=http://你的IP或域名:8443
SESSION_SECURE=false
```

生产环境至少确认这些字段：

- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `SQLITE_PATH=/opt/4vpx/data/4vpx.db`
- `SERVER_ADDRESS`
- `SERVER_PORT=443`
- `REALITY_DEST`
- `REALITY_SERVER_NAME`
- `REALITY_PRIVATE_KEY`
- `REALITY_PUBLIC_KEY`
- `REALITY_SHORT_ID`
- `XRAY_CONFIG_PATH=/usr/local/etc/xray/config.json`
- `XRAY_BACKUP_PATH=/usr/local/etc/xray/config.json.bak`
- `XRAY_BIN=/usr/local/bin/xray`
- `XRAY_RELOAD_CMD=systemctl restart xray.service`

推荐目录：

- 项目根目录：`/opt/4vpx`
- 二进制：`/opt/4vpx/bin/4vpx`
- 环境文件：`/opt/4vpx/.env.local`
- 数据文件：`/opt/4vpx/data/4vpx.db`
- 生成配置：`/opt/4vpx/generated/`

### SSH 隧道访问后台

如果你把后台监听在 `127.0.0.1:8080`，本地这样访问：

```bash
ssh -L 8080:127.0.0.1:8080 root@your-server-ip
```

浏览器打开：

```text
http://127.0.0.1:8080/login
```

### 启动检查

```bash
sudo systemctl status 4vpx --no-pager
sudo systemctl status xray --no-pager
sudo ss -lntp | grep ':443'
```

如果你把后台开放到公网，再额外检查：

```bash
sudo ss -lntp | grep ':8443'
```

如果新增用户后看起来创建成功，但客户端仍然不通，优先再查这几项：

```bash
sudo journalctl -u 4vpx -n 100 --no-pager
sudo journalctl -u xray -n 100 --no-pager
grep -E '^(REALITY_PRIVATE_KEY|REALITY_PUBLIC_KEY|REALITY_SHORT_ID)=' /opt/4vpx/.env.local
sudo systemctl cat xray.service
ls -l /usr/local/etc/xray /usr/local/etc/xray/config.json /usr/local/etc/xray/config.json.bak
```

### 重要提醒

- 如果本机 `Xray` 已经直接监听公网 `443`，不要再让其他服务抢占同一个公网 `443`
- `4vpx` 当前是普通 `HTTP` 服务，不会自动提供 HTTPS
- 如果你把后台直接开放在 `8443`，这只是“HTTP 跑在 8443 端口”，不是 HTTPS
- 当前版本已补基础 `CSRF` 防护和持久化管理员会话，但仍更推荐把后台限制在 `127.0.0.1:8080`
- `REALITY_PRIVATE_KEY`、`REALITY_PUBLIC_KEY`、`REALITY_SHORT_ID` 不能为空
- `xray.service` 的 `User=` 必须与 `XRAY_CONFIG_PATH` 的目录权限、文件 owner/group/mode 匹配
- 如遇 `permission denied`，优先检查 `XRAY_CONFIG_PATH`、`XRAY_RELOAD_CMD`、`xray.service` 的 `User=` 和配置文件权限，不要先怀疑客户端模板

## 备份

备份方式有两种：

- 直接备份 `SQLite` 文件
- 在后台导出 `JSON`

建议至少备份：

- `data/4vpx.db`
- `.env.local`
- `generated/`

## 关键文件

- 启动入口：[main.go](file:///Users/qiufeihai/workspace/4vpx/cmd/4vpx/main.go)
- 路由装配：[router.go](file:///Users/qiufeihai/workspace/4vpx/internal/http/router.go)
- 协议渲染：[renderer.go](file:///Users/qiufeihai/workspace/4vpx/internal/xray/renderer.go)
- 配置发布：[runtime.go](file:///Users/qiufeihai/workspace/4vpx/internal/xray/runtime.go)
- Rocky 9 安装脚本：[install-rocky9.sh](file:///Users/qiufeihai/workspace/4vpx/scripts/install-rocky9.sh)
- Rocky 9 环境模板：[.env.rocky9.example](file:///Users/qiufeihai/workspace/4vpx/.env.rocky9.example)
- systemd 服务文件：[4vpx.service](file:///Users/qiufeihai/workspace/4vpx/deploy/4vpx.service)
