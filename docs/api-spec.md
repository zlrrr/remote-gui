# remote-executor API 规范

## Base URL

```
https://<executor-host>:<port>/api/v1
```

所有请求均需携带客户端证书（mTLS）。TLS 版本要求 1.3。

---

## 接口列表

### GET /api/v1/scripts

返回所有已加载的脚本列表及其参数规范。

**响应示例**（200 OK）：

```json
{
  "scripts": [
    {
      "name": "query-rocketmq-msg",
      "description": "查询 RocketMQ 指定 Topic 下的消息体",
      "timeout_seconds": 60,
      "params": [
        {
          "name": "topic",
          "description": "消息所在的 Topic 名称",
          "type": "string",
          "required": true,
          "rules": {
            "pattern": "^[a-zA-Z0-9_\\-]{1,64}$",
            "min_length": 1,
            "max_length": 64
          }
        },
        {
          "name": "message_id",
          "description": "消息的 UniqueKey（Message ID）",
          "type": "string",
          "required": true,
          "rules": {
            "pattern": "^[A-F0-9]{32,40}$",
            "min_length": 32,
            "max_length": 40
          }
        }
      ]
    }
  ]
}
```

---

### POST /api/v1/execute

触发指定脚本执行。

**请求头**：

```
Content-Type: application/json
```

**请求体**：

```json
{
  "script": "query-rocketmq-msg",
  "params": {
    "topic": "test-topic",
    "message_id": "01DE76E4BEE026003309B83633000000D7"
  }
}
```

**响应体**（200 OK，执行成功）：

```json
{
  "record_id":   "rec-20250301-143022-abc123",
  "script":      "query-rocketmq-msg",
  "status":      "success",
  "exit_code":   0,
  "stdout":      "...",
  "stderr":      "",
  "duration_ms": 1240,
  "executed_at": "2025-03-01T14:30:22Z"
}
```

**响应体**（200 OK，脚本执行失败，exit_code != 0）：

```json
{
  "record_id":   "rec-20250301-143022-abc124",
  "script":      "query-rocketmq-msg",
  "status":      "failed",
  "exit_code":   1,
  "stdout":      "",
  "stderr":      "error message...",
  "duration_ms": 320,
  "executed_at": "2025-03-01T14:30:22Z"
}
```

**错误响应**（422 Unprocessable Entity，参数校验失败）：

```json
{
  "error": "validation_failed",
  "details": [
    { "param": "message_id", "reason": "格式不匹配，期望 ^[A-F0-9]{32,40}$" }
  ]
}
```

**错误响应**（404 Not Found，脚本不存在）：

```json
{
  "error": "script_not_found",
  "details": "script 'unknown-script' is not registered"
}
```

**错误响应**（408 Request Timeout，脚本超时）：

```json
{
  "error": "execution_timeout",
  "details": "script exceeded timeout of 60 seconds"
}
```

**错误响应**（400 Bad Request，请求体格式错误）：

```json
{
  "error": "invalid_request",
  "details": "failed to decode request body"
}
```

---

### GET /api/v1/records

查询本地操作记录列表（支持分页）。

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page` | int | 1 | 页码（从 1 开始） |
| `page_size` | int | 20 | 每页记录数（最大 100） |

**响应示例**（200 OK）：

```json
{
  "total": 42,
  "page": 1,
  "page_size": 20,
  "records": [
    {
      "record_id":   "rec-20250301-143022-abc123",
      "script":      "query-rocketmq-msg",
      "status":      "success",
      "exit_code":   0,
      "duration_ms": 1240,
      "executed_at": "2025-03-01T14:30:22Z"
    }
  ]
}
```

---

### GET /api/v1/records/{record_id}

查询单条操作记录详情。

**路径参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `record_id` | string | 记录 ID，格式：`rec-YYYYMMDD-HHMMSS-{6位随机串}` |

**响应示例**（200 OK）：

```json
{
  "record_id":   "rec-20250301-143022-abc123",
  "script":      "query-rocketmq-msg",
  "params":      { "topic": "test-topic", "message_id": "01DE76E4BEE026003309B83633000000D7" },
  "status":      "success",
  "exit_code":   0,
  "stdout":      "完整 stdout 内容（最多 4KB）",
  "stderr":      "",
  "duration_ms": 1240,
  "executed_at": "2025-03-01T14:30:22Z"
}
```

**错误响应**（404 Not Found）：

```json
{
  "error": "record_not_found",
  "details": "record 'rec-xxx' not found"
}
```

---

## 错误码汇总

| HTTP 状态码 | error 字段 | 说明 |
|-------------|-----------|------|
| 400 | `invalid_request` | 请求体格式错误 |
| 404 | `script_not_found` | 脚本未注册 |
| 404 | `record_not_found` | 记录不存在 |
| 408 | `execution_timeout` | 脚本执行超时 |
| 422 | `validation_failed` | 参数校验失败 |
| 500 | `internal_error` | 服务器内部错误 |
