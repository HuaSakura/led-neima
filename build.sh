#!/bin/bash
set -e

# 输出目录
BIN_DIR="./bin"
mkdir -p "$BIN_DIR"

# ldflags 去掉调试信息
LDFLAGS="-s -w"

# 架构列表
ARCHS=("arm64" "arm32" "amd64")  # arm64, arm32, x86_64

# 可执行文件前缀
NAME="led_neima"

for ARCH in "${ARCHS[@]}"; do
    echo "Building $NAME for $ARCH..."

    case $ARCH in
        arm64)
            GOARCH="arm64"
            ;;
        arm)
            GOARCH="arm32"
            ;;
        amd64)
            GOARCH="amd64"
            ;;
        *)
            echo "Unknown architecture: $ARCH"
            exit 1
            ;;
    esac

    # 设置 GOOS，如果需要交叉编译可改成 linux/darwin/windows
    GOOS=linux

    # 输出文件名
    OUTPUT="$BIN_DIR/${NAME}_${ARCH}"

    echo "GOOS=$GOOS GOARCH=$GOARCH go build -ldflags \"$LDFLAGS\" -o $OUTPUT"
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "$LDFLAGS" -o "$OUTPUT"
done

echo "All builds done."
