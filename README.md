# goImage 图床

基于 Go 语言开发的图片托管服务，使用 Telegram 作为存储后端。

## 安装要求

- 使用Systemd的Linux系统
- Telegram 频道和 Bot



## 配置文件说明

配置文件位于 `/opt/imagehosting/config.json`

```json
{
    "telegram": {
        "token": "your-bot-token",
        "chatId": -your-chat-id
    },
    "admin": {
        "username": "admin-username",
        "password": "admin-password"
    },
    "site": {
        "name": "站点名称",
        "maxFileSize": 10,
        "port": 8080,
        "host": "0.0.0.0"
    }
}

详细说明如下
```
telegram:
    - token: 替换为您的Telegram Bot Token
    - chatId: 替换为您的Telegram Chat ID
admin:
    - username: 设置管理员用户名
    - password: 设置管理员密码
site: 
    - name: 设置您的网站名称
    - maxFileSize: 最大文件上传大小（MB）
    - port: 服务运行端口（默认18080）
    - host: 服务监听地址（127.0.0.1监听本地，0.0.0.0监听所有IP）
```

## 编译部署步骤

1. 准备运行环境
```bash
# 创建服务目录
sudo mkdir -p /opt/imagehosting
sudo cp imagehosting /opt/imagehosting/
sudo cp config.json /opt/imagehosting/
sudo cp -r templates /opt/imagehosting/

# 设置权限
sudo chown -R root:root /opt/imagehosting/imagehosting
sudo chmod -R 755 /opt/imagehosting/imagehosting
```

2. 创建 systemd 服务文件
   使用vim新建服务文件，写入配置文件。
```bash
sudo vim /etc/systemd/system/imagehosting.service
```

将以下内容写入服务文件：
```ini
[Unit]
Description=Image Hosting Service
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/opt/imagehosting
ExecStart=/opt/imagehosting/imagehosting
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

3. Nginx 反代服务
    以下是用于Nginx的反向代理配置示例。请将其添加到你的Nginx配置文件中。注意：网站必须启用SSL/TLS，否则Telegram Bot API将无法正常工作。
```nginx
location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 100m;
    }
```
完成之后，重新读取Nginx配置文件，使用以下命令：
```bash
nginx -t # 检查配置文件，如果ok则继续
sudo systemctl reload nginx
```

4. 服务配置与启动
```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start imagehosting

# 设置开机自启
sudo systemctl enable imagehosting

# 重启服务
sudo systemctl restart imagehosting

# 停止服务
sudo systemctl stop imagehosting

# 检查服务状态
sudo systemctl status imagehosting

# 查看日志
sudo journalctl -u imagehosting -f
```



```

## 安全建议

1. 确保配置文件权限正确：
```bash
sudo chmod 600 /opt/imagehosting/config.json
```

