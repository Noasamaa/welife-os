package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
)

// EntityType classifies knowledge graph entities.
type EntityType string

const (
	EntityPerson  EntityType = "person"
	EntityEvent   EntityType = "event"
	EntityTopic   EntityType = "topic"
	EntityPromise EntityType = "promise"
	EntityPlace   EntityType = "place"
)

// ExtractedEntity is an entity extracted by LLM from chat messages.
type ExtractedEntity struct {
	Type       EntityType        `json:"type"`
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ExtractedRelationship is a relationship extracted by LLM.
type ExtractedRelationship struct {
	SourceName string `json:"source_name"`
	TargetName string `json:"target_name"`
	Type       string `json:"type"`
}

// ExtractionResult holds entities and relationships from a single extraction.
type ExtractionResult struct {
	Entities      []ExtractedEntity      `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
}

// Extractor uses LLM to extract entities and relationships from messages.
type Extractor struct {
	llm *llm.Client
}

// NewExtractor creates a new entity extractor.
func NewExtractor(llmClient *llm.Client) *Extractor {
	return &Extractor{llm: llmClient}
}

const extractionPrompt = `你是一个信息提取专家。请从以下聊天记录中提取实体和关系。

实体类型：person（人物）、event（事件）、topic（话题）、promise（承诺）、place（地点）

请以 JSON 格式输出，格式如下：
{"entities":[{"type":"person","name":"张三"},{"type":"topic","name":"AI项目"}],"relationships":[{"source_name":"张三","target_name":"AI项目","type":"参与"}]}

注意：
- 只提取明确提到的实体，不要推测
- 人名要用对话中的原始称呼
- 关系类型用简短的中文描述

聊天记录：
%s

请只输出 JSON，不要输出其他内容。`

// Extract sends messages to LLM for entity/relationship extraction.
func (e *Extractor) Extract(ctx context.Context, messages []MessageSnippet) (ExtractionResult, error) {
	if len(messages) == 0 {
		return ExtractionResult{}, nil
	}

	var sb strings.Builder
	for _, m := range messages {
		fmt.Fprintf(&sb, "[%s] %s: %s\n", m.Timestamp, m.SenderName, m.Content)
	}

	prompt := fmt.Sprintf(extractionPrompt, sb.String())

	response, err := e.llm.Generate(ctx, prompt)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("LLM extraction: %w", err)
	}

	// Parse JSON from response (may contain markdown code blocks)
	jsonStr := extractJSON(response)

	var result ExtractionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return ExtractionResult{}, fmt.Errorf("parsing extraction result: %w", err)
	}

	return result, nil
}

// MessageSnippet is a simplified message for LLM context.
type MessageSnippet struct {
	Timestamp  string
	SenderName string
	Content    string
}

// extractJSON tries to find JSON content within a response that may include markdown.
func extractJSON(s string) string {
	// Try to find JSON between code blocks
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end >= 0 {
			return strings.TrimSpace(s[:end])
		}
	}
	if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end >= 0 {
			return strings.TrimSpace(s[:end])
		}
	}
	// Try to find raw JSON
	if idx := strings.Index(s, "{"); idx >= 0 {
		if end := strings.LastIndex(s, "}"); end > idx {
			return s[idx : end+1]
		}
	}
	return s
}
