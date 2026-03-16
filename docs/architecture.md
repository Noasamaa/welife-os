# WeLife OS 架构文档

## 总览

WeLife OS 采用 monorepo 结构，分为三大模块：

| 模块 | 技术 | 职责 |
|---|---|---|
| `engine/` | Go 1.26 + chi/v5 | 后端引擎：导入、图谱、Agent、辩论、报告、模拟、提醒 |
| `tauri-app/` | Tauri v2 + Vue 3 + TypeScript | 桌面壳 + 前端 UI |
| `docs/` | Markdown | 架构、协议文档 |

## 架构分层

```
Tauri v2 桌面壳 (Rust)
  └─ Vue 3 前端 (TypeScript + Tailwind + ECharts)
       └─ HTTP API (127.0.0.1:18080)
            └─ Go 后端引擎
                 ├─ 数据导入层 (8 平台解析器 → Chat IR)
                 ├─ 知识图谱 (gonum/graph + LLM 实体抽取)
                 ├─ Agent 系统 (5 Agent + ForumEngine 辩论)
                 ├─ 报告系统 (ReACT 循环 + PDF/HTML 导出)
                 ├─ 模拟系统 (数字分身 + 多步演化)
                 ├─ 提醒系统 (规则评估 + 定时调度)
                 ├─ 异步任务管理 (goroutine worker pool)
                 └─ 加密存储 (SQLCipher v4)
                      └─ Ollama 本地 LLM
```

## 后端包结构

| 包 | 路径 | 职责 |
|---|---|---|
| `server` | `internal/server/` | HTTP 路由、处理器、服务器生命周期 |
| `chatir` | `internal/chatir/` | 统一聊天中间格式 (Chat IR) |
| `parser` | `internal/parser/` | 8 个平台解析器 (WeChat/Telegram/WhatsApp/QQ/Discord/Lark/iMessage/CSV) |
| `importer` | `internal/importer/` | 导入编排 (解析 → 存储 → 图谱) |
| `graph` | `internal/graph/` | 知识图谱引擎 (实体抽取 + gonum 图 + 克隆) |
| `agent` | `internal/agent/` | 5 个 AI Agent (Emotion/Opportunity/Risk/Coach/Simulator) |
| `forum` | `internal/forum/` | ForumEngine 三轮辩论引擎 + Moderator |
| `report` | `internal/report/` | ReACT 报告生成 + 检索工具 + PDF/HTML 渲染 |
| `simulation` | `internal/simulation/` | 平行人生模拟 (ProfileBuilder + 多步 Engine) |
| `reminder` | `internal/reminder/` | 提醒系统 (Checker + Scheduler + Service) |
| `storage` | `internal/storage/` | SQLCipher 数据层 (Schema v8, 19 张表) |
| `llm` | `internal/llm/` | Ollama 客户端 + JSON 提取工具 |
| `task` | `internal/task/` | 异步任务管理器 (goroutine pool) |

## 数据库 Schema (v8)

| 表 | Phase | 用途 |
|---|---|---|
| `schema_state` | 0 | 版本跟踪 |
| `conversations` | 1 | 导入的对话 |
| `messages` | 1 | 聊天消息 |
| `participants` | 1 | 对话参与者 |
| `entities` | 1 | 知识图谱实体 |
| `relationships` | 1 | 知识图谱关系 |
| `import_jobs` | 1 | 导入任务记录 |
| `forum_sessions` | 2 | 辩论会话 |
| `forum_messages` | 2 | 辩论消息 |
| `reports` | 3 | 生成的报告 |
| `action_items` | 4 | 行动项 |
| `reminder_rules` | 4 | 提醒规则 |
| `reminders` | 4 | 触发的提醒 |
| `person_profiles` | 4 | 数字分身画像 |
| `simulation_sessions` | 4 | 模拟会话 |
| `simulation_steps` | 4 | 模拟步骤 |
| `system_settings` | 5 | 系统设置键值对 |
| `vec_messages` | 6 | 消息向量索引 |
| `vec_messages_data` | 6 | 消息向量数据 |

## API 端点

### 系统
- `GET /health` — 健康检查
- `GET /api/v1/system/status` — 系统状态

### 导入
- `POST /api/v1/import` — 上传聊天文件
- `GET /api/v1/import/jobs` — 导入任务列表

### 对话
- `GET /api/v1/conversations` — 对话列表
- `GET /api/v1/conversations/{id}/messages` — 消息列表

### 图谱
- `POST /api/v1/graph/build` — 构建知识图谱
- `GET /api/v1/graph/overview` — 图谱概览

### 辩论
- `POST /api/v1/forum/debate` — 触发辩论
- `GET /api/v1/forum/sessions` — 辩论列表
- `GET /api/v1/forum/sessions/{id}` — 辩论详情

### 报告
- `POST /api/v1/reports/generate` — 生成报告
- `GET /api/v1/reports` — 报告列表
- `GET /api/v1/reports/{id}` — 报告详情
- `GET /api/v1/reports/{id}/html` — 导出 HTML
- `GET /api/v1/reports/{id}/pdf` — 导出 PDF
- `DELETE /api/v1/reports/{id}` — 删除报告

### 教练
- `POST /api/v1/coach/generate-plan` — 生成行动计划
- `GET /api/v1/action-items` — 行动项列表
- `PATCH /api/v1/action-items/{id}` — 更新行动项

### 提醒
- `GET /api/v1/reminders/pending` — 待处理提醒
- `PATCH /api/v1/reminders/{id}/read` — 标记已读
- `GET /api/v1/reminder-rules` — 提醒规则列表
- `POST /api/v1/reminder-rules` — 创建规则

### 模拟
- `POST /api/v1/simulation/profiles/build` — 构建人物画像
- `GET /api/v1/simulation/profiles` — 画像列表
- `POST /api/v1/simulation/run` — 运行模拟
- `GET /api/v1/simulation/sessions/{id}` — 模拟详情

## 前端页面

| 页面 | 路由 | 功能 |
|---|---|---|
| Dashboard | `/` | 系统状态 + 行动摘要 + 提醒 |
| Import | `/import` | 文件上传 + 导入历史 |
| Reports | `/reports` | 报告生成 + 浏览 + 导出 |
| Forum | `/forum` | 辩论触发 + 辩论记录浏览 |
| Coach | `/coach` | 行动看板 + 状态过滤 |
| Timeline | `/timeline` | 聚合时间线 |
| Simulation | `/simulation` | 人物画像 + 分叉模拟 |
| Settings | `/settings` | 配置 + 主题 + 关于 |
