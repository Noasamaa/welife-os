# 剩余 6 项功能开发计划

> 每项都是独立的功能开发任务，建议按优先级逐个推进。

---

## 1. Cloud LLM API 集成

**优先级：** 高（解锁无本地 GPU 用户）
**预估工作量：** 中

### 目标
实现 OpenAI 兼容协议的云端 LLM 客户端，支持在无 Ollama 环境下使用。

### 实施步骤
1. 在 `engine/internal/llm/cloud.go` 实现 `CloudClient`，复用 `Client` 同样的 `Generate(ctx, prompt) (string, error)` 签名
2. 支持 OpenAI API 格式（`/v1/chat/completions`），兼容 DeepSeek / 通义千问等国产 API
3. 在 `llm/` 包中抽取 `LLMClient` 接口，让 `Client`（Ollama）和 `CloudClient` 共用
4. 修改 `server.go` 的 `Config`：增加 `LLMProvider` 字段（`"ollama"` / `"openai-compatible"`）
5. 增加 `API_KEY` 环境变量支持，启动时校验
6. Settings 页面增加 Cloud LLM 配置区域

### 关键文件
- 新建：`engine/internal/llm/cloud.go`（重写）
- 修改：`engine/internal/llm/ollama.go`（抽取接口）
- 修改：`engine/internal/server/server.go`（Provider 选择）
- 修改：`tauri-app/src/views/Settings.vue`（配置 UI）

---

## 2. 前端测试覆盖

**优先级：** 高（零覆盖 → 基础覆盖）
**预估工作量：** 大

### 目标
建立 Vitest + Vue Test Utils 测试基础设施，覆盖核心 composables 和关键组件。

### 实施步骤
1. 安装依赖：`vitest`, `@vue/test-utils`, `happy-dom`
2. 配置 `vitest.config.ts`（使用 happy-dom 环境）
3. 在 `package.json` 增加 `"test"` 和 `"test:coverage"` 脚本
4. 优先编写 composable 单元测试（纯逻辑，无 DOM）：
   - `useImport.test.ts`
   - `useReport.test.ts`
   - `useReminder.test.ts`
   - `useGraph.test.ts`
   - `useForum.test.ts`
   - `useSimulation.test.ts`
5. 编写组件快照/交互测试：
   - `DropZone.test.ts`
   - `ReportViewer.test.ts`
   - `ReminderBell.test.ts`
6. 目标：composables 80%+ 覆盖率

### 关键文件
- 新建：`tauri-app/vitest.config.ts`
- 新建：`tauri-app/src/composables/__tests__/*.test.ts`（9 个）
- 新建：`tauri-app/src/components/__tests__/*.test.ts`（3 个）

---

## 3. API Handler 测试

**优先级：** 中（后端已有 31 个测试文件，handler 层缺口最大）
**预估工作量：** 大

### 目标
为 7 个 handler 文件补全 HTTP 级测试，覆盖正常路径 + 错误路径。

### 实施步骤
1. 创建 `server/testutil_test.go`：封装测试 Server 创建（mock Store + mock LLM）
2. 按 handler 逐个编写：
   - `handler_import_test.go` — 上传、任务状态查询、格式检测
   - `handler_report_test.go` — 生成、列表、详情、PDF/HTML 导出、删除
   - `handler_graph_test.go` — 构建、Overview 查询
   - `handler_forum_test.go` — 发起辩论、会话列表、消息查询
   - `handler_reminder_test.go` — CRUD 规则、待处理列表、标记已读
   - `handler_coach_test.go` — 行动计划、CRUD
   - `handler_simulation_test.go` — 模拟运行、会话列表、步骤查询
3. 使用 `httptest.NewRecorder` + `chi.NewRouter` 构建隔离测试
4. 目标：每个 handler 至少 2 个测试（正常 + 错误）

### 关键文件
- 新建：`engine/internal/server/testutil_test.go`
- 新建：7 个 `handler_*_test.go` 文件

---

## 4. Graph 持久化

**优先级：** 中（重启后图谱丢失影响体验）
**预估工作量：** 中

### 目标
将 GraphStore 的实体和关系持久化到 SQLite，启动时自动恢复。

### 实施步骤
1. 在 `storage/migrations.go` 添加 `graph_edges` 表：
   ```sql
   CREATE TABLE IF NOT EXISTS graph_edges (
     source_id TEXT NOT NULL,
     target_id TEXT NOT NULL,
     weight REAL DEFAULT 1.0,
     PRIMARY KEY (source_id, target_id)
   );
   ```
2. 在 `storage/` 新增 `graph_persistence.go`：
   - `SaveGraphEdges(ctx, edges []Edge) error`
   - `LoadGraphEdges(ctx) ([]Edge, error)`
3. 修改 `graph/engine.go` 的 `BuildGraph()`：构建完成后调用 `SaveGraphEdges()`
4. 修改 `graph/engine.go` 的 `NewGraphEngine()`：启动时调用 `LoadGraphEdges()` 恢复
5. `AddEdge` 操作也同步写入 DB（或批量异步）

### 关键文件
- 修改：`engine/internal/storage/migrations.go`
- 新建：`engine/internal/storage/graph_persistence.go`
- 修改：`engine/internal/graph/engine.go`

---

## 5. sqlite-vec 向量搜索集成

**优先级：** 低（当前 NoopVectorStore 不影响核心功能）
**预估工作量：** 大

### 目标
用 sqlite-vec 扩展替换 NoopVectorStore，实现消息级语义搜索。

### 实施步骤
1. 研究 sqlite-vec Go 绑定可用性（CGO 编译要求）
2. 在 `storage/vector.go` 实现 `SqliteVecStore`：
   - `StoreEmbedding(id, vector, metadata)` — INSERT 向量
   - `Search(query, topK)` — 余弦相似度搜索
   - `Ready()` — 检查扩展是否加载
3. 在 `storage/sqlite.go` 的 `Open()` 中尝试加载 sqlite-vec 扩展
4. 添加嵌入生成：调用 Ollama 的 `/api/embed` 端点生成向量
5. 导入流程中自动为每条消息生成嵌入并存储
6. ReACT 工具集中增加 `semantic_search` 工具

### 前置条件
- sqlite-vec 需要 CGO + C 编译器
- Ollama 需支持 embedding model（如 `nomic-embed-text`）

### 关键文件
- 重写：`engine/internal/storage/vector.go`
- 修改：`engine/internal/llm/ollama.go`（增加 Embed 方法）
- 修改：`engine/internal/importer/service.go`（导入时生成嵌入）
- 新建：`engine/internal/report/tool_semantic.go`（语义搜索工具）

---

## 6. Tauri 原生功能

**优先级：** 低（桌面体验增强，非核心功能）
**预估工作量：** 中

### 目标
启用系统托盘、桌面通知和自动更新，提升桌面应用体验。

### 实施步骤

#### 6a. 系统托盘
1. `Cargo.toml` 添加 `tauri-plugin-tray`
2. `tauri.conf.json` 添加 tray 配置和图标
3. `main.rs` 中注册 TrayBuilder，添加菜单项（显示/隐藏、退出）
4. 窗口关闭时最小化到托盘而非退出

#### 6b. 桌面通知
1. `Cargo.toml` 添加 `tauri-plugin-notification`
2. `tauri.conf.json` 添加 notification 权限
3. 前端 `useReminder.ts` 中，新提醒触发时调用 Tauri notification API
4. 添加通知权限请求提示

#### 6c. 自动更新
1. `Cargo.toml` 添加 `tauri-plugin-updater`
2. `tauri.conf.json` 配置 updater endpoint（GitHub Releases）
3. 前端 Settings 页面增加「检查更新」按钮
4. 启动时静默检查更新

### 关键文件
- 修改：`tauri-app/src-tauri/Cargo.toml`
- 修改：`tauri-app/src-tauri/tauri.conf.json`
- 修改：`tauri-app/src-tauri/src/main.rs`
- 修改：`tauri-app/src/composables/useReminder.ts`
- 修改：`tauri-app/src/views/Settings.vue`

---

## 建议执行顺序

```
1. Cloud LLM API        ← 最高用户价值
2. 前端测试覆盖          ← 质量基础
3. Graph 持久化          ← 体验改善
4. API Handler 测试      ← 测试补全
5. sqlite-vec 集成       ← 高级特性
6. Tauri 原生功能        ← 桌面增强
```
