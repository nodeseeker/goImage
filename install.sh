#!/bin/bash

# 设置错误时退出
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}请使用root权限运行此脚本${NC}"
    exit 1
fi

# 检查系统要求
echo -e "${YELLOW}正在检查系统要求...${NC}"
if ! command -v systemctl &> /dev/null; then
    echo -e "${RED}错误: 系统需要支持systemd${NC}"
    exit 1
fi

# 创建安装目录
echo -e "${YELLOW}创建安装目录...${NC}"
INSTALL_DIR="/opt/imagehosting"
mkdir -p $INSTALL_DIR

# 复制程序文件
echo -e "${YELLOW}复制程序文件...${NC}"
cp imagehosting $INSTALL_DIR/
cp config.json $INSTALL_DIR/
cp -r templates $INSTALL_DIR/

# 创建服务用户
echo -e "${YELLOW}创建服务用户...${NC}"
id -u imagehosting &>/dev/null || useradd -r -s /bin/false imagehosting

# 设置权限
echo -e "${YELLOW}设置文件权限...${NC}"
chown -R imagehosting:imagehosting $INSTALL_DIR
chmod -R 755 $INSTALL_DIR
chmod 600 $INSTALL_DIR/config.json

# 创建日志文件
echo -e "${YELLOW}创建日志文件...${NC}"
touch /var/log/imagehosting.log /var/log/imagehosting.error.log
chown imagehosting:imagehosting /var/log/imagehosting.log /var/log/imagehosting.error.log

# 创建systemd服务文件
echo -e "${YELLOW}创建服务文件...${NC}"
cat > /etc/systemd/system/imagehosting.service << EOF
[Unit]
Description=Image Hosting Service
After=network.target

[Service]
Type=simple
User=imagehosting
Group=imagehosting
WorkingDirectory=/opt/imagehosting
ExecStart=/opt/imagehosting/imagehosting
Restart=always
RestartSec=10
StandardOutput=append:/var/log/imagehosting.log
StandardError=append:/var/log/imagehosting.error.log

# 安全加固
PrivateTmp=true
ProtectSystem=full
NoNewPrivileges=true
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

# 重载systemd配置
echo -e "${YELLOW}重载systemd配置...${NC}"
systemctl daemon-reload

# 启动服务
echo -e "${YELLOW}启动服务...${NC}"
systemctl start imagehosting
systemctl enable imagehosting

# 配置防火墙（如果存在）
echo -e "${YELLOW}配置防火墙...${NC}"
if command -v ufw &> /dev/null; then
    ufw allow 8080/tcp
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --reload
fi

# 检查服务状态
echo -e "${YELLOW}检查服务状态...${NC}"
if systemctl is-active --quiet imagehosting; then
    echo -e "${GREEN}服务安装成功并正在运行!${NC}"
    echo -e "服务状态: $(systemctl status imagehosting | grep Active)"
    echo -e "端口监听: $(netstat -tln | grep 8080)"
    echo -e "\n配置文件位置: ${INSTALL_DIR}/config.json"
    echo "请确保修改配置文件中的 Telegram Bot Token 和 Chat ID"
    echo "日志文件:"
    echo "  - /var/log/imagehosting.log"
    echo "  - /var/log/imagehosting.error.log"
else
    echo -e "${RED}服务安装失败，请检查日志文件${NC}"
    exit 1
fi

# 提示配置建议
echo -e "\n${YELLOW}建议操作:${NC}"
echo "1. 编辑配置文件: nano ${INSTALL_DIR}/config.json"
echo "2. 检查日志: tail -f /var/log/imagehosting.log"
echo "3. 重启服务: systemctl restart imagehosting"

# 设置文件句柄限制
echo -e "${YELLOW}设置系统限制...${NC}"
if ! grep -q "imagehosting soft nofile" /etc/security/limits.conf; then
    echo "imagehosting soft nofile 65535" >> /etc/security/limits.conf
    echo "imagehosting hard nofile 65535" >> /etc/security/limits.conf
fi

echo -e "\n${GREEN}安装完成!${NC}"
