#!/usr/bin/env bash
# gen-certs.sh — 生成开发用自签 mTLS 证书
# 产生：ca.crt, executor.crt/key, gui.crt/key
# 生产环境请替换为正式 CA 签发的证书，并将 certs/ 加入 .gitignore

set -euo pipefail

CERTS_DIR="$(cd "$(dirname "$0")/.." && pwd)/certs"
mkdir -p "${CERTS_DIR}"
cd "${CERTS_DIR}"

DAYS=3650   # 10 年有效期（仅供开发）
BITS=4096

echo "[gen-certs] 生成目录: ${CERTS_DIR}"

# ── 1. CA ──────────────────────────────────────────────────────────────────
echo "[gen-certs] 1/5 生成 CA 私钥和自签证书..."
openssl genrsa -out ca.key "${BITS}"
openssl req -new -x509 -days "${DAYS}" \
  -key ca.key \
  -subj "/CN=remote-gui-CA/O=remote-gui/C=CN" \
  -out ca.crt

# ── 2. executor 服务端证书 ──────────────────────────────────────────────────
echo "[gen-certs] 2/5 生成 executor 私钥..."
openssl genrsa -out executor.key "${BITS}"

echo "[gen-certs] 3/5 生成 executor CSR 和证书..."
openssl req -new \
  -key executor.key \
  -subj "/CN=remote-executor/O=remote-gui/C=CN" \
  -out executor.csr

cat > executor-ext.cnf <<EOF
[SAN]
subjectAltName=DNS:localhost,IP:127.0.0.1
EOF

openssl x509 -req -days "${DAYS}" \
  -in executor.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -extfile executor-ext.cnf -extensions SAN \
  -out executor.crt

# ── 3. GUI 客户端证书 ───────────────────────────────────────────────────────
echo "[gen-certs] 4/5 生成 GUI 私钥..."
openssl genrsa -out gui.key "${BITS}"

echo "[gen-certs] 5/5 生成 GUI CSR 和证书..."
openssl req -new \
  -key gui.key \
  -subj "/CN=remote-gui-client/O=remote-gui/C=CN" \
  -out gui.csr

openssl x509 -req -days "${DAYS}" \
  -in gui.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out gui.crt

# ── 清理临时文件 ──────────────────────────────────────────────────────────
rm -f ca.key ca.srl executor.csr executor-ext.cnf gui.csr

echo ""
echo "[gen-certs] 完成！生成的文件："
ls -lh "${CERTS_DIR}"
echo ""
echo "  CA 证书:          certs/ca.crt"
echo "  executor 证书:    certs/executor.crt + executor.key"
echo "  GUI 客户端证书:   certs/gui.crt + gui.key"
echo ""
echo "注意：CA 私钥已删除，如需重新签发证书需重新生成 CA。"
