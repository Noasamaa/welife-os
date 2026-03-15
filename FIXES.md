# WeLife OS 审计修复追踪

## 立即修复（CRITICAL）— 全部完成

| # | 问题 | 文件 | 状态 |
|---|------|------|------|
| 1 | LLM HTTP 超时仅 5s | `engine/internal/server/server.go` | ✅ 5s→120s |
| 2 | `react_test.go` callCount 竞态 | `engine/internal/report/react_test.go` | ✅ atomic.Int32 |
| 3 | Scheduler `Stop()` 双关闭 panic | `engine/internal/reminder/scheduler.go` | ✅ sync.Once |

## 短期修复（HIGH）— 全部完成

| # | 问题 | 文件 | 状态 |
|---|------|------|------|
| 4 | Task manager 无 TTL 清理 | `engine/internal/task/manager.go` | ✅ 1h TTL + 10m 清理 |
| 5 | `io.ReadAll` 大文件无上限 | `engine/internal/importer/service.go` | ✅ 512MB 上限 |
| 6 | `w.Write(pdf)` 返回值未检查 | `engine/internal/server/handler_report.go` | ✅ log 错误 |
| 7 | Reminder ID 碰撞风险 | `engine/internal/reminder/scheduler.go` | ✅ UUID v4 |
| 8 | Importer 静默忽略 DB 错误 | `engine/internal/importer/service.go` | ✅ 4处改为 log |
| 9 | ReminderBell 组件未使用 | `tauri-app/src/components/AppShell.vue` | ✅ 集成到 header |

## 补充修复（同类扫描）— 全部完成

| # | 问题 | 文件 | 状态 |
|---|------|------|------|
| 10 | Import ID 无 seq | `engine/internal/importer/service.go` | ✅ atomic seq |
| 11 | Report failReport 静默忽略 | `engine/internal/report/generator.go` | ✅ log |
| 12 | Simulation failSession 静默忽略 | `engine/internal/simulation/engine.go` | ✅ log |
| 13 | Forum failSession 静默忽略 | `engine/internal/forum/engine.go` | ✅ log |

## 功能级修复 — 全部完成

| # | 问题 | 文件 | 状态 |
|---|------|------|------|
| 14 | `chat-ir-spec.md` 仅 21 行草案 | `docs/chat-ir-spec.md` | ✅ 完整重写，对齐 model.go |
| 15 | `agent-protocol.md` 字段名不匹配 | `docs/agent-protocol.md` | ✅ 完整重写，对齐 agent.go |
| 16 | 暗色模式颜色硬编码 | Reports/Simulation/Import.vue | ✅ 全部替换为 CSS 变量 |

## 验证

- [x] `go vet ./...` 通过
- [x] `vue-tsc --noEmit` 通过

---

## 剩余 6 项（功能开发级，需独立规划）

1. ~~Cloud LLM API（`engine/internal/llm/cloud.go` 仅占位符）~~ ✅ OpenAI 兼容实现
2. ~~前端测试覆盖（零 .test.ts 文件）~~ ✅ vitest + 79 个组件/composable 测试
3. ~~API handler 测试（7 个 handler 34 个函数无测试）~~ ✅ 48 个 handler 测试，覆盖率 68.3%
4. ~~Graph 持久化（纯内存，重启丢失）~~ ✅ Load() 启动时从 SQLite 重建
5. ~~sqlite-vec 向量搜索集成（仅 NoopVectorStore）~~ ✅ CGO 集成 + SqliteVecStore + Embed API
6. Tauri 原生功能（系统托盘 / 通知 / 自动更新）
