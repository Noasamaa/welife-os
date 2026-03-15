# Chat IR 规范

> 中间表示层（Intermediate Representation），将不同平台的聊天数据统一映射为稳定格式，
> 使后续 Agent / 图谱 / 报告引擎无需面向平台细节编码。

## 数据结构

### ChatIR（顶层）

| 字段 | 类型 | 说明 |
|------|------|------|
| `platform` | `string` | 来源平台标识，如 `wechat`、`telegram` |
| `conversation_id` | `string` | 对话唯一 ID |
| `conversation_type` | `ConversationType` | `"private"` / `"group"` / `"channel"` |
| `participants` | `[]Participant` | 对话参与者列表 |
| `messages` | `[]Message` | 消息列表，按时间排序 |
| `metadata` | `Metadata` | 导出元信息 |

### ConversationType（枚举）

- `"private"` — 一对一私聊
- `"group"` — 群聊
- `"channel"` — 频道 / 公开群组

### Participant

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 参与者 ID |
| `name` | `string` | 显示名称 |
| `is_self` | `bool` | 是否为用户本人 |

### Message

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 消息唯一 ID |
| `timestamp` | `time.Time` | 发送时间（RFC 3339） |
| `sender_id` | `string` | 发送者 ID，关联 Participant.id |
| `content` | `string` | 消息文本内容 |
| `type` | `MessageType` | 消息类型 |
| `reply_to` | `string` | 引用消息 ID（可选） |
| `attachments` | `[]Attachment` | 附件列表（可选） |

### MessageType（枚举）

- `"text"` — 文本消息
- `"image"` — 图片
- `"file"` — 文件
- `"audio"` — 语音
- `"video"` — 视频
- `"system"` — 系统消息（入群、退群等）

### Attachment

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | `string` | 附件类型 |
| `name` | `string` | 文件名（可选） |
| `path` | `string` | 本地路径（可选） |
| `mime_type` | `string` | MIME 类型（可选） |

### Metadata

| 字段 | 类型 | 说明 |
|------|------|------|
| `exported_at` | `time.Time` | 导出时间 |
| `message_count` | `int` | 消息总数 |
| `date_range` | `[2]string` | 起止日期（可选） |

## 设计原则

1. **平台无关** — 保留最稳定字段，避免平台特化污染核心模型
2. **时间标准化** — 所有时间使用 RFC 3339 格式
3. **附件统一** — 通过 `attachments` 数组统一表达各类媒体
4. **可选字段** — `reply_to`、`attachments`、`date_range` 为可选，缺失时为空

## 源码位置

`engine/internal/chatir/model.go`
