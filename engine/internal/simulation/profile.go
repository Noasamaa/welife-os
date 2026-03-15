package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
	"github.com/welife-os/welife-os/engine/internal/storage"
)

// ProfileBuilder generates person profiles from entity and relationship data.
type ProfileBuilder struct {
	llm   *llm.Client
	store *storage.Store
}

// NewProfileBuilder creates a ProfileBuilder.
func NewProfileBuilder(llmClient *llm.Client, store *storage.Store) *ProfileBuilder {
	return &ProfileBuilder{llm: llmClient, store: store}
}

const profilePrompt = `你是一位人物画像分析专家。根据以下信息，为这个人生成详细的性格画像。

人物名称: %s

与此人相关的关系:
%s

此人的部分对话记录:
%s

请以 JSON 格式输出：
{
  "personality": "性格特点描述（100-200字）",
  "relationship_to_self": "与用户的关系特征描述",
  "behavioral_patterns": "行为模式和习惯描述"
}

请只输出 JSON。`

type profileLLMResponse struct {
	Personality        string `json:"personality"`
	RelationshipToSelf string `json:"relationship_to_self"`
	BehavioralPatterns string `json:"behavioral_patterns"`
}

// BuildProfile generates a profile for a single entity.
func (b *ProfileBuilder) BuildProfile(ctx context.Context, entityID string) (storage.PersonProfile, error) {
	entities, err := b.store.ListEntities(ctx)
	if err != nil {
		return storage.PersonProfile{}, fmt.Errorf("listing entities: %w", err)
	}

	var entity storage.Entity
	found := false
	for _, e := range entities {
		if e.ID == entityID {
			entity = e
			found = true
			break
		}
	}
	if !found {
		return storage.PersonProfile{}, fmt.Errorf("entity %q not found", entityID)
	}

	rels, err := b.store.GetRelationships(ctx, entityID)
	if err != nil {
		return storage.PersonProfile{}, fmt.Errorf("getting relationships: %w", err)
	}

	// Build relationship descriptions.
	var relDesc strings.Builder
	entityNames := buildEntityNameMap(entities)
	for _, r := range rels {
		sourceName := entityNames[r.SourceEntityID]
		targetName := entityNames[r.TargetEntityID]
		fmt.Fprintf(&relDesc, "- %s → %s (类型: %s, 权重: %.1f)\n", sourceName, targetName, r.Type, r.Weight)
	}

	// Get sample messages from the entity's source conversation.
	var msgText strings.Builder
	if entity.SourceConversation != "" {
		msgs, err := b.store.SearchMessages(ctx, storage.MessageSearchParams{
			ConversationID: entity.SourceConversation,
			SenderName:     entity.Name,
			Limit:          20,
		})
		if err == nil {
			for _, m := range msgs {
				fmt.Fprintf(&msgText, "[%s] %s\n", m.Timestamp, m.Content)
			}
		}
	}

	prompt := fmt.Sprintf(profilePrompt, entity.Name, relDesc.String(), msgText.String())

	response, err := b.llm.Generate(ctx, prompt)
	if err != nil {
		return storage.PersonProfile{}, fmt.Errorf("LLM generate: %w", err)
	}

	jsonStr := llm.ExtractJSON(response)
	var result profileLLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return storage.PersonProfile{}, fmt.Errorf("parsing profile response: %w", err)
	}

	profile := storage.PersonProfile{
		ID:                    "pp_" + entityID,
		EntityID:              entityID,
		Name:                  entity.Name,
		Personality:           result.Personality,
		RelationshipToSelf:    result.RelationshipToSelf,
		BehavioralPatterns:    result.BehavioralPatterns,
		SourceConversationIDs: entity.SourceConversation,
	}

	if err := b.store.SavePersonProfile(ctx, profile); err != nil {
		return storage.PersonProfile{}, fmt.Errorf("saving profile: %w", err)
	}

	return profile, nil
}

// BuildAllProfiles generates profiles for all "person" type entities.
func (b *ProfileBuilder) BuildAllProfiles(ctx context.Context) ([]storage.PersonProfile, error) {
	entities, err := b.store.FindEntitiesByType(ctx, "person")
	if err != nil {
		return nil, fmt.Errorf("finding person entities: %w", err)
	}

	var profiles []storage.PersonProfile
	for _, e := range entities {
		if ctx.Err() != nil {
			return profiles, ctx.Err()
		}

		profile, err := b.BuildProfile(ctx, e.ID)
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func buildEntityNameMap(entities []storage.Entity) map[string]string {
	m := make(map[string]string, len(entities))
	for _, e := range entities {
		m[e.ID] = e.Name
	}
	return m
}
