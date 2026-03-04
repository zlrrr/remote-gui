# remote-gui

远端脚本执行框架 — 通过安全 GUI 触发远端预定义脚本并实时返回结果

基于 TDD/SDD 原则，分阶段构建可长期迭代的生产级项目

---

## 项目概述

### 目标

`remote-gui` 由两个独立程序组成：

| 组件 | 角色 | 语言 | 运行方式 |
|------|------|------|----------|
| **remote-executor** | 服务端，执行预定义脚本 | Go | 系统服务（systemd / Docker） |
| **gui** | 客户端，提供操作界面 | Go (Fyne) / Web | 桌面 App 或浏览器 |

两者通过 **mTLS + HTTPS** 安全通信，所有可执行脚本由维护人员预先部署，GUI 只能触发已注册的操作，不能上传或注入新命令。

### MVP 目标

- remote-executor 以服务形式启动，加载预定义脚本，暴露安全 HTTP API
- GUI 读取配置文件，展示操作列表，填写参数后调用 executor，展示结果
- 内置 RocketMQ 消息查询脚本作为默认示例，验证完整链路

### 核心约束

- ✅ executor **只执行预定义脚本**，不接受任意命令
- ✅ 严格参数校验（类型、格式、长度），规则随脚本一起定义
- ✅ GUI 最多 10 个操作，每个操作最多 5 个参数
- ✅ executor 本地保留每次调用的操作记录和结果
- ✅ 全链路 mTLS 加密，证书由项目统一管理
- ✅ GitHub Actions 自动打包二进制 / Docker 镜像

---

## 一、技术栈

### remote-executor（服务端）

| 类别 | 选型 |
|------|------|
| 语言 | Go 1.22+ |
| HTTP 框架 | `net/http` + `chi` router |
| 配置解析 | `gopkg.in/yaml.v3` |
| 日志 | `go.uber.org/zap` |
| TLS | Go 标准库 `crypto/tls`，mTLS 双向认证 |
| 测试 | `testify`, `httptest` |
| 服务管理 | systemd unit 文件 |

### gui（客户端）

| 类别 | 选型 |
|------|------|
| 语言 | Go 1.22+ |
| UI 框架 | `fyne.io/fyne/v2`（跨平台桌面） |
| HTTP 客户端 | Go 标准库，配置 mTLS |
| 配置解析 | `gopkg.in/yaml.v3` |
| 测试 | `testify` |

> **备注**：如团队后续希望改为 Web GUI，可在 Phase 5 切换为 React + TypeScript 前端，executor API 接口不变。

### DevOps

| 类别 | 选型 |
|------|------|
| 容器化 | Docker（多阶段构建） |
| CI/CD | GitHub Actions |
| 证书管理 | `scripts/gen-certs.sh`（基于 OpenSSL） |

---

## 二、项目结构（最终目标）

```
remote-gui/
├── remote-executor/
│   ├── cmd/
│   │   └── executor/
│   │       └── main.go               # 程序入口
│   ├── internal/
│   │   ├── config/
│   │   │   ├── config.go             # 配置加载
│   │   │   └── config_test.go
│   │   ├── script/
│   │   │   ├── loader.go             # 脚本加载器
│   │   │   ├── loader_test.go
│   │   │   ├── validator.go          # 参数校验引擎
│   │   │   └── validator_test.go
│   │   ├── runner/
│   │   │   ├── runner.go             # 脚本执行器
│   │   │   └── runner_test.go
│   │   ├── api/
│   │   │   ├── handler.go            # HTTP 处理器
│   │   │   ├── handler_test.go
│   │   │   └── middleware.go         # mTLS 认证中间件
│   │   ├── record/
│   │   │   ├── store.go              # 操作记录存储（本地 JSON）
│   │   │   └── store_test.go
│   │   └── server/
│   │       └── server.go             # HTTP 服务启动
│   ├── scripts/                      # 预定义脚本目录（维护人员管理）
│   │   └── query-rocketmq-msg/
│   │       ├── run.sh                # 可执行脚本
│   │       └── spec.yaml             # 参数校验规则
│   ├── configs/
│   │   └── executor.yaml             # 服务配置
│   ├── deployments/
│   │   ├── Dockerfile
│   │   └── remote-executor.service   # systemd unit
│   ├── go.mod
│   └── go.sum
│
├── gui/
│   ├── cmd/
│   │   └── gui/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   │   ├── config.go             # 读取 gui.yaml
│   │   │   └── config_test.go
│   │   ├── client/
│   │   │   ├── executor_client.go    # mTLS HTTP 客户端
│   │   │   └── executor_client_test.go
│   │   └── ui/
│   │       ├── app.go                # Fyne App 入口
│   │       ├── operation_panel.go    # 操作选择 + 参数输入
│   │       └── result_panel.go       # 结果展示
│   ├── configs/
│   │   └── gui.yaml                  # 操作别名 + 参数定义（≤10 操作，≤5 参数/操作）
│   ├── deployments/
│   │   └── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── certs/                            # TLS 证书（gitignore 生产证书）
│   ├── ca.crt
│   ├── executor.crt / executor.key
│   └── gui.crt / gui.key
│
├── scripts/
│   └── gen-certs.sh                  # 证书生成脚本
│
├── .github/
│   └── workflows/
│       ├── executor-ci.yaml          # remote-executor CI + 打包
│       └── gui-ci.yaml               # gui CI + 打包
│
├── docs/
│   ├── architecture.md
│   ├── api-spec.md
│   └── deployment.md
│
├── docker-compose.yml                # 本地联调
├── Makefile
└── README.md
```

---

## 三、API 规范

### Base URL

```
https://<executor-host>:<port>/api/v1
```

所有请求均需携带客户端证书（mTLS）。

### 接口列表

#### `GET /scripts`

返回所有已加载的脚本列表及其参数规范。

**响应示例**：

```json
{
  "scripts": [
    {
      "name": "query-rocketmq-msg",
      "description": "查询 RocketMQ 消息体",
      "params": [
        { "name": "topic",      "type": "string", "required": true,  "pattern": "^[a-zA-Z0-9_-]{1,64}$" },
        { "name": "message_id", "type": "string", "required": true,  "pattern": "^[A-F0-9]{32,40}$" }
      ]
    }
  ]
}
```

#### `POST /execute`

触发指定脚本执行。

**请求体**：

```json
{
  "script": "query-rocketmq-msg",
  "params": {
    "topic":      "test-topic",
    "message_id": "01DE76E4BEE026003309B83633000000D7"
  }
}
```

**响应体**：

```json
{
  "record_id":  "rec-20250301-143022-abc123",
  "script":     "query-rocketmq-msg",
  "status":     "success",
  "exit_code":  0,
  "stdout":     "...",
  "stderr":     "",
  "duration_ms": 1240,
  "executed_at": "2025-03-01T14:30:22Z"
}
```

**错误响应**（参数校验失败）：

```json
{
  "error": "validation_failed",
  "details": [
    { "param": "message_id", "reason": "格式不匹配，期望 ^[A-F0-9]{32,40}$" }
  ]
}
```

#### `GET /records`

查询本地操作记录列表（支持分页）。

#### `GET /records/{record_id}`

查询单条操作记录详情。

---

## 四、脚本规范 — 内置示例

### 脚本目录结构

```
remote-executor/scripts/query-rocketmq-msg/
├── run.sh       # 可执行脚本
└── spec.yaml    # 参数校验规则
```

### `run.sh`

```bash
#!/usr/bin/env bash
# query-rocketmq-msg/run.sh
# 参数由 executor 注入为环境变量：PARAM_TOPIC, PARAM_MESSAGE_ID

set -euo pipefail

TOPIC="${PARAM_TOPIC}"
MSG_ID="${PARAM_MESSAGE_ID}"
NAMESRV="${ROCKETMQ_NAMESRV:-192.168.0.43:31430}"

docker run --rm --privileged --network host \
  --entrypoint bash apache/rocketmq:5.1.4 \
  -c "/home/rocketmq/rocketmq-5.1.4/bin/mqadmin queryMsgByUniqueKey \
      -n ${NAMESRV} \
      -t ${TOPIC} \
      -i ${MSG_ID} \
      && echo \
      && cat /tmp/rocketmq/msgbodys/${MSG_ID}"
```

### `spec.yaml`

```yaml
# query-rocketmq-msg/spec.yaml

name: query-rocketmq-msg
description: "查询 RocketMQ 指定 Topic 下的消息体"
timeout_seconds: 60

params:
  - name: topic
    description: "消息所在的 Topic 名称"
    type: string
    required: true
    rules:
      pattern: '^[a-zA-Z0-9_\-]{1,64}$'
      min_length: 1
      max_length: 64

  - name: message_id
    description: "消息的 UniqueKey（Message ID）"
    type: string
    required: true
    rules:
      pattern: '^[A-F0-9]{32,40}$'
      min_length: 32
      max_length: 40
```

### GUI 配置示例

```yaml
# gui/configs/gui.yaml

executor:
  endpoint: "https://192.168.0.10:8443"
  tls:
    ca_cert:     "certs/ca.crt"
    client_cert: "certs/gui.crt"
    client_key:  "certs/gui.key"

operations:           # 最多 10 个操作
  - alias: "查询 RocketMQ 消息"
    script: "query-rocketmq-msg"
    params:           # 最多 5 个参数
      - label: "Topic 名称"
        name:  topic
        placeholder: "e.g. test-topic"
      - label: "Message ID"
        name:  message_id
        placeholder: "e.g. 01DE76E4BEE026..."
```

---

## 五、安全设计

### mTLS 双向认证

```
CA (自签)
 ├── executor.crt  —— remote-executor 服务端证书
 └── gui.crt       —— GUI 客户端证书
```

- executor 只接受由同一 CA 签发的客户端证书
- GUI 只信任由同一 CA 签发的服务端证书
- 所有流量 TLS 1.3 加密，不支持降级

### 脚本注入防护

- 参数只作为**环境变量**注入脚本，不拼接进 shell 命令字符串
- spec.yaml 的 `pattern` 正则在服务端校验，校验通过前不执行
- 脚本目录**只读挂载**，executor 进程无写权限

### 操作记录

- 每次调用后，executor 将完整请求 + 响应写入本地 `records/` 目录（JSON Lines）
- 记录包含：时间戳、脚本名、入参（脱敏）、退出码、stdout/stderr 前 4KB

---

## 六、阶段划分（严格 TDD/SDD 流程）

> **工作约定**
> - 每个检查点必须：① 所有测试通过 ② `git commit` + `git tag` 后再继续
> - Tag 格式：`phase-X.Y`
> - 每次会话开始，先执行 `git log --oneline -10` 确认当前进度，从最新 tag 继续

---

### Phase 0 — 项目初始化与架构设计（0.5 天）

**目标**：搭好骨架，定义接口，不写业务逻辑。

**任务**：

1. 初始化 monorepo 目录结构（`remote-executor/`、`gui/`、`docs/`、`.github/`）
2. 编写 `docs/architecture.md` 和 `docs/api-spec.md`
3. 在两个子模块中定义核心接口（空实现），让编译通过
4. 编写证书生成脚本 `scripts/gen-certs.sh`，生成开发用自签证书

**交付物**：

```
remote-gui/
├── remote-executor/go.mod + go.sum（空实现编译通过）
├── gui/go.mod + go.sum（空实现编译通过）
├── certs/（开发用证书，gitignore 生产证书）
├── scripts/gen-certs.sh
├── docs/architecture.md
├── docs/api-spec.md
└── Makefile（make certs / make build / make test）
```

**检查点 #0.1 — 骨架验证**：

```bash
# 两个模块编译通过
cd remote-executor && go build ./...
cd ../gui          && go build ./...

# 证书生成成功
bash scripts/gen-certs.sh
ls certs/  # ca.crt executor.crt executor.key gui.crt gui.key

git add .
git commit -m "Phase 0: 项目骨架初始化，接口定义，证书脚本"
git tag phase-0.1
```

**进入下一阶段前确认**：

- [ ] 两个模块 `go build ./...` 无报错
- [ ] 证书文件完整生成
- [ ] `docs/api-spec.md` 覆盖全部 API 端点

---

### Phase 1 — remote-executor 核心：脚本加载与参数校验（1 天）

**目标**：实现脚本加载器和参数校验引擎，TDD 驱动，不涉及 HTTP。

#### Phase 1.1 — 参数校验引擎

**先写测试**：

```go
// internal/script/validator_test.go

func TestValidateParam_String_PatternOK(t *testing.T) {
    rule := ParamRule{Type: "string", Pattern: `^[a-zA-Z0-9_\-]{1,64}$`}
    err := ValidateParam("test-topic", rule)
    assert.NoError(t, err)
}

func TestValidateParam_String_PatternFail(t *testing.T) {
    rule := ParamRule{Type: "string", Pattern: `^[a-zA-Z0-9_\-]{1,64}$`}
    err := ValidateParam("test; rm -rf /", rule)
    assert.Error(t, err)
}

func TestValidateParam_Required_Empty(t *testing.T) {
    rule := ParamRule{Required: true}
    err := ValidateParam("", rule)
    assert.Error(t, err)
}

func TestValidateParam_MaxLength(t *testing.T) {
    rule := ParamRule{Type: "string", MaxLength: 5}
    err := ValidateParam("toolong", rule)
    assert.Error(t, err)
}
```

**再实现**：`internal/script/validator.go`

**检查点 #1.1**：

```bash
cd remote-executor
go test ./internal/script/... -v -run Validator
# 所有 Validator 测试通过

git commit -m "Phase 1.1: 参数校验引擎 TDD 实现"
git tag phase-1.1
```

---

#### Phase 1.2 — 脚本加载器

**先写测试**：

```go
// internal/script/loader_test.go

func TestLoadScripts_Success(t *testing.T) {
    // 使用 testdata/scripts/ 目录
    registry, err := LoadScripts("testdata/scripts")
    assert.NoError(t, err)
    assert.Contains(t, registry, "query-rocketmq-msg")
}

func TestLoadScripts_MissingSpec(t *testing.T) {
    // spec.yaml 缺失时应报错
    _, err := LoadScripts("testdata/missing-spec")
    assert.Error(t, err)
}

func TestLoadScripts_InvalidSpec(t *testing.T) {
    // spec.yaml 格式错误时应报错
    _, err := LoadScripts("testdata/invalid-spec")
    assert.Error(t, err)
}
```

准备 `testdata/scripts/query-rocketmq-msg/`（与生产脚本相同）。

**再实现**：`internal/script/loader.go`

**检查点 #1.2**：

```bash
go test ./internal/script/... -v
# 全部通过

git commit -m "Phase 1.2: 脚本加载器 TDD 实现，含内置 RocketMQ 示例脚本"
git tag phase-1.2
```

**进入下一阶段前确认**：

- [ ] `go test ./internal/script/...` 全绿
- [ ] `scripts/query-rocketmq-msg/` 目录及 `spec.yaml` 完整
- [ ] 加载器正确处理缺失 / 格式错误的 spec

---

### Phase 2 — remote-executor 核心：脚本执行器（1 天）

**目标**：实现安全的脚本执行，参数以环境变量注入，防注入。

#### Phase 2.1 — Runner 单元测试

**先写测试**：

```go
// internal/runner/runner_test.go

func TestRun_Success(t *testing.T) {
    r := NewRunner()
    result, err := r.Run(context.Background(), RunRequest{
        ScriptPath: "testdata/echo-params.sh",
        Params:     map[string]string{"PARAM_TOPIC": "test-topic"},
        TimeoutSec: 10,
    })
    assert.NoError(t, err)
    assert.Equal(t, 0, result.ExitCode)
    assert.Contains(t, result.Stdout, "test-topic")
}

func TestRun_Timeout(t *testing.T) {
    r := NewRunner()
    _, err := r.Run(context.Background(), RunRequest{
        ScriptPath: "testdata/sleep.sh",
        TimeoutSec: 1,
    })
    assert.ErrorIs(t, err, ErrTimeout)
}

func TestRun_ScriptNotFound(t *testing.T) {
    r := NewRunner()
    _, err := r.Run(context.Background(), RunRequest{
        ScriptPath: "testdata/not-exist.sh",
    })
    assert.Error(t, err)
}
```

准备 `testdata/echo-params.sh`（打印 `$PARAM_TOPIC`）和 `testdata/sleep.sh`（`sleep 30`）。

**再实现**：`internal/runner/runner.go`，重点：

- 使用 `exec.CommandContext` 支持超时取消
- 参数**仅**通过 `cmd.Env` 注入，不拼接命令行
- stdout / stderr 截断至 4KB

**检查点 #2.1**：

```bash
go test ./internal/runner/... -v
git commit -m "Phase 2.1: 脚本执行器 TDD 实现，环境变量注入，超时控制"
git tag phase-2.1
```

---

#### Phase 2.2 — 操作记录存储

**先写测试**：

```go
// internal/record/store_test.go

func TestSave_And_Get(t *testing.T) {
    dir := t.TempDir()
    store := NewFileStore(dir)

    rec := Record{
        Script:     "query-rocketmq-msg",
        Params:     map[string]string{"topic": "t1"},
        ExitCode:   0,
        Stdout:     "result",
        ExecutedAt: time.Now(),
    }

    id, err := store.Save(rec)
    assert.NoError(t, err)
    assert.NotEmpty(t, id)

    got, err := store.Get(id)
    assert.NoError(t, err)
    assert.Equal(t, rec.Script, got.Script)
}

func TestList_Pagination(t *testing.T) {
    // 插入 5 条，每页 2 条，验证分页正确
}
```

**再实现**：`internal/record/store.go`（JSON Lines 文件存储）

**检查点 #2.2**：

```bash
go test ./internal/record/... -v
git commit -m "Phase 2.2: 操作记录存储 TDD 实现（JSON Lines）"
git tag phase-2.2
```

**进入下一阶段前确认**：

- [ ] `go test ./internal/runner/...` 全绿，包含超时场景
- [ ] `go test ./internal/record/...` 全绿
- [ ] 参数注入代码审查：确认无字符串拼接

---

### Phase 3 — remote-executor HTTP 服务（1.5 天）

**目标**：将 Phase 1-2 的核心逻辑通过 mTLS HTTP 暴露，TDD 驱动 handler。

#### Phase 3.1 — HTTP Handler 测试（使用 `httptest`，不启动真实 TLS）

**先写测试**：

```go
// internal/api/handler_test.go

func TestExecuteHandler_Success(t *testing.T) {
    mockRunner := &MockRunner{Result: &RunResult{ExitCode: 0, Stdout: "ok"}}
    h := NewHandler(mockScriptRegistry, mockRunner, mockStore)
    
    body := `{"script":"query-rocketmq-msg","params":{"topic":"t1","message_id":"A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4"}}`
    req := httptest.NewRequest("POST", "/api/v1/execute", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    h.Execute(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    var resp ExecuteResponse
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "success", resp.Status)
}

func TestExecuteHandler_ValidationFail(t *testing.T) {
    // message_id 格式错误，期望 422
    body := `{"script":"query-rocketmq-msg","params":{"topic":"t1","message_id":"invalid!!"}}`
    // ... 断言 422 + validation_failed
}

func TestExecuteHandler_UnknownScript(t *testing.T) {
    // 请求不存在的脚本，期望 404
}

func TestListScriptsHandler(t *testing.T) {
    // 期望返回已注册脚本列表
}
```

**再实现**：`internal/api/handler.go`

**检查点 #3.1**：

```bash
go test ./internal/api/... -v
git commit -m "Phase 3.1: HTTP Handler TDD 实现，含参数校验与错误响应"
git tag phase-3.1
```

---

#### Phase 3.2 — mTLS 服务器启动

**实现**：`internal/server/server.go`

```go
// 关键配置
tlsCfg := &tls.Config{
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    caPool,
    MinVersion:   tls.VersionTLS13,
}
```

**集成测试**（使用测试证书）：

```go
// internal/server/server_test.go

func TestServer_mTLS_Accepted(t *testing.T) {
    // 使用 testdata/certs/ 启动测试服务器
    // 携带正确客户端证书发起请求 → 200
}

func TestServer_mTLS_Rejected_NoCert(t *testing.T) {
    // 不携带客户端证书 → TLS 握手失败
}
```

**检查点 #3.2**：

```bash
go test ./internal/server/... -v
go build ./cmd/executor/...  # 编译成功

git commit -m "Phase 3.2: mTLS 服务器集成，TLS 1.3，双向认证"
git tag phase-3.2
```

**进入下一阶段前确认**：

- [ ] `go test ./...` 在 remote-executor 下全绿
- [ ] `curl --cert certs/gui.crt --key certs/gui.key --cacert certs/ca.crt https://localhost:8443/api/v1/scripts` 返回脚本列表
- [ ] 不携带证书时连接被拒绝（TLS 握手失败）

---

### Phase 4 — GUI 客户端（1.5 天）

**目标**：实现 GUI 的 HTTP 客户端和基础 UI，能完整调用 executor。

#### Phase 4.1 — mTLS HTTP 客户端

**先写测试**：

```go
// internal/client/executor_client_test.go
// 启动 httptest.TLSServer 模拟 executor

func TestClient_Execute_Success(t *testing.T) {
    // mock server 返回 200
    client := NewExecutorClient(ExecutorClientConfig{...})
    result, err := client.Execute("query-rocketmq-msg", map[string]string{...})
    assert.NoError(t, err)
    assert.Equal(t, "success", result.Status)
}

func TestClient_Execute_ValidationError(t *testing.T) {
    // mock server 返回 422 → 客户端封装为 ValidationError
}

func TestClient_ListScripts(t *testing.T) {
    // 验证返回的脚本列表结构
}
```

**再实现**：`internal/client/executor_client.go`

**检查点 #4.1**：

```bash
cd gui
go test ./internal/client/... -v
git commit -m "Phase 4.1: GUI mTLS HTTP 客户端 TDD 实现"
git tag phase-4.1
```

---

#### Phase 4.2 — 配置加载

**先写测试**：

```go
// internal/config/config_test.go

func TestLoadConfig_OperationLimit(t *testing.T) {
    // 超过 10 个操作时应报错
}

func TestLoadConfig_ParamLimit(t *testing.T) {
    // 超过 5 个参数时应报错
}

func TestLoadConfig_ValidConfig(t *testing.T) {
    cfg, err := LoadConfig("testdata/gui.yaml")
    assert.NoError(t, err)
    assert.Len(t, cfg.Operations, 1)
}
```

**检查点 #4.2**：

```bash
go test ./internal/config/... -v
git commit -m "Phase 4.2: GUI 配置加载，操作/参数上限校验"
git tag phase-4.2
```

---

#### Phase 4.3 — Fyne UI

**实现**：`internal/ui/`

UI 布局（MVP）：

```
┌─────────────────────────────────┐
│  remote-gui                     │
│                                 │
│  操作: [查询 RocketMQ 消息  ▾]  │
│                                 │
│  Topic 名称:    [____________]  │
│  Message ID:    [____________]  │
│                                 │
│           [ 执行 ]              │
│                                 │
│  ─────────── 结果 ──────────── │
│  状态: success  耗时: 1240ms    │
│  ┌─────────────────────────┐   │
│  │ stdout 内容...          │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
```

**检查点 #4.3**：

```bash
go build ./cmd/gui/...   # 编译成功
# 手动测试：启动 GUI，填写参数，点击执行，看到结果

git commit -m "Phase 4.3: Fyne GUI MVP 实现，操作选择 + 参数输入 + 结果展示"
git tag phase-4.3
```

**进入下一阶段前确认**：

- [ ] `go test ./...` 在 gui 下全绿
- [ ] GUI 启动后能正确读取 `gui.yaml` 操作列表
- [ ] 执行操作后结果正确展示
- [ ] 参数填写错误时 GUI 显示校验失败信息（来自 executor 422 响应）

---

### Phase 5 — 端到端集成测试（1 天）

**目标**：executor + GUI 联调，完整链路验证，使用内置 RocketMQ 脚本（可 mock docker 调用）。

#### Phase 5.1 — 集成测试脚本（Mock 版）

```bash
# scripts/integration-test.sh
# 启动 executor（使用测试证书）
# 调用 /api/v1/execute 并断言返回
# 验证本地 records/ 生成了记录
```

```go
// tests/integration/e2e_test.go

func TestE2E_ExecuteScript(t *testing.T) {
    // 1. 启动 executor 进程（测试证书）
    // 2. GUI 客户端发起请求
    // 3. 断言响应 status == "success"
    // 4. 断言 records/ 目录新增了一条记录
}
```

**检查点 #5.1**：

```bash
go test ./tests/integration/... -v -timeout 60s
git commit -m "Phase 5.1: 端到端集成测试，executor + GUI 客户端全链路"
git tag phase-5.1
```

---

#### Phase 5.2 — 冒烟测试（真实环境可选）

```bash
# 若有可用的 RocketMQ 环境，执行真实脚本
curl --cert certs/gui.crt --key certs/gui.key --cacert certs/ca.crt \
  -X POST https://localhost:8443/api/v1/execute \
  -d '{"script":"query-rocketmq-msg","params":{"topic":"test-topic","message_id":"01DE76E4BEE026003309B83633000000D7"}}'
```

**检查点 #5.2**：

```bash
git commit -m "Phase 5.2: 冒烟测试通过（或 mock 验证完整链路）"
git tag phase-5.2
```

**进入下一阶段前确认**：

- [ ] 全链路 e2e 测试通过
- [ ] 操作记录正确写入磁盘
- [ ] mTLS 认证在集成环境下正常工作

---

### Phase 6 — GitHub Actions CI/CD（0.5 天）

**目标**：自动化测试 + 打包二进制 + 推送 Docker 镜像。

#### `.github/workflows/executor-ci.yaml`

```yaml
name: remote-executor CI

on:
  push:
    paths: ["remote-executor/**"]
  pull_request:
    paths: ["remote-executor/**"]

jobs:
  test:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: remote-executor
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - name: Run tests
        run: go test ./... -v -race -coverprofile=coverage.out
      - name: Upload coverage
        uses: codecov/codecov-action@v4

  build:
    needs: test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [linux, darwin, windows]
        arch: [amd64, arm64]
    defaults:
      run:
        working-directory: remote-executor
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - name: Build binary
        env:
          GOOS: ${{ matrix.os }}
          GOARCH: ${{ matrix.arch }}
        run: |
          go build -ldflags="-s -w" \
            -o dist/remote-executor-${{ matrix.os }}-${{ matrix.arch }} \
            ./cmd/executor/
      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: remote-executor-${{ matrix.os }}-${{ matrix.arch }}
          path: remote-executor/dist/

  docker:
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v5
        with:
          context: remote-executor
          file: remote-executor/deployments/Dockerfile
          push: true
          tags: ghcr.io/${{ github.repository }}/remote-executor:latest
```

#### `.github/workflows/gui-ci.yaml`

结构与 executor 相同，针对 `gui/` 路径，同样输出多平台二进制和 Docker 镜像。

**Dockerfile 示例（remote-executor）**：

```dockerfile
# remote-executor/deployments/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /remote-executor ./cmd/executor/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /remote-executor /usr/local/bin/remote-executor
COPY scripts/ /opt/remote-executor/scripts/
EXPOSE 8443
ENTRYPOINT ["/usr/local/bin/remote-executor"]
```

**检查点 #6.1**：

```bash
# 推送到 main，验证 Actions 全部绿
# 确认 Releases / Artifacts 包含各平台二进制
# 确认 ghcr.io 镜像可拉取并启动

git commit -m "Phase 6: GitHub Actions CI/CD，多平台二进制 + Docker 镜像"
git tag phase-6.1
```

**进入下一阶段前确认**：

- [ ] `executor-ci` / `gui-ci` Actions 全绿
- [ ] Artifacts 中包含 linux/darwin/windows × amd64/arm64 二进制
- [ ] `docker pull ghcr.io/<repo>/remote-executor:latest` 成功

---

### Phase 7 — systemd 部署与文档（0.5 天）

**目标**：提供生产部署方案和完整文档。

#### systemd unit 文件

```ini
# remote-executor/deployments/remote-executor.service

[Unit]
Description=remote-executor Script Execution Service
After=network.target

[Service]
Type=simple
User=executor
ExecStart=/usr/local/bin/remote-executor \
  --config /etc/remote-executor/executor.yaml \
  --scripts-dir /opt/remote-executor/scripts
Restart=on-failure
RestartSec=5
# 只读脚本目录，防止脚本被修改
ReadOnlyPaths=/opt/remote-executor/scripts
ReadWritePaths=/var/lib/remote-executor/records

[Install]
WantedBy=multi-user.target
```

**文档**：

- `docs/deployment.md`：systemd 安装步骤 + Docker Compose 部署步骤
- `docs/architecture.md`：补全时序图（GUI → executor → script → 返回结果）
- `README.md`：快速开始（5 分钟运行 MVP）

**检查点 #7.1**：

```bash
# 按文档步骤在干净 VM 上安装 executor 并验证
systemctl status remote-executor  # active (running)
curl --cert ... https://<vm-ip>:8443/api/v1/scripts  # 返回脚本列表

git commit -m "Phase 7: systemd 部署文件，完整文档"
git tag phase-7.1
```

**MVP 完成确认**：

- [ ] executor 以 systemd 服务运行，重启后自动恢复
- [ ] GUI 连接远端 executor，完整执行 RocketMQ 查询脚本
- [ ] `docs/` 文档完整覆盖部署和使用流程
- [ ] GitHub Actions 自动化打包全部通过

---

## 七、Makefile 参考

```makefile
.PHONY: certs test build docker

certs:
	bash scripts/gen-certs.sh

test:
	cd remote-executor && go test ./... -race -v
	cd gui             && go test ./... -race -v

build:
	cd remote-executor && go build -o dist/remote-executor ./cmd/executor/
	cd gui             && go build -o dist/gui             ./cmd/gui/

docker:
	docker build -t remote-executor -f remote-executor/deployments/Dockerfile remote-executor/
	docker build -t remote-gui      -f gui/deployments/Dockerfile              gui/

compose-up:
	docker-compose up -d

lint:
	cd remote-executor && golangci-lint run ./...
	cd gui             && golangci-lint run ./...
```

---

## 八、开发时间线

| Phase | 任务 | 预计工期 | 累计 |
|-------|------|---------|------|
| 0 | 项目骨架 + 接口定义 + 证书脚本 | 0.5 天 | 0.5 天 |
| 1 | executor：脚本加载 + 参数校验 | 1 天 | 1.5 天 |
| 2 | executor：脚本执行 + 操作记录 | 1 天 | 2.5 天 |
| 3 | executor：mTLS HTTP 服务 | 1.5 天 | 4 天 |
| 4 | gui：HTTP 客户端 + 配置 + Fyne UI | 1.5 天 | 5.5 天 |
| 5 | 端到端集成测试 | 1 天 | 6.5 天 |
| 6 | GitHub Actions CI/CD | 0.5 天 | 7 天 |
| 7 | 部署文档 + systemd | 0.5 天 | 7.5 天 |
| **总计** | **MVP 完整交付** | **约 7.5 天** | — |

---

## 九、验收标准（MVP）

### 功能验收

- [ ] executor 启动后自动加载 `scripts/` 目录下所有脚本
- [ ] 未知参数、不合规格式被 executor 拒绝（422）
- [ ] GUI 展示配置中定义的所有操作别名
- [ ] GUI 调用 executor 后结果实时展示在界面
- [ ] 每次操作在 executor 本地 `records/` 留下完整记录

### 安全验收

- [ ] 不携带客户端证书的请求无法建立连接
- [ ] 参数校验 pattern 覆盖注入字符（`;`, `$`, `` ` ``, `|`, `&` 等不满足规则时被拒绝）
- [ ] 脚本参数通过环境变量注入，无命令行拼接
- [ ] TLS 版本 ≥ 1.3

### CI/CD 验收

- [ ] PR 触发自动测试，测试失败则阻断合并
- [ ] 推送 main 后自动发布多平台二进制和 Docker 镜像
- [ ] Docker 镜像可直接 `docker run` 启动服务

### 文档验收

- [ ] README 包含 5 分钟快速启动指南
- [ ] `docs/api-spec.md` 覆盖所有 API 端点
- [ ] `docs/deployment.md` 涵盖 systemd 和 Docker 两种部署方式

---

## 十、会话恢复指南

每次新会话开始时执行以下步骤快速定位进度：

```bash
# 1. 查看最近 commit 和 tag
git log --oneline -10
git tag --sort=-creatordate | head -5

# 2. 确认当前测试状态
cd remote-executor && go test ./... 2>&1 | tail -5
cd ../gui          && go test ./... 2>&1 | tail -5

# 3. 对照上方阶段划分，从最新 tag 对应的 Phase 继续
```

> **规则**：每个 Phase 的检查点命令全部通过，且完成 `git commit + git tag` 后，才可继续下一 Phase。如发现测试失败，先修复再推进，不允许跳过。
