package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

const emotionAgentName = "emotion_analyst"

// EmotionAgent analyzes emotional patterns in conversations.
// It groups messages by contact, identifies emotion shifts, and calculates
// relationship temperatures.
type EmotionAgent struct {
	llm *llm.Client
}

// NewEmotionAgent creates a new emotion analyst agent.
func NewEmotionAgent(llmClient *llm.Client) *EmotionAgent {
	return &EmotionAgent{llm: llmClient}
}

func (a *EmotionAgent) Name() string { return emotionAgentName }

const emotionAnalysisPrompt = `你是一位人际关系情感分析专家。请分析以下对话中的情感模式。

分析要求：
1. 识别每条消息的情感倾向（积极/消极/中性）和强度（0.0-1.0）
2. 发现情绪拐点（突然的情感变化）
3. 评估对话双方的关系温度（0-100，0=冰冷疏远，100=亲密无间）

请以 JSON 格式输出：
{
  "emotion_timeline": [
    {"message_id": "...", "emotion": "积极|消极|中性", "intensity": 0.8, "note": "简短说明"}
  ],
  "emotion_shifts": [
    {"from_message_id": "...", "to_message_id": "...", "from_emotion": "积极", "to_emotion": "消极", "reason": "原因分析"}
  ],
  "relationship_temperature": 75,
  "summary": "总体情感分析摘要"
}

对话记录（联系人: %s）：
%s

请只输出 JSON，不要输出其他内容。`

// emotionLLMResponse is the expected JSON structure from the LLM.
type emotionLLMResponse struct {
	EmotionTimeline []emotionTimelineEntry `json:"emotion_timeline"`
	EmotionShifts   []emotionShift         `json:"emotion_shifts"`
	RelTemp         float64                `json:"relationship_temperature"`
	Summary         string                 `json:"summary"`
}

type emotionTimelineEntry struct {
	MessageID string  `json:"message_id"`
	Emotion   string  `json:"emotion"`
	Intensity float64 `json:"intensity"`
	Note      string  `json:"note"`
}

type emotionShift struct {
	FromMessageID string `json:"from_message_id"`
	ToMessageID   string `json:"to_message_id"`
	FromEmotion   string `json:"from_emotion"`
	ToEmotion     string `json:"to_emotion"`
	Reason        string `json:"reason"`
}

func (a *EmotionAgent) Analyze(ctx context.Context, input AnalysisInput) (AnalysisOutput, error) {
	if len(input.Messages) == 0 {
		return AnalysisOutput{AgentName: emotionAgentName, Summary: "没有消息可供分析"}, nil
	}

	grouped := groupMessagesBySender(input.Messages)

	var allFindings []Finding
	timelines := make(map[string][]emotionTimelineEntry)
	temperatures := make(map[string]float64)
	var overallSummaries []string

	for contact, msgs := range grouped {
		result, err := a.analyzeContactMessages(ctx, contact, msgs)
		if err != nil {
			return AnalysisOutput{}, fmt.Errorf("analyzing contact %q: %w", contact, err)
		}

		timelines[contact] = result.EmotionTimeline
		temperatures[contact] = result.RelTemp
		overallSummaries = append(overallSummaries, fmt.Sprintf("%s: %s", contact, result.Summary))

		for _, shift := range result.EmotionShifts {
			allFindings = append(allFindings, Finding{
				Type:       "emotion_shift",
				Title:      fmt.Sprintf("%s 的情绪变化: %s → %s", contact, shift.FromEmotion, shift.ToEmotion),
				Content:    shift.Reason,
				Evidence:   []string{shift.FromMessageID, shift.ToMessageID},
				Confidence: 0.7,
			})
		}
	}

	return AnalysisOutput{
		AgentName: emotionAgentName,
		Summary:   strings.Join(overallSummaries, "; "),
		Details:   allFindings,
		Data: map[string]any{
			"emotion_timeline":          timelines,
			"relationship_temperatures": temperatures,
		},
	}, nil
}

func (a *EmotionAgent) analyzeContactMessages(ctx context.Context, contact string, msgs []storage.StoredMessage) (emotionLLMResponse, error) {
	var sb strings.Builder
	for _, m := range msgs {
		fmt.Fprintf(&sb, "[%s] %s: %s\n", m.Timestamp, m.SenderName, m.Content)
	}

	prompt := fmt.Sprintf(emotionAnalysisPrompt, contact, sb.String())

	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return emotionLLMResponse{}, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result emotionLLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return emotionLLMResponse{}, fmt.Errorf("parsing emotion response: %w", err)
	}

	return result, nil
}

func (a *EmotionAgent) Debate(ctx context.Context, state DebateState) (ForumMessage, error) {
	return debateHelper(ctx, a.llm, emotionAgentName, "情感分析师", state)
}

// groupMessagesBySender groups messages by sender name.
func groupMessagesBySender(msgs []storage.StoredMessage) map[string][]storage.StoredMessage {
	grouped := make(map[string][]storage.StoredMessage)
	for _, m := range msgs {
		grouped[m.SenderName] = append(grouped[m.SenderName], m)
	}
	return grouped
}
