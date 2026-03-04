# remote-gui

远端脚本执行框架 — 通过安全 GUI 触发远端预定义脚本并实时返回结果

[![executor CI](https://github.com/remote-gui/remote-gui/actions/workflows/executor-ci.yaml/badge.svg)](https://github.com/remote-gui/remote-gui/actions/workflows/executor-ci.yaml)
[![gui CI](https://github.com/remote-gui/remote-gui/actions/workflows/gui-ci.yaml/badge.svg)](https://github.com/remote-gui/remote-gui/actions/workflows/gui-ci.yaml)

---

## 5 分钟快速启动

### 前置要求

- Go 1.22+
- OpenSSL
- Bash

### 步骤

```bash
# 1. 生成开发用 mTLS 证书
make certs

# 2. 编译
make build

# 3. 启动 remote-executor（终端 1）
./remote-executor/dist/remote-executor \
  --config remote-executor/configs/executor.yaml \
  --scripts-dir remote-executor/scripts

# 4. 启动 GUI（终端 2）
./gui/dist/gui --config gui/configs/gui.yaml

# 5. 验证 executor API（终端 3）
curl --cert certs/gui.crt \
     --key  certs/gui.key \
     --cacert certs/ca.crt \
     https://localhost:8443/api/v1/scripts
```

---

## 架构概述

```
┌─────────────────┐         mTLS HTTPS          ┌──────────────────────┐
│      GUI        │ ──────────────────────────▶  │  remote-executor     │
│  (桌面 / CLI)   │  POST /api/v1/execute        │  (Go HTTP 服务)      │
│                 │ ◀──────────────────────────   │                      │
│  读取 gui.yaml  │  JSON 响应（stdout/stderr）   │  执行预定义脚本      │
└─────────────────┘                              └──────────────────────┘
```

| 组件 | 语言 | 说明 |
|------|------|------|
| **remote-executor** | Go 1.22 | 服务端，执行预定义脚本，mTLS HTTP API |
| **gui** | Go 1.22 (+ Fyne) | 客户端，操作选择 + 参数输入 + 结果展示 |

---

## 核心约束

- executor **只执行预定义脚本**，不接受任意命令
- 参数通过**环境变量**注入脚本（防命令注入）
- 严格参数校验（类型、格式、长度），规则随脚本定义
- 全链路 **mTLS 加密**（TLS 1.3），双向证书认证
- 每次调用在 executor 本地 `records/` 保留完整记录

---

## 内置示例：RocketMQ 消息查询

预定义脚本 `query-rocketmq-msg` 实现 RocketMQ 消息体查询：

```bash
# 通过 curl 调用（开发调试）
curl --cert certs/gui.crt --key certs/gui.key --cacert certs/ca.crt \
  -X POST https://localhost:8443/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "script": "query-rocketmq-msg",
    "params": {
      "topic": "test-topic",
      "message_id": "01DE76E4BEE026003309B83633000000D7"
    }
  }'
```

---

## 运行测试

```bash
# remote-executor（含 e2e 集成测试）
cd remote-executor && go test ./... -race -v

# GUI
cd gui && go test ./... -race -v

# 或使用 Makefile
make test
```

---

## 部署

- [systemd 部署](docs/deployment.md#systemd-部署生产)
- [Docker Compose 部署](docs/deployment.md#docker-compose-部署)
- [证书管理](docs/deployment.md#证书管理)

---

## 文档

| 文档 | 说明 |
|------|------|
| [docs/architecture.md](docs/architecture.md) | 系统架构、时序图、安全设计 |
| [docs/api-spec.md](docs/api-spec.md) | API 端点规范 |
| [docs/deployment.md](docs/deployment.md) | 部署指南 |

---

## 许可证

MIT
