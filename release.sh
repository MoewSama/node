#!/bin/bash
# 打包脚本：构建前端、编译 slinx + sing-box，创建 GitHub Release 并上传
# 用法: ./release.sh <版本号> [changelog消息]
# 示例: ./release.sh v0.1.2 "修复 sing-box 更新 text file busy 问题"

set -euo pipefail
cd "$(dirname "$0")"

VERSION="${1:?用法: $0 <版本号> [changelog]}"
CHANGELOG="${2:-}"

# ── 检查依赖 ────────────────────────────────────────────────────────
command -v gh >/dev/null 2>&1 || { echo "错误: 需要安装 gh CLI (apt install gh)"; exit 1; }
command -v aarch64-linux-gnu-gcc >/dev/null 2>&1 || { echo "错误: 需要安装 gcc-aarch64-linux-gnu"; exit 1; }

# ── 获取 sing-box 版本 ──────────────────────────────────────────────
SING_BOX_VERSION=$(grep 'sagernet/sing-box' go.mod | head -1 | awk '{print $2}')
echo "=== 构建 slinx ${VERSION} (sing-box ${SING_BOX_VERSION}) ==="

# ── 构建前端 ─────────────────────────────────────────────────────────
echo "--- 构建前端 ---"
npm --prefix web ci --silent
npm --prefix web run build --silent

# ── 编译 ─────────────────────────────────────────────────────────────
echo "--- 编译二进制 ---"
mkdir -p dist

echo "  slinx_linux_amd64..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w -X main.Version=${VERSION}" \
  -o "dist/slinx_linux_amd64" .

echo "  slinx_linux_arm64..."
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  go build -ldflags="-s -w -X main.Version=${VERSION}" \
  -o "dist/slinx_linux_arm64" .

SING_BOX_TAGS='with_v2ray_api,with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_clash_api,with_cloudflared'

echo "  sing-box_linux_amd64..."
GOBIN=$(pwd)/dist GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go install -tags "$SING_BOX_TAGS" \
  -ldflags="-s -w -X github.com/sagernet/sing-box/constant.Version=${SING_BOX_VERSION}" \
  github.com/sagernet/sing-box/cmd/sing-box@${SING_BOX_VERSION}
mv dist/sing-box dist/sing-box_linux_amd64

echo "  sing-box_linux_arm64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go install -tags "$SING_BOX_TAGS" \
  -ldflags="-s -w -X github.com/sagernet/sing-box/constant.Version=${SING_BOX_VERSION}" \
  github.com/sagernet/sing-box/cmd/sing-box@${SING_BOX_VERSION}
# cross 编译产物在 GOPATH/bin/linux_arm64/sing-box
ARM_BIN=$(go env GOPATH)/bin/linux_arm64/sing-box
if [ -f "$ARM_BIN" ]; then
  mv "$ARM_BIN" dist/sing-box_linux_arm64
else
  echo "错误: 未找到 arm64 产物 $ARM_BIN"; exit 1
fi

# ── 压缩 sing-box ───────────────────────────────────────────────────
echo "--- 压缩 sing-box ---"
gzip -f -k "dist/sing-box_linux_amd64"
gzip -f -k "dist/sing-box_linux_arm64"

# 重命名为带版本号的格式（sing-box-版本号_linux_arch.gz）
mv "dist/sing-box_linux_amd64.gz" "dist/sing-box-${SING_BOX_VERSION}_linux_amd64.gz"
mv "dist/sing-box_linux_arm64.gz" "dist/sing-box-${SING_BOX_VERSION}_linux_arm64.gz"

echo ""
echo "=== 构建产物 ==="
ls -lh dist/

# ── 创建 Release ─────────────────────────────────────────────────────
echo "--- 发布 ${VERSION} ---"
NOTES="${CHANGELOG:-自动发布 ${VERSION}}"
gh release create "${VERSION}" \
  dist/slinx_linux_amd64 \
  dist/slinx_linux_arm64 \
  "dist/sing-box-${SING_BOX_VERSION}_linux_amd64.gz" \
  "dist/sing-box-${SING_BOX_VERSION}_linux_arm64.gz" \
  --repo MoewSama/node \
  --title "${VERSION}" \
  --notes "${NOTES}"

# ── 清理 ─────────────────────────────────────────────────────────────
echo "--- 清理构建产物 ---"
rm -rf dist web/dist web/node_modules

echo ""
echo "✅ 发布完成: https://github.com/MoewSama/node/releases/tag/${VERSION}"