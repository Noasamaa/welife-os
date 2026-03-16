# WeLife Engine

Go 后端引擎负责：

- 聊天记录导入与归一化
- SQLCipher 本地存储
- Ollama 交互与探活
- Agent / GraphRAG / Report / Simulation 的扩展骨架

## 开发命令

```bash
go test ./...
go run ./cmd/welife
```

## 默认配置

- `WELIFE_HOST=127.0.0.1`
- `WELIFE_PORT=18080`
- `WELIFE_DB_PATH=./.data/welife.db`
- `WELIFE_DB_KEY=(自动生成并存入 welife.key，也可通过环境变量覆盖)`
- `WELIFE_OLLAMA_BASE_URL=http://127.0.0.1:11434`
- `WELIFE_OLLAMA_MODEL=qwen3.5:9b`

