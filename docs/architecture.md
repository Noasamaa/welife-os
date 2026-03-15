# WeLife OS 架构草案

## 总览

仓库采用 monorepo 结构：

- `engine/`：Go 后端引擎，负责导入、Chat IR、存储、LLM、任务管理和后续 Agent 扩展
- `tauri-app/`：Tauri v2 + Vue 3 桌面壳
- `docs/`：协议与架构文档

## Phase 0 目标

1. 固化目录边界
2. 固化 Chat IR 契约
3. 打通 Tauri -> Go sidecar -> SQLite/SQLCipher -> Ollama 探活 的最小闭环
4. 为 GraphRAG / Agent / Report / Simulation 预留清晰扩展点

## 当前骨架

- Go HTTP：`net/http + chi/v5`
- 默认地址：`127.0.0.1:18080`
- 状态接口：`GET /health`、`GET /api/v1/system/status`
- 本地存储：SQLCipher（Phase 0 先保证加密 SQLite 可初始化）
- LLM：Ollama 官方 Go client 探活
- 任务管理：最小 goroutine worker pool

## 目录重点

- `cmd/welife/main.go`：入口
- `internal/server/`：HTTP 服务与状态接口
- `internal/chatir/`：统一聊天中间格式
- `internal/storage/`：SQLCipher 与 sqlite-vec 占位
- `internal/llm/`：Ollama 封装
- `internal/task/`：异步任务骨架

## 下一步

- 实现真实导入链路
- 接入 sqlite-vec
- 构建本地知识图谱
- 打通 Tauri 与 Go sidecar 联调
