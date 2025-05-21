#!/bin/bash

# 设置程序名称和版本
APP_NAME="imagehosting"
VERSION="0.1.2"
MAIN_PATH="./cmd/server"
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
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/loong64"
    "linux/riscv64"
)

# 开始编译
echo "🚀 开始编译 $APP_NAME v$VERSION..."

# 循环编译每个平台
for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    
    OUTPUT="$OUTPUT_DIR/$APP_NAME-$GOOS-$GOARCH"
    ZIP_NAME="$APP_NAME-$GOOS-$GOARCH.zip"
    
    if [ $GOOS = "windows" ]; then
        OUTPUT="$OUTPUT.exe"
    fi
    
    echo "📦 编译 $GOOS/$GOARCH..."
    
    # 设置平台特定的环境变量并执行构建
    env GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags "$LDFLAGS" -o $OUTPUT $MAIN_PATH
    
    if [ $? -ne 0 ]; then
        echo "❌ 编译 $GOOS/$GOARCH 失败"
    else
        echo "✅ 成功编译: $OUTPUT"
        
        # 创建ZIP文件
        echo "📦 打包为 $ZIP_NAME..."
        (cd $OUTPUT_DIR && zip -j "$ZIP_NAME" "$(basename $OUTPUT)" && echo "✅ 打包完成: $OUTPUT_DIR/$ZIP_NAME") || echo "❌ 打包失败: $ZIP_NAME"
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
