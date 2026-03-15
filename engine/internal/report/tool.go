package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

// Tool defines the interface for search tools used by the ReACT agent.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]string) (string, error)
}

// GraphSearchTool searches the knowledge graph for entities and relationships.
type GraphSearchTool struct {
	store *storage.Store
}

// NewGraphSearchTool creates a new graph search tool.
func NewGraphSearchTool(store *storage.Store) *GraphSearchTool {
	return &GraphSearchTool{store: store}
}

func (t *GraphSearchTool) Name() string { return "graph_search" }

func (t *GraphSearchTool) Description() string {
	return "搜索知识图谱中的实体和关系。参数：entity_type（实体类型，可选）、entity_name（实体名称，可选）"
}

func (t *GraphSearchTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	entityType := params["entity_type"]
	entityName := params["entity_name"]
	conversationID := params["conversation_id"]

	var entities []storage.Entity
	var err error

	if entityType != "" {
		entities, err = t.store.FindEntitiesByType(ctx, entityType)
	} else {
		entities, err = t.store.ListEntities(ctx)
	}
	if err != nil {
		return "", fmt.Errorf("querying entities: %w", err)
	}

	// Filter by name if specified
	if entityName != "" {
		var filtered []storage.Entity
		for _, e := range entities {
			if strings.Contains(e.Name, entityName) {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}
	if conversationID != "" {
		var filtered []storage.Entity
		for _, e := range entities {
			if e.SourceConversation == conversationID {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}

	// Get relationships for found entities
	var rels []storage.Relationship
	for _, e := range entities {
		r, err := t.store.GetRelationships(ctx, e.ID)
		if err != nil {
			continue
		}
		rels = append(rels, r...)
	}

	result := map[string]any{
		"entities":      entities,
		"relationships": rels,
		"entity_count":  len(entities),
		"rel_count":     len(rels),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshalling graph result: %w", err)
	}
	return string(data), nil
}

// ForumSearchTool searches debate sessions and messages.
type ForumSearchTool struct {
	store *storage.Store
}

// NewForumSearchTool creates a new forum search tool.
func NewForumSearchTool(store *storage.Store) *ForumSearchTool {
	return &ForumSearchTool{store: store}
}

func (t *ForumSearchTool) Name() string { return "forum_search" }

func (t *ForumSearchTool) Description() string {
	return "搜索辩论记录和会话。参数：conversation_id（对话ID，可选）、status（状态过滤，可选）、after（起始时间，可选）、before（结束时间，可选）"
}

func (t *ForumSearchTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	sessions, err := t.store.ListSessions(ctx)
	if err != nil {
		return "", fmt.Errorf("listing sessions: %w", err)
	}

	convID := params["conversation_id"]
	status := params["status"]
	after := params["after"]
	before := params["before"]

	var filtered []storage.ForumSession
	for _, s := range sessions {
		if convID != "" && s.ConversationID != convID {
			continue
		}
		if status != "" && s.Status != status {
			continue
		}
		if after != "" || before != "" {
			createdAt, err := ParseFlexibleTime(s.CreatedAt, false)
			if err != nil {
				continue
			}
			if after != "" {
				start, err := ParseFlexibleTime(after, false)
				if err != nil || createdAt.Before(start) {
					continue
				}
			}
			if before != "" {
				end, err := ParseFlexibleTime(before, true)
				if err != nil || createdAt.After(end) {
					continue
				}
			}
		}
		filtered = append(filtered, s)
	}

	// Get messages for each session
	type sessionDetail struct {
		Session  storage.ForumSession         `json:"session"`
		Messages []storage.ForumMessageRecord `json:"messages"`
	}

	var details []sessionDetail
	for _, s := range filtered {
		msgs, err := t.store.GetForumMessages(ctx, s.ID)
		if err != nil {
			continue
		}
		details = append(details, sessionDetail{Session: s, Messages: msgs})
	}

	data, err := json.Marshal(map[string]any{
		"sessions":      details,
		"session_count": len(details),
	})
	if err != nil {
		return "", fmt.Errorf("marshalling forum result: %w", err)
	}
	return string(data), nil
}

// MessageSearchTool performs keyword-based search on messages.
type MessageSearchTool struct {
	store *storage.Store
}

// NewMessageSearchTool creates a new message search tool.
func NewMessageSearchTool(store *storage.Store) *MessageSearchTool {
	return &MessageSearchTool{store: store}
}

func (t *MessageSearchTool) Name() string { return "message_search" }

func (t *MessageSearchTool) Description() string {
	return "搜索聊天消息。参数：keyword（关键词）、conversation_id（对话ID，可选）、sender（发送者，可选）、after（起始时间，可选）、before（结束时间，可选）"
}

func (t *MessageSearchTool) Execute(ctx context.Context, params map[string]string) (string, error) {
	searchParams := storage.MessageSearchParams{
		Keyword:        params["keyword"],
		ConversationID: params["conversation_id"],
		SenderName:     params["sender"],
		After:          params["after"],
		Before:         params["before"],
		Limit:          50,
	}

	msgs, err := t.store.SearchMessages(ctx, searchParams)
	if err != nil {
		return "", fmt.Errorf("searching messages: %w", err)
	}

	data, err := json.Marshal(map[string]any{
		"messages":      msgs,
		"message_count": len(msgs),
	})
	if err != nil {
		return "", fmt.Errorf("marshalling message result: %w", err)
	}
	return string(data), nil
}
