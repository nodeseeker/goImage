# goImage 图床

基于 Go 语言开发的图片托管服务，使用 Telegram 作为存储后端。

## 功能特性
- 无限容量，上传图片到 Telegram 频道
- 轻量级要求，内存占用小于 10MB
- 支持管理员登录，查看上传记录和删除图片
- 提供 RESTful API 接口，支持第三方集成
- 包含独立的命令行客户端工具
- 支持跨域资源共享（CORS），可嵌入其他网站使用


## 页面展示
首页支持点击、拖拽或者剪贴板上传图片。

![首页](https://github.com/nodeseeker/goImage/blob/main/images/index.png?raw=true)

上传进度展示和后台处理显示。

![进度](https://github.com/nodeseeker/goImage/blob/main/images/home.png?raw=true)

登录页面，输入用户名和密码登录。

![登录](https://github.com/nodeseeker/goImage/blob/main/images/login.png?raw=true)

管理页面，查看访问统计和删除图片。`v0.1.5`版本新增了缩略图功能，以便快速检索和查找、管理等。
注意：删除操作为禁止访问图片，数据依旧存留在telegram频道中。

![管理](https://github.com/nodeseeker/goImage/blob/main/images/admin.png?raw=true)


## 前置准备

1. Telegram 准备工作：
   - 创建 Telegram Bot（通过 @BotFather）
   - 记录获取的 Bot Token
   - 创建一个频道用于存储图片
   - 将 Bot 添加为频道管理员
   - 获取频道的 Chat ID（可通过 @getidsbot 获取）

2. 系统要求：
   - 使用 Systemd 的 Linux 系统
   - 已安装并配置 Nginx
   - 域名已配置 SSL 证书（必需）

## 安装步骤

**注意文件名称和路径，以实际文件为准**

1. 创建服务目录：
```bash
sudo mkdir -p /opt/imagehosting
cd /opt/imagehosting
```

2. 下载并解压程序：
   从 [releases页面](https://github.com/nodeseeker/goImage/releases) 下载最新版本并解压到 `/opt/imagehosting` 目录。
```bash
# 下载服务器端程序
wget https://github.com/nodeseeker/goImage/releases/download/v0.1.3/imagehosting-server-linux-amd64.zip
# 解压文件
unzip imagehosting-server-linux-amd64.zip -d /opt/imagehosting
```
解压后的目录结构：
```
/opt/imagehosting/imagehosting # 服务器程序文件
/opt/imagehosting/config.json # 配置文件
/opt/imagehosting/static/favicon.ico # 网站图标
/opt/imagehosting/static/robots.txt # 爬虫协议
/opt/imagehosting/templates/home.html # 首页模板
/opt/imagehosting/templates/login.html # 登录模板
/opt/imagehosting/templates/upload.html # 上传模板
/opt/imagehosting/templates/admin.html # 管理模板
/opt/imagehosting/templates/deleted.jpg # 已删除图片的占位图片
```

3. 设置权限：
```bash
sudo chown -R root:root /opt/imagehosting
sudo chmod 755 /opt/imagehosting/imagehosting
```

## 配置说明

### 1. 程序配置文件

编辑 `/opt/imagehosting/config.json`，示例如下：

```json
{
    "telegram": {
        "token": "1234567890:ABCDEFG_ab1-asdfghjkl12345",
        "chatId": -123456789
    },
    "admin": {
        "username": "nodeseeker",
        "password": "nodeseeker@123456"
    },
    "site": {
        "name": "NodeSeek",
        "maxFileSize": 10,
        "port": 18080,
        "host": "127.0.0.1",
        "favicon": "favicon.ico"
    },
    "database": {
        "path": "./images.db",
        "maxOpenConns": 25,
        "maxIdleConns": 10,
        "connMaxLifetime": "5m"
    },
    "security": {
        "rateLimit": {
            "enabled": true,
            "limit": 60,
            "window": "1m"
        },
        "allowedHosts": ["localhost", "127.0.0.1"],
        "sessionSecret": "",
        "statusKey": "nodeseek_status",
        "requireLoginForUpload": false
    },
    "environment": "production"
}
```
详细的说明如下：

**基本配置**
- `telegram.token`：电报机器人的Bot Token
- `telegram.chatId`：频道的Chat ID
- `admin.username`：网站管理员用户名
- `admin.password`：网站管理员密码
- `site.name`：网站名称
- `site.favicon`：网站图标文件名
- `site.maxFileSize`：最大上传文件大小（单位：MB），建议10MB
- `site.port`：服务端口，默认18080
- `site.host`：服务监听地址，默认127.0.0.1本地监听；如果需要调试或外网访问，可修改为0.0.0.0

**数据库配置**
- `database.path`：SQLite数据库文件路径，默认为"./images.db"
- `database.maxOpenConns`：最大数据库连接数，默认25
- `database.maxIdleConns`：最大空闲连接数，默认10
- `database.connMaxLifetime`：连接最大生存时间，格式为时间字符串，如"5m"表示5分钟

**安全配置**
- `security.rateLimit.enabled`：是否启用请求速率限制，true或false
- `security.rateLimit.limit`：在指定时间窗口内允许的最大请求数，默认60
- `security.rateLimit.window`：速率限制的时间窗口，格式为时间字符串，如"1m"表示1分钟
- `security.allowedHosts`：允许访问的主机名列表
- `security.sessionSecret`：会话密钥，留空将自动生成
- `security.statusKey`：状态页面访问密钥
- `security.requireLoginForUpload`：是否要求登录后才能上传图片，true表示仅登录用户可上传，false表示所有用户都可上传（默认false）

**环境配置**
- `environment`：运行环境，"development"（开发环境）或"production"（生产环境）

### 2. Systemd 服务配置

创建服务文件：
```bash
sudo vim /etc/systemd/system/imagehosting.service
```

服务文件内容：
```ini
[Unit]
Description=Image Hosting Service
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=5
User=root
WorkingDirectory=/opt/imagehosting
ExecStart=/opt/imagehosting/imagehosting

[Install]
WantedBy=multi-user.target
```

## 2. Nginx 配置示例

在你的网站配置文件中添加：
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com; # 填写你的域名
    
    # SSL 配置部分
    ssl_certificate /path/to/cert.pem; # 填写你的 SSL 证书路径，以实际为准
    ssl_certificate_key /path/to/key.pem; # 填写你的 SSL 证书密钥路径，以实际为准
    
    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 50m; # 限制上传文件大小，必须大于程序配置的最大文件大小
    }
}
```

## 启动和维护

### 命令行参数

程序支持以下命令行参数：

| 参数 | 说明 |
|------|------|
| `-config` | 指定配置文件的绝对或相对路径（默认: ./config.json） |
| `-workdir` | 指定工作目录，程序会切换到该目录运行 |
| `-help` | 显示帮助信息 |
| `-version` | 显示版本信息 |

**使用示例：**
```bash
# 使用默认配置（当前目录下的 config.json）
./imagehosting

# 指定配置文件路径
./imagehosting -config /etc/goimage/config.json

# 指定工作目录（templates/、static/ 等相对路径都基于此目录）
./imagehosting -workdir /opt/imagehosting

# 同时指定配置文件和工作目录
./imagehosting -workdir /opt/imagehosting -config /etc/goimage/config.json
```

### Systemd 服务管理

1. 启动服务：
```bash
sudo systemctl daemon-reload # 重新加载配置，仅首次安装时执行
sudo systemctl enable imagehosting # 设置开机自启
sudo systemctl start imagehosting # 启动服务
sudo systemctl restart imagehosting # 重启服务
sudo systemctl status imagehosting # 查看服务状态
sudo systemctl stop imagehosting # 停止服务
```

2. 检查日志：
```bash
sudo journalctl -u imagehosting -f # 查看服务日志
```

## 安全建议

1. **文件类型验证**：
   - 服务器会验证上传的文件类型，只允许指定图片格式
   - 建议定期审查上传的文件

2. **限制上传大小**：
   - 合理设置 `site.maxFileSize` 参数
   - 保持 Nginx 的 `client_max_body_size` 稍大于程序配置值

3. **速率限制**：
   - 启用内置速率限制功能（`security.rateLimit.enabled` 设为 `true`）
   - 根据实际需求调整速率限制参数
   - 结合 Nginx 的速率限制功能实现多层防护

4. **API 访问控制**：
   - 如果启用API，则**强烈建议启用 API Key 认证**（设置 `security.requireAPIKey` 为 `true`）
   - 使用 API Key 生成工具生成强随机密钥（至少 32 字节）
   - 为不同的客户端或用户分配不同的 API Key，便于追踪和管理
   - 考虑在 Nginx 中限制 API 访问来源（IP 白名单）
   - 定期轮换 API Key，降低密钥泄露风险
   - 监控 API 使用情况，通过日志追踪未授权访问尝试

## 更新日志
- 2024-12-22：v0.0.1 初始版本发布
- 2025-02-20：v0.1.0 修复telegram的URL有效期失效bug，与此前的预发布版本数据库不兼容，需要全新安装
- 2025-04-11：v0.1.1 新增从剪贴板上传图片功能，支持多架构Linux系统
- 2025-05-21：v0.1.2 一大堆性能优化
- 2025-05-22：v0.1.3 新增RESTful API接口和独立的命令行客户端工具，服务端和客户端分离
- 2025-06-29：v0.1.4 修复WebP在telegram channel中被错误识别
- 2025-10-14：v0.1.5 新增缩略图功能
- 2025-12-07：v0.1.6 新增只允许已登录上传图片的功能
- 2025-12-07：v0.1.7 新增跨域资源共享（CORS）支持，修复响应头设置顺序问题

## 常见问题

1. 上传失败：
   - 检查 Bot Token 是否正确
   - 确认 Bot 是否具有频道管理员权限
   - 验证 SSL 证书是否正确配置

2. 无法访问管理界面：
   - 确认配置文件中的管理员账号密码正确
   - 检查服务是否正常运行
   - 查看服务日志排查问题

3. 上传文件大小限制：
   - 修改 Nginx 配置中的 `client_max_body_size` 参数
   - 修改程序配置文件中的 `site.maxFileSize` 参数

4. API 相关问题：
   - **认证失败**：确保 API Key 正确配置在 `config.json` 的 `security.apiKeys` 数组中
   - **未授权错误**：检查客户端是否使用 `-key` 参数传递了正确的 API Key
   - **API Key 不工作**：修改配置后需要重启服务（`sudo systemctl restart imagehosting`）
   - 检查 API 端点是否正确配置
   - 在使用客户端时，确保 API URL 完整且正确
   - 查看服务日志以获取详细的认证失败信息

5. 已知bug：
   - 登录时，输入错误的用户名或密码将提示`Invalid credentials`，需要在新标签页再次打开登录页面.直接在原先标签页刷新，将一直报错`Invalid credentials`。

6. 动态图片限制（Telegram 存储限制）：
   - **动态 WebP**：上传后会被 Telegram 转换为静态图片，动画效果丢失
   - **GIF**：上传后会被 Telegram 转换为 MP4 视频格式
   - 这是 Telegram 服务端的固有行为，无法通过程序规避
   - 如需完整支持动态图片，建议考虑其他存储方案（如 S3、Cloudflare R2 等）
   - 静态图片（JPG/PNG/静态 WebP）不受影响，可正常显示

7. 图片嵌入其他网站：
   - 程序已支持 CORS，可在其他网站通过 `<img>` 标签嵌入图片
   - 对于被转换为 MP4 的 GIF，需使用 `<video>` 标签播放：
     ```html
     <video autoplay loop muted playsinline>
       <source src="https://your-domain.com/file/xxx.gif" type="video/mp4">
     </video>
     ```
  
---

## 客户端和API
程序提供符合RESTful规范的API接口，方便第三方集成和自动化上传。具体内容参考 [API.md](API.md) 文件。