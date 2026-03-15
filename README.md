# WeLife OS

WeLife OS 是一个面向个人私密数据的人生复盘系统。

当前仓库处于 `Phase 0` 骨架阶段，已经包含：

- `engine/`：Go 后端、Chat IR、SQLCipher 存储、Ollama 探活骨架
- `tauri-app/`：Tauri v2 + Vue 3 桌面壳基础结构
- `docs/`：架构、Chat IR、Agent 协议草案

## 快速开始

### 后端

```bash
cd engine
go test ./...
go run ./cmd/welife
```

### 前端

```bash
cd tauri-app
npm install
npm run tauri:dev
```

## 当前阶段目标

1. 打通本地开发环境
2. 固化统一的聊天记录中间格式 `Chat IR`
3. 为后续解析器、GraphRAG、Agent 和辩论引擎提供稳定骨架

### 默认地址

- Go 后端：`127.0.0.1:18080`
- 健康检查：`GET /health`
- 系统状态：`GET /api/v1/system/status`
