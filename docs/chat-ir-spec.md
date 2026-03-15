# Chat IR 规范草案

## 目标

把不同平台的聊天数据统一映射到一个稳定中间格式，避免后续 Agent 面向平台细节写死逻辑。

## 核心字段

- `platform`
- `conversation_id`
- `conversation_type`
- `participants`
- `messages`
- `metadata`

## 设计原则

1. 保留最稳定字段，避免平台特化污染核心模型
2. 时间、参与者、消息类型必须标准化
3. 附件通过统一 `attachments` 表达

