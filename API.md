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

## API Key 生成与管理

程序提供符合RESTful规范的API接口，方便第三方集成和自动化上传。如果启用 API 认证功能，需要先生成 API Key。

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

1. **服务器组件 (imagehosting)**:
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


## Status 检查接口
GoImage 服务器提供了一个简单的状态检查接口，方便监控服务的运行状态。
例如`https://img.example.com/status?key=goimage_status_key`，其中`goimage_status_key`是在配置文件中设置的状态检查密钥。

具体信息如下：
- **端点**: `/status`
- **参数**:
  - `key` (必需): 状态检查密钥，必须与配置文件中的 `status.checkKey` 相匹配
- **方法**: `GET`
- **响应格式**: JSON
- **示例响应**:
```json
{
  "status": "ok", # 服务状态，"ok"表示正常
  "startTime": "2025-12-07T14:47:33.151149269+08:00", # 服务启动时间
  "uptime": "725h46m49.280276368s", # 运行时间
  "goVersion": "go1.25.5", # Go 语言版本
  "numGoroutine": 8, # 当前 Goroutine 数量
  "numCPU": 1, # CPU 核心数
  "memStats": {
    "alloc": 1607912, # 当前分配的内存字节数
    "totalAlloc": 4917031680, # 自启动以来分配的总内存字节数
    "sys": 22370568, # 从系统获取的内存字节数
    "numGC": 20707, # 垃圾回收次数
    "pauseTotalNs": 10890448805 # 垃圾回收暂停的总时间（纳秒）
  },
  "urlCacheSize": 208 # URL 缓存大小
}
```