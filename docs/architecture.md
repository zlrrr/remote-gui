# remote-gui 架构设计

## 系统概述

remote-gui 由两个独立程序组成，通过 mTLS HTTPS 安全通信：

```
┌─────────────────┐         mTLS HTTPS          ┌──────────────────────┐
│      GUI        │  ─────────────────────────▶ │  remote-executor     │
│  (Fyne 桌面)    │  POST /api/v1/execute        │  (Go HTTP 服务)      │
│                 │  ◀─────────────────────────  │                      │
│  读取 gui.yaml  │  JSON 响应（stdout/stderr）   │  执行预定义脚本      │
└─────────────────┘                              └──────────────────────┘
```

## 组件说明

### remote-executor（服务端）

- **职责**：加载预定义脚本，暴露 HTTP API，安全执行脚本，记录操作日志
- **语言**：Go 1.22+
- **HTTP 框架**：`net/http` + `chi` router
- **安全**：mTLS 双向认证（TLS 1.3），参数校验，环境变量注入（防命令注入）

### gui（客户端）

- **职责**：读取配置文件，展示操作列表，填写参数后调用 executor，展示结果
- **语言**：Go 1.22+，UI 框架：`fyne.io/fyne/v2`

## 时序图

```
GUI                        remote-executor                 Script
 │                               │                           │
 │  POST /api/v1/execute         │                           │
 │  (携带 mTLS 客户端证书)        │                           │
 │──────────────────────────────▶│                           │
 │                               │  1. 验证客户端证书         │
 │                               │  2. 查找脚本定义           │
 │                               │  3. 校验参数（pattern/长度）│
 │                               │  4. 注入环境变量           │
 │                               │──────────────────────────▶│
 │                               │                           │ 执行 run.sh
 │                               │                           │ (PARAM_XXX 环境变量)
 │                               │◀──────────────────────────│
 │                               │  stdout/stderr/exit_code  │
 │                               │  5. 写入操作记录           │
 │◀──────────────────────────────│                           │
 │  JSON 响应（status/stdout...） │                           │
```

## 安全设计

### mTLS 双向认证

```
CA (自签)
 ├── executor.crt  —— remote-executor 服务端证书
 └── gui.crt       —— GUI 客户端证书
```

- executor 只接受由同一 CA 签发的客户端证书
- 所有流量 TLS 1.3 加密，不支持降级
- 证书由 `scripts/gen-certs.sh` 统一生成

### 脚本注入防护

- 参数只作为**环境变量**注入脚本（`cmd.Env`），不拼接进 shell 命令字符串
- `spec.yaml` 的 `pattern` 正则在服务端校验，校验通过前不执行
- 脚本目录**只读挂载**，executor 进程无写权限

## 目录结构

```
remote-gui/
├── remote-executor/         # 服务端
│   ├── cmd/executor/        # 程序入口
│   ├── internal/
│   │   ├── config/          # 配置加载
│   │   ├── script/          # 脚本加载器 + 参数校验引擎
│   │   ├── runner/          # 脚本执行器
│   │   ├── api/             # HTTP 处理器
│   │   ├── record/          # 操作记录存储
│   │   └── server/          # HTTP 服务启动
│   ├── scripts/             # 预定义脚本目录
│   └── configs/             # 服务配置
├── gui/                     # 客户端
│   ├── cmd/gui/             # 程序入口
│   ├── internal/
│   │   ├── config/          # GUI 配置加载
│   │   ├── client/          # mTLS HTTP 客户端
│   │   └── ui/              # Fyne UI 组件
│   └── configs/             # GUI 配置
├── certs/                   # TLS 证书
├── scripts/                 # 工具脚本
└── docs/                    # 文档
```
