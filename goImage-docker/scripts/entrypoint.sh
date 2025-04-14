#!/bin/bash

# 设置错误时退出
set -e

# 进入应用程序目录
cd /opt/imagehosting

# 运行数据库迁移（如果有）
# 这里可以添加数据库迁移的命令，例如：
# migrate -path db/migrations -database "$DATABASE_URL" up

# 启动应用程序
exec ./imagehosting