#!/bin/bash

# 设置程序名称和版本
APP_NAME="imagehosting"
VERSION="0.1.6"
SERVER_PATH="./cmd/server"
CLIENT_PATH="./cmd/client"
OUTPUT_DIR="./bin"
# 禁用 CGO，使构建更易于交叉编译并生成纯静态二进制
export CGO_ENABLED=0

# 创建输出目录
mkdir -p $OUTPUT_DIR

# 清理旧的构建文件
echo "🧹 清理旧的构建文件..."
rm -rf $OUTPUT_DIR/*

# 检查zip命令是否存在
if ! command -v zip &> /dev/null; then
    echo "❌ 错误: 未找到zip命令。请安装zip后再运行此脚本。"
    echo "   可通过 'sudo apt-get install zip' 或 'sudo yum install zip' 安装"
    exit 1
fi

# 设置编译时间
BUILD_TIME=$(date "+%Y-%m-%d %H:%M:%S")
BUILD_TIME_FLAGS="-X 'main.buildTime=$BUILD_TIME'"

# 移除个人信息的包和信息
echo "🔒 设置移除个人信息..."
LDFLAGS="-s -w $BUILD_TIME_FLAGS"

# 支持的目标平台/架构
# 服务器平台列表
SERVER_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/loong64"
    "linux/riscv64"
)

# 客户端编译已禁用 — 本脚本仅编译服务器端

# 开始编译
echo "🚀 开始编译 $APP_NAME v$VERSION..."

# 编译服务器端
echo "🖥️ 编译服务器端..."
for PLATFORM in "${SERVER_PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    
    SERVER_BINARY="${APP_NAME}-server-$GOOS-$GOARCH"
    SERVER_ZIP_NAME="${APP_NAME}-server-$GOOS-$GOARCH.zip"
    
    if [ $GOOS = "windows" ]; then
        SERVER_BINARY="${SERVER_BINARY}.exe"
    fi
    
    echo "📦 编译服务器 $GOOS/$GOARCH..."
    
    # 编译服务器
    env GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/$SERVER_BINARY" $SERVER_PATH
    SERVER_SUCCESS=$?
    
    if [ $SERVER_SUCCESS -ne 0 ]; then
        echo "❌ 服务器编译失败: $GOOS/$GOARCH"
    else
        echo "✅ 服务器编译成功: $GOOS/$GOARCH"
        
        # 创建临时打包目录，模拟 /opt/imagehosting 结构
        PACK_DIR="$OUTPUT_DIR/pack_temp"
        rm -rf "$PACK_DIR"
        mkdir -p "$PACK_DIR/imagehosting/static"
        mkdir -p "$PACK_DIR/imagehosting/templates"
        
        # 复制服务器程序并重命名为 imagehosting
        cp "$OUTPUT_DIR/$SERVER_BINARY" "$PACK_DIR/imagehosting/imagehosting"
        
        # 复制配置文件
        cp ./config.json "$PACK_DIR/imagehosting/"
        
        # 复制静态文件
        cp ./static/favicon.ico "$PACK_DIR/imagehosting/static/"
        cp ./static/robots.txt "$PACK_DIR/imagehosting/static/"
        cp ./static/deleted.jpg "$PACK_DIR/imagehosting/templates/"
        
        # 复制模板文件（保持 .tmpl 后缀）
        cp ./templates/home.tmpl "$PACK_DIR/imagehosting/templates/"
        cp ./templates/login.tmpl "$PACK_DIR/imagehosting/templates/"
        cp ./templates/upload.tmpl "$PACK_DIR/imagehosting/templates/"
        cp ./templates/admin.tmpl "$PACK_DIR/imagehosting/templates/"
        
        # 创建ZIP文件
        echo "📦 打包服务器为 $SERVER_ZIP_NAME..."
        (cd "$PACK_DIR" && zip -r "../$SERVER_ZIP_NAME" imagehosting && echo "✅ 服务器打包完成: $OUTPUT_DIR/$SERVER_ZIP_NAME") || echo "❌ 服务器打包失败: $SERVER_ZIP_NAME"
        
        # 清理临时目录和二进制文件
        rm -rf "$PACK_DIR"
        rm -f "$OUTPUT_DIR/$SERVER_BINARY"
    fi
done

# 已跳过客户端编译（如果需要，可在未来启用）
echo "ℹ️ 跳过客户端编译。"

# 计算SHA256校验和
echo "🔐 生成校验和文件..."
(cd $OUTPUT_DIR && sha256sum *.zip > SHA256SUMS.txt)
echo "✅ 校验和文件已生成: $OUTPUT_DIR/SHA256SUMS.txt"

echo "🎉 编译完成!"
ls -la $OUTPUT_DIR
