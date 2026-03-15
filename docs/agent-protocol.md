# Agent 协议

> 定义 WeLife OS 中 AI Agent 的统一接口、数据结构和辩论流程。

## Agent 接口

每个 Agent 实现 `Agent` 接口，包含两个核心方法：

```go
type Agent interface {
    Name() string
    Analyze(ctx, input AnalysisInput) (AnalysisOutput, error)
    Debate(ctx, state DebateState) (ForumMessage, error)
}
```

## 已实现的 Agent

| Agent | 名称 | 职责 |
|-------|------|------|
| EmotionCartographer | `emotion_cartographer` | 情感趋势分析 |
| OpportunityMiner | `opportunity_miner` | 机会挖掘 |
| RiskTribunal | `risk_tribunal` | 风险评估（3 个子 goroutine 并发） |
| ExecutionCoach | `execution_coach` | 行动计划生成 |
| FutureSimulator | `future_simulator` | 平行人生模拟 |

## 数据结构

### AnalysisInput（分析输入）

| 字段 | 类型 | 说明 |
|------|------|------|
| `ConversationID` | `string` | 对话 ID |
| `Messages` | `[]StoredMessage` | 对话消息列表 |
| `Entities` | `[]Entity` | 知识图谱实体 |
| `Relationships` | `[]Relationship` | 知识图谱关系 |

### AnalysisOutput（分析输出）

| 字段 | JSON | 说明 |
|------|------|------|
| `AgentName` | `agent_name` | Agent 标识 |
| `Summary` | `summary` | 分析摘要 |
| `Details` | `details` | 发现列表 `[]Finding` |
| `Data` | `data` | 扩展数据（可选） |

### Finding（单条发现）

| 字段 | JSON | 说明 |
|------|------|------|
| `Type` | `type` | 发现类型 |
| `Title` | `title` | 标题 |
| `Content` | `content` | 详细内容 |
| `Evidence` | `evidence` | 支撑证据 `[]string` |
| `Confidence` | `confidence` | 置信度 `0.0-1.0` |

### DebateState（辩论状态）

| 字段 | 说明 |
|------|------|
| `SessionID` | 辩论会话 ID |
| `Round` | 当前轮次 |
| `Topic` | 辩论议题 |
| `History` | 之前的发言 `[]ForumMessage` |
| `MyPrior` | 自己的先验分析（可选） |
| `OtherViews` | 其他 Agent 的分析 |

### ForumMessage（辩论发言）

| 字段 | JSON | 说明 |
|------|------|------|
| `AgentName` | `agent_name` | 发言者 |
| `Round` | `round` | 所属轮次 |
| `Stance` | `stance` | 立场 |
| `Content` | `content` | 论点内容 |
| `Evidence` | `evidence` | 证据 `[]string`（可选） |
| `Confidence` | `confidence` | 置信度 |

## 辩论流程（ForumEngine）

```
Round 1: 独立分析
  所有 Agent 并行执行 Analyze()，输出 AnalysisOutput

Round 2+: 交叉辩论
  Moderator 生成议题 → 所有 Agent 并行执行 Debate()
  每轮可见其他 Agent 的观点和历史发言

Final: 主持人共识
  Moderator.Summarize() 汇总共识与分歧
```

## 源码位置

- Agent 接口：`engine/internal/agent/agent.go`
- 各 Agent 实现：`engine/internal/agent/emotion.go`、`opportunity.go`、`risk.go`、`coach.go`
- ForumEngine：`engine/internal/forum/engine.go`
- Moderator：`engine/internal/forum/moderator.go`
