# goImage 图床

基于 Go 语言开发的图片托管服务，使用 Telegram 作为存储后端。

## 功能特性
- 无限容量，上传图片到 Telegram 频道
- 轻量级要求，内存占用小于 10MB
- 支持管理员登录，查看上传记录和删除图片
- 提供 RESTful API 接口，支持第三方集成
- 包含独立的命令行客户端工具


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
/opt/imagehosting/imagehosting-server # 服务器程序文件
/opt/imagehosting/config.json # 配置文件
/opt/imagehosting/static/favicon.ico # 网站图标
/opt/imagehosting/static/robots.txt # 爬虫协议
/opt/imagehosting/templates/home.html # 首页模板
/opt/imagehosting/templates/login.html # 登录模板
/opt/imagehosting/templates/upload.html # 上传模板
/opt/imagehosting/templates/admin.html # 管理模板
```

3. 设置权限：
```bash
sudo chown -R root:root /opt/imagehosting
sudo chmod 755 /opt/imagehosting/imagehosting-server
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
    "environment": "development"
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
ExecStart=/opt/imagehosting/imagehosting-server

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

## API Key 生成与管理

如果启用 API 认证功能，需要先生成 API Key。

### 生成 API Key

项目提供了专门的 API Key 生成工具：

1. **下载生成工具**：
   从 [releases页面](https://github.com/nodeseeker/goImage/releases) 下载 `generate_apikey` 工具，或使用源码编译：

```bash
cd /opt/imagehosting/tools
go build -o generate_apikey generate_apikey.go
```

2. **生成密钥**：

```bash
# 生成一个密钥（默认32字节）
./generate_apikey

# 生成5个密钥
./generate_apikey -count 5

# 生成64字节的密钥
./generate_apikey -length 64
```

生成示例：
```
生成 1 个 API 密钥 (长度: 32 字节):

1. Xy9kP2mN5rQ8sT1vW4zB7cD0fG3hJ6kL

使用说明:
1. 将生成的密钥添加到 config.json 的 security.apiKeys 数组中
2. 设置 security.requireAPIKey 为 true 以启用API认证
3. 客户端使用 -key 参数传递API密钥
```

### 配置 API Key

编辑 `/opt/imagehosting/config.json`，在 `security` 部分添加生成的密钥：

```json
{
  "security": {
    "requireAPIKey": true,
    "apiKeys": [
      "Xy9kP2mN5rQ8sT1vW4zB7cD0fG3hJ6kL",
      "Another-API-Key-If-Needed"
    ],
    "rateLimit": {
      "enabled": true,
      "limit": 60,
      "window": "1m"
    }
  }
}
```

### API Key 管理最佳实践

1. **定期轮换**：建议定期更换 API Key
2. **多密钥策略**：可以为不同的用户或应用分配不同的密钥
3. **安全存储**：不要将 API Key 提交到版本控制系统
4. **及时撤销**：当密钥泄露时，立即从配置中删除并重启服务
5. **密钥长度**：推荐使用至少 32 字节的密钥长度

### 启用/禁用 API 认证

- **启用认证**：设置 `security.requireAPIKey` 为 `true`
- **禁用认证**：设置 `security.requireAPIKey` 为 `false`（默认）

修改配置后需要重启服务：
```bash
sudo systemctl restart imagehosting
```

## 启动和维护

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

1. **API 访问控制**：
   - **强烈建议启用 API Key 认证**（设置 `security.requireAPIKey` 为 `true`）
   - 使用 API Key 生成工具生成强随机密钥（至少 32 字节）
   - 为不同的客户端或用户分配不同的 API Key，便于追踪和管理
   - 考虑在 Nginx 中限制 API 访问来源（IP 白名单）
   - 定期轮换 API Key，降低密钥泄露风险
   - 监控 API 使用情况，通过日志追踪未授权访问尝试

2. **文件类型验证**：
   - 服务器会验证上传的文件类型，只允许图片格式
   - 建议定期审查上传的文件

3. **限制上传大小**：
   - 合理设置 `site.maxFileSize` 参数
   - 保持 Nginx 的 `client_max_body_size` 稍大于程序配置值

4. **速率限制**：
   - 启用内置速率限制功能（`security.rateLimit.enabled` 设为 `true`）
   - 根据实际需求调整速率限制参数
   - 结合 Nginx 的速率限制功能实现多层防护

## 更新日志
- 2024-12-22：v0.0.1 初始版本发布
- 2025-02-20：v0.1.0 修复telegram的URL有效期失效bug，与此前的预发布版本数据库不兼容，需要全新安装
- 2025-04-11：v0.1.1 新增从剪贴板上传图片功能，支持多架构Linux系统
- 2025-05-21：v0.1.2 一大堆性能优化
- 2025-05-22：v0.1.3 新增RESTful API接口和独立的命令行客户端工具，服务端和客户端分离
- 2025-06-29：v0.1.4 修复WebP在telegram channel中被错误识别
- 2025-10-14：v0.1.5 新增缩略图功能

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
  
---

# GoImage 命令行客户端

GoImage 现提供独立的命令行客户端，可以方便地通过 RESTful API 上传图片到 GoImage 图床服务。客户端工具与服务器是分离的，可以单独下载和安装。

## 功能特点

- 遵循 RESTful API 设计规范
- 简洁的命令行界面
- 支持多种图片格式
- 详细的错误报告
- 可配置的超时设置
- 详细模式下提供更多信息
- 支持多平台：Linux、Windows 和 macOS



## 安装客户端

从 [releases页面](https://github.com/nodeseeker/goImage/releases) 下载适合您系统架构的客户端版本，例如：

**Linux系统：**
```bash
# 下载Linux amd64架构的客户端
wget https://github.com/nodeseeker/goImage/releases/download/v0.1.3/imagehosting-client-linux-amd64.zip

# 解压缩
unzip imagehosting-client-linux-amd64.zip

# 设置执行权限
chmod +x imagehosting-client-linux-amd64
```

**Windows系统：**
```bash
# 下载Windows amd64架构的客户端
# 浏览器下载: https://github.com/nodeseeker/goImage/releases/download/v0.1.3/imagehosting-client-windows-amd64.zip

# 解压缩后直接运行 imagehosting-client-windows-amd64.exe
```

**macOS系统：**
```bash
# 下载macOS amd64架构的客户端
curl -L -o imagehosting-client-darwin-amd64.zip https://github.com/nodeseeker/goImage/releases/download/v0.1.3/imagehosting-client-darwin-amd64.zip

# 解压缩
unzip imagehosting-client-darwin-amd64.zip

# 设置执行权限
chmod +x imagehosting-client-darwin-amd64

# 如果出现"无法打开，因为开发者身份无法验证"的提示
# 请在"系统偏好设置" > "安全性与隐私"中允许运行
```
```

## 使用方法

最基本用法：

**Linux/macOS**:
```bash
./imagehosting-client-linux-amd64 -url https://img.example.com/api/v1/upload -file /path/to/image.jpg
```

**Windows**:
```bash
imagehosting-client-windows-amd64.exe -url https://img.example.com/api/v1/upload -file C:\path\to\image.jpg
```

使用 API Key 认证（当服务器启用认证时）：

**Linux/macOS**:
```bash
./imagehosting-client-linux-amd64 -url https://img.example.com/api/v1/upload -file /path/to/image.jpg -key your-api-key
```

**Windows**:
```bash
imagehosting-client-windows-amd64.exe -url https://img.example.com/api/v1/upload -file C:\path\to\image.jpg -key your-api-key
```

启用详细输出：

**Linux/macOS**:
```bash
./imagehosting-client-linux-amd64 -url https://img.example.com/api/v1/upload -file /path/to/image.jpg -verbose
```

**Windows**:
```bash
imagehosting-client-windows-amd64.exe -url https://img.example.com/api/v1/upload -file C:\path\to\image.jpg -verbose
```

指定上传超时时间：

**Linux/macOS**:
```bash
./imagehosting-client-linux-amd64 -url https://img.example.com/api/v1/upload -file /path/to/image.jpg -timeout 120
```

**Windows**:
```bash
imagehosting-client-windows-amd64.exe -url https://img.example.com/api/v1/upload -file C:\path\to\image.jpg -timeout 120
```

显示帮助信息：

**Linux/macOS**:
```bash
./imagehosting-client-linux-amd64 -help
```

**Windows**:
```bash
imagehosting-client-windows-amd64.exe -help
```

## 参数说明

| 参数 | 描述 | 必填 | 默认值 |
|------|------|------|------|
| `-url` | 图床服务器API地址 | 是 | - |
| `-file` | 要上传的图片文件路径 | 是 | - |
| `-key` | API认证密钥（服务器启用认证时必需） | 条件必填 | - |
| `-timeout` | 上传超时时间(秒) | 否 | 60 |
| `-verbose` | 显示详细输出 | 否 | false |
| `-help` | 显示帮助信息 | 否 | false |
| `-version` | 显示版本信息 | 否 | false |

## 支持的文件类型

- JPG/JPEG
- PNG
- GIF
- WebP

## RESTful API 说明

goImage 服务器提供了标准的 RESTful API 接口，可以被任何支持 HTTP 请求的客户端或应用程序调用：

### API 基本信息

- **服务器端点**: `/api/v1/upload`
- **方法**: `POST`
- **Content-Type**: `multipart/form-data`
- **参数**: `image` - 图片文件
- **响应格式**: JSON
- **跨域支持**: 默认启用，允许来自任何源的请求

### API 认证

goImage 支持基于 API Key 的认证机制，用于保护 API 端点免受未授权访问。

**认证配置**（在 `config.json` 中）：
```json
{
  "security": {
    "requireAPIKey": false,
    "apiKeys": [
      "your-api-key-1",
      "your-api-key-2"
    ]
  }
}
```

- **`requireAPIKey`**: 设置为 `true` 启用 API 认证，`false` 则不需要认证（默认）
- **`apiKeys`**: 允许的 API Key 列表，支持配置多个密钥

**认证方式**：

API Key 可以通过两种方式传递：

1. **使用 `X-API-Key` 请求头**（推荐）：
```bash
curl -X POST https://your-domain.com/api/v1/upload \
  -H "X-API-Key: your-api-key" \
  -F "image=@/path/to/image.jpg"
```

2. **使用 `Authorization: Bearer` 请求头**：
```bash
curl -X POST https://your-domain.com/api/v1/upload \
  -H "Authorization: Bearer your-api-key" \
  -F "image=@/path/to/image.jpg"
```

成功响应示例：
```json
{
  "success": true,
  "message": "上传成功",
  "data": {
    "url": "https://example.com/file/abc123.jpg",
    "filename": "example.jpg",
    "contentType": "image/jpeg",
    "size": 123456,
    "uploadTime": "2025-05-22T12:00:00Z"
  }
}
```

失败响应示例：
```json
{
  "success": false,
  "message": "文件大小超过限制",
  "data": null
}
```

所有可能的错误消息包括：
- "未指定文件"
- "文件大小超过限制"
- "不支持的文件类型"
- "上传处理超时"
- "服务器内部错误"
- "存储处理失败"
- "未授权：需要有效的API密钥" (当启用API认证时)

## 错误处理

客户端会处理常见的错误情况，包括：

1. 文件不存在
2. 不支持的文件类型
3. 服务器连接失败
4. 上传超时
5. 服务器返回的错误

在使用 `-verbose` 模式时，会显示更详细的错误信息。

## 注意事项

- 确保服务器API地址正确
- 如果URL不包含 `http://` 或 `https://` 前缀，客户端会自动添加 `http://`
- 大文件上传可能需要增加超时时间
- 在详细模式下可以获取更多上传相关信息

## 服务器与客户端架构

从 v0.1.3 版本开始，GoImage 采用了服务器/客户端分离的架构：

1. **服务器组件 (imagehosting-server)**:
   - 提供完整的 Web 界面
   - 负责图片存储和管理
   - 提供 RESTful API 接口
   - 处理用户认证和权限控制

2. **客户端组件 (imagehosting-client)**:
   - 轻量级命令行工具
   - 通过 RESTful API 与服务器通信
   - 支持脚本集成和批处理
   - 多平台支持

## 开发者信息

### API 集成

如果您想将 GoImage 集成到您自己的应用中，可以通过 RESTful API 接口进行：

```python
# Python 示例（不使用认证）
import requests

def upload_image(api_url, image_path):
    with open(image_path, 'rb') as f:
        files = {'image': f}
        response = requests.post(api_url, files=files)
    return response.json()

# 使用方法
result = upload_image('https://your-domain.com/api/v1/upload', 'path/to/image.jpg')
print(result['data']['url'])  # 打印上传后的URL
```

```python
# Python 示例（使用 API Key 认证）
import requests

def upload_image_with_key(api_url, image_path, api_key):
    with open(image_path, 'rb') as f:
        files = {'image': f}
        headers = {'X-API-Key': api_key}  # 或使用 'Authorization': f'Bearer {api_key}'
        response = requests.post(api_url, files=files, headers=headers)
    return response.json()

# 使用方法
api_key = 'your-api-key'
result = upload_image_with_key('https://your-domain.com/api/v1/upload', 'path/to/image.jpg', api_key)
print(result['data']['url'])
```

```javascript
// JavaScript 示例（不使用认证）
async function uploadImage(apiUrl, imageFile) {
  const formData = new FormData();
  formData.append('image', imageFile);
  
  const response = await fetch(apiUrl, {
    method: 'POST',
    body: formData
  });
  
  return await response.json();
}

// 使用方法
uploadImage('https://your-domain.com/api/v1/upload', fileInput.files[0])
  .then(result => console.log(result.data.url));
```

```javascript
// JavaScript 示例（使用 API Key 认证）
async function uploadImageWithKey(apiUrl, imageFile, apiKey) {
  const formData = new FormData();
  formData.append('image', imageFile);
  
  const response = await fetch(apiUrl, {
    method: 'POST',
    headers: {
      'X-API-Key': apiKey  // 或使用 'Authorization': `Bearer ${apiKey}`
    },
    body: formData
  });
  
  return await response.json();
}

// 使用方法
const apiKey = 'your-api-key';
uploadImageWithKey('https://your-domain.com/api/v1/upload', fileInput.files[0], apiKey)
  .then(result => console.log(result.data.url));
```

```bash
# cURL 示例（不使用认证）
curl -X POST https://your-domain.com/api/v1/upload \
  -F "image=@/path/to/image.jpg"
```

```bash
# cURL 示例（使用 API Key 认证 - X-API-Key 方式）
curl -X POST https://your-domain.com/api/v1/upload \
  -H "X-API-Key: your-api-key" \
  -F "image=@/path/to/image.jpg"
```

```bash
# cURL 示例（使用 API Key 认证 - Bearer Token 方式）
curl -X POST https://your-domain.com/api/v1/upload \
  -H "Authorization: Bearer your-api-key" \
  -F "image=@/path/to/image.jpg"
```
