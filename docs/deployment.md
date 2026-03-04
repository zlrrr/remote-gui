# 部署文档

## 目录

1. [快速开始（开发环境）](#快速开始开发环境)
2. [systemd 部署（生产）](#systemd-部署生产)
3. [Docker Compose 部署](#docker-compose-部署)
4. [证书管理](#证书管理)
5. [配置参考](#配置参考)

---

## 快速开始（开发环境）

### 前置要求

- Go 1.22+
- OpenSSL（生成证书）
- Bash

### 步骤

```bash
# 1. 克隆仓库
git clone <repo-url>
cd remote-gui

# 2. 生成开发证书
make certs

# 3. 编译两个程序
make build

# 4. 启动 remote-executor（新终端）
./remote-executor/dist/remote-executor \
  --config remote-executor/configs/executor.yaml \
  --scripts-dir remote-executor/scripts

# 5. 启动 GUI
./gui/dist/gui --config gui/configs/gui.yaml
```

---

## systemd 部署（生产）

### 1. 安装二进制

```bash
# 从 GitHub Releases 下载或自行编译
GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -o /usr/local/bin/remote-executor \
  ./remote-executor/cmd/executor/

chmod 755 /usr/local/bin/remote-executor
```

### 2. 创建用户和目录

```bash
useradd -r -s /bin/false -d /var/lib/remote-executor executor
usermod -aG docker executor  # 若脚本需要 docker 权限

mkdir -p /etc/remote-executor/certs
mkdir -p /opt/remote-executor/scripts
mkdir -p /var/lib/remote-executor/records

chown -R root:root /opt/remote-executor/scripts
chmod -R 755 /opt/remote-executor/scripts

chown -R executor:executor /var/lib/remote-executor/records
chmod 750 /var/lib/remote-executor/records
```

### 3. 安装配置和脚本

```bash
# 配置文件
cp remote-executor/configs/executor.yaml /etc/remote-executor/executor.yaml
# 编辑配置，设置正确的 TLS 证书路径

# 预定义脚本
cp -r remote-executor/scripts/query-rocketmq-msg /opt/remote-executor/scripts/
chmod +x /opt/remote-executor/scripts/*/run.sh

# TLS 证书
cp certs/ca.crt       /etc/remote-executor/certs/
cp certs/executor.crt /etc/remote-executor/certs/
cp certs/executor.key /etc/remote-executor/certs/
chmod 640 /etc/remote-executor/certs/*.key
chown root:executor /etc/remote-executor/certs/*.key
```

### 4. 安装 systemd unit

```bash
cp remote-executor/deployments/remote-executor.service \
   /etc/systemd/system/remote-executor.service

systemctl daemon-reload
systemctl enable remote-executor
systemctl start remote-executor

# 验证
systemctl status remote-executor
```

### 5. 验证

```bash
curl --cert certs/gui.crt \
     --key  certs/gui.key \
     --cacert certs/ca.crt \
     https://<server-ip>:8443/api/v1/scripts
# 期望返回脚本列表 JSON
```

---

## Docker Compose 部署

### 前置要求

- Docker 20.10+
- docker-compose v2

### 步骤

```bash
# 1. 生成证书
make certs

# 2. 配置 gui.yaml（修改 executor.endpoint 为实际地址）
vim gui/configs/gui.yaml

# 3. 启动服务
docker-compose up -d

# 4. 验证
docker-compose ps
docker-compose logs remote-executor
```

### 单独运行 executor

```bash
docker run -d \
  --name remote-executor \
  -p 8443:8443 \
  -v $(pwd)/certs:/etc/remote-executor/certs:ro \
  -v $(pwd)/remote-executor/scripts:/opt/remote-executor/scripts:ro \
  -v executor-records:/var/lib/remote-executor/records \
  ghcr.io/remote-gui/remote-executor:latest
```

---

## 证书管理

### 开发证书

```bash
make certs
# 生成 certs/{ca.crt, executor.crt, executor.key, gui.crt, gui.key}
# 注意：CA 私钥在生成后被删除，如需重签需重新生成 CA
```

### 生产证书建议

1. 使用企业 CA 或 Let's Encrypt（如服务暴露公网）
2. 将私钥权限设为 `640`，仅 root 和服务用户可读
3. 定期轮换（建议每年）
4. **不要**将生产私钥提交至版本控制（.gitignore 已配置忽略 `*.key`）

---

## 配置参考

### remote-executor/configs/executor.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8443

scripts:
  dir: "scripts"          # 脚本目录路径

records:
  dir: "records"          # 操作记录存储目录

tls:
  ca_cert:     "certs/ca.crt"
  server_cert: "certs/executor.crt"
  server_key:  "certs/executor.key"
```

### gui/configs/gui.yaml

```yaml
executor:
  endpoint: "https://192.168.0.10:8443"
  tls:
    ca_cert:     "certs/ca.crt"
    client_cert: "certs/gui.crt"
    client_key:  "certs/gui.key"

operations:           # 最多 10 个操作
  - alias: "查询 RocketMQ 消息"
    script: "query-rocketmq-msg"
    params:           # 每个操作最多 5 个参数
      - label: "Topic 名称"
        name: topic
        placeholder: "e.g. test-topic"
      - label: "Message ID"
        name: message_id
        placeholder: "e.g. 01DE76E4BEE026..."
```
