# Agent 协议草案

## 输入

- `conversation_id`
- `objective`
- `context`

## 输出

- `agent`
- `summary`
- `findings`
- `evidence`
- `confidence`

## Forum 输出

`DebateTurn` 负责保存单次发言，`DebateSummary` 负责保存主持人汇总后的共识与分歧。
