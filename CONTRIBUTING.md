# 贡献指南

感谢你对 WeLife OS 的关注！以下是参与贡献的指南。

## 开发环境

### 前置要求

| 工具 | 版本 | 用途 |
|---|---|---|
| Go | 1.26+ | 后端引擎 |
| Node.js | 22+ | 前端构建 |
| Ollama | 最新 | 本地 LLM 推理 |
| libsqlcipher-dev | - | SQLCipher 编译依赖 |

### 搭建步骤

```bash
# 克隆仓库
git clone https://github.com/Noasamaa/welife-os.git
cd welife-os

# 后端
cd engine
go mod download
go test ./...

# 前端
cd ../tauri-app
npm install
npm run typecheck
```

## 项目结构

```
welife-os/
├── engine/                 # Go 后端引擎
│   ├── cmd/welife/         # 入口
│   └── internal/
│       ├── agent/          # 5 个 AI Agent
│       ├── forum/          # 辩论引擎
│       ├── report/         # ReACT 报告 + PDF 导出
│       ├── simulation/     # 平行人生模拟
│       ├── reminder/       # 提醒系统
│       ├── parser/         # 8 个聊天解析器
│       ├── graph/          # 知识图谱
│       ├── storage/        # SQLCipher 数据层
│       ├── llm/            # Ollama 封装
│       ├── task/           # 异步任务管理
│       └── server/         # HTTP 服务
├── tauri-app/              # Tauri + Vue 3 前端
│   └── src/
│       ├── views/          # 页面
│       ├── components/     # 组件
│       ├── composables/    # Vue 组合函数
│       ├── types/          # TypeScript 类型
│       └── services/       # API 调用
└── docs/                   # 文档
```

## 代码规范

### Go 后端

- **必须通过**: `go vet ./...` + `staticcheck ./...`
- **测试**: `go test -race ./...`，目标 80%+ 覆盖率
- **文件大小**: 单文件不超过 400 行
- **错误处理**: 所有错误必须显式处理，不允许静默忽略
- **不可变性**: 优先返回新对象，避免修改入参

### Vue 前端

- **类型检查**: `npm run typecheck` 必须通过
- **CSS**: 使用 CSS 变量（`var(--color-*)`），支持暗色主题
- **组件**: 单文件组件，`<script setup lang="ts">`

## 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>: <description>

<optional body>
```

| 类型 | 说明 |
|---|---|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `refactor` | 重构 |
| `docs` | 文档 |
| `test` | 测试 |
| `chore` | 构建/工具 |
| `perf` | 性能优化 |

## PR 流程

1. **Fork** 本仓库
2. **创建分支**: `git checkout -b feat/your-feature`
3. **开发并测试**:
   - `go test -race ./...`
   - `npm run typecheck`
4. **提交**: 遵循提交规范
5. **推送**: `git push origin feat/your-feature`
6. **创建 PR**: 描述改动内容、测试方案

### PR 要求

- [ ] Go 测试通过 (`go test -race ./...`)
- [ ] 前端类型检查通过 (`npm run typecheck`)
- [ ] 无 staticcheck 警告
- [ ] 新功能有对应测试
- [ ] 提交信息符合规范

## 问题反馈

- 使用 [GitHub Issues](https://github.com/Noasamaa/welife-os/issues) 提交 Bug 或功能请求
- Bug 报告请包含：复现步骤、期望行为、实际行为、环境信息

## 许可证

贡献的代码将遵循 [AGPL-3.0](./LICENSE) 许可证。
