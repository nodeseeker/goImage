#!/bin/bash

# 设置程序名称和版本
APP_NAME="imagehosting"
VERSION="0.1.3"
SERVER_PATH="./cmd/server"
CLIENT_PATH="./cmd/client"
OUTPUT_DIR="./bin"

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

# 客户端平台列表（增加Windows和macOS支持）
CLIENT_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/loong64"
    "linux/riscv64"
    "windows/amd64"
    "windows/386"
    "windows/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# 开始编译
echo "🚀 开始编译 $APP_NAME v$VERSION..."

# 编译服务器端
echo "🖥️ 编译服务器端..."
for PLATFORM in "${SERVER_PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    
    SERVER_OUTPUT="$OUTPUT_DIR/${APP_NAME}-server-$GOOS-$GOARCH"
    SERVER_ZIP_NAME="${APP_NAME}-server-$GOOS-$GOARCH.zip"
    
    if [ $GOOS = "windows" ]; then
        SERVER_OUTPUT="${SERVER_OUTPUT}.exe"
    fi
    
    echo "📦 编译服务器 $GOOS/$GOARCH..."
    
    # 编译服务器
    env GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags "$LDFLAGS" -o $SERVER_OUTPUT $SERVER_PATH
    SERVER_SUCCESS=$?
    
    if [ $SERVER_SUCCESS -ne 0 ]; then
        echo "❌ 服务器编译失败: $GOOS/$GOARCH"
    else
        echo "✅ 服务器编译成功: $GOOS/$GOARCH"
        
        # 创建ZIP文件
        echo "📦 打包服务器为 $SERVER_ZIP_NAME..."
        (cd $OUTPUT_DIR && zip -j "$SERVER_ZIP_NAME" "$(basename $SERVER_OUTPUT)" && echo "✅ 服务器打包完成: $OUTPUT_DIR/$SERVER_ZIP_NAME") || echo "❌ 服务器打包失败: $SERVER_ZIP_NAME"
    fi
done

# 编译客户端
echo "📱 编译客户端..."
for PLATFORM in "${CLIENT_PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    
    CLIENT_OUTPUT="$OUTPUT_DIR/${APP_NAME}-client-$GOOS-$GOARCH"
    CLIENT_ZIP_NAME="${APP_NAME}-client-$GOOS-$GOARCH.zip"
    
    if [ $GOOS = "windows" ]; then
        CLIENT_OUTPUT="${CLIENT_OUTPUT}.exe"
    fi
    
    echo "📦 编译客户端 $GOOS/$GOARCH..."
    
    # 编译客户端
    env GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags "$LDFLAGS" -o $CLIENT_OUTPUT $CLIENT_PATH
    CLIENT_SUCCESS=$?
    
    if [ $CLIENT_SUCCESS -ne 0 ]; then
        echo "❌ 客户端编译失败: $GOOS/$GOARCH"
    else
        echo "✅ 客户端编译成功: $GOOS/$GOARCH"
        
        # 创建ZIP文件
        echo "📦 打包客户端为 $CLIENT_ZIP_NAME..."
        (cd $OUTPUT_DIR && zip -j "$CLIENT_ZIP_NAME" "$(basename $CLIENT_OUTPUT)" && echo "✅ 客户端打包完成: $OUTPUT_DIR/$CLIENT_ZIP_NAME") || echo "❌ 客户端打包失败: $CLIENT_ZIP_NAME"
    fi
done

# 计算SHA256校验和
echo "🔐 生成校验和文件..."
(cd $OUTPUT_DIR && sha256sum *.zip > SHA256SUMS.txt)
echo "✅ 校验和文件已生成: $OUTPUT_DIR/SHA256SUMS.txt"

# 清理二进制文件，只保留zip包
echo "🧹 清理编译文件，只保留zip包..."
find $OUTPUT_DIR -type f -not -name "*.zip" -not -name "SHA256SUMS.txt" -delete

echo "🎉 编译完成!"
ls -la $OUTPUT_DIR
