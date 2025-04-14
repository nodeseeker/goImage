# goImage 图床 Docker 版本

基于 Go 语言开发的图片托管服务，使用 Telegram 作为存储后端，现已封装为 Docker 容器。

## 项目结构
```
goImage-docker
├── Dockerfile                # Docker 镜像构建文件
├── docker-compose.yml        # Docker Compose 配置文件
├── config                    # 配置文件目录
│   └── config.json          # 应用配置文件
├── data                      # 数据目录
│   └── .gitkeep             # 确保数据目录被 Git 跟踪
├── scripts                   # 脚本目录
│   └── entrypoint.sh        # Docker 容器入口脚本
└── README.md                 # 项目文档
```

## 功能特性
- 无限容量，上传图片到 Telegram 频道
- 轻量级要求，内存占用小于 10MB
- 支持管理员登录，查看上传记录和删除图片

## 前置准备
1. 创建 Telegram Bot（通过 @BotFather）
2. 记录获取的 Bot Token
3. 创建一个频道用于存储图片
4. 将 Bot 添加为频道管理员
5. 获取频道的 Chat ID（可通过 @getidsbot 获取）

## 使用说明

### 构建 Docker 镜像
在项目根目录下运行以下命令构建 Docker 镜像：
```
docker build -t goimage .
```

### 启动服务
使用 Docker Compose 启动服务：
```
docker-compose up -d
```

### 访问应用
应用将运行在 `http://localhost:8080`，您可以通过浏览器访问。

### 配置文件
编辑 `config/config.json` 文件，填写您的 Telegram Bot Token 和 Chat ID：
```json
{
    "telegram": {
        "token": "YOUR_TELEGRAM_BOT_TOKEN",
        "chatId": YOUR_CHAT_ID
    },
    "admin": {
        "username": "YOUR_ADMIN_USERNAME",
        "password": "YOUR_ADMIN_PASSWORD"
    },
    "site": {
        "name": "Your Site Name",
        "maxFileSize": 10,
        "port": 8080,
        "host": "0.0.0.0"
    }
}
```

## 日志
日志文件将保存在 Docker 容器中，您可以通过以下命令查看：
```
docker logs <container_id>
```

## 常见问题
1. **上传失败**：
   - 检查 Bot Token 是否正确
   - 确认 Bot 是否具有频道管理员权限

2. **无法访问管理界面**：
   - 确认配置文件中的管理员账号密码正确

3. **上传文件大小限制**：
   - 修改配置文件中的 `maxFileSize` 参数

## 贡献
欢迎提交 Issue 或 Pull Request，帮助我们改进项目。