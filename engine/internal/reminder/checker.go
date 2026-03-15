package reminder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

// Checker evaluates reminder rules to determine whether they should fire.
type Checker struct {
	store *storage.Store
}

// NewChecker creates a new rule checker.
func NewChecker(store *storage.Store) *Checker {
	return &Checker{store: store}
}

// Evaluate checks whether a rule should fire at the given time.
// Returns whether to fire, the rendered message, and any error.
func (c *Checker) Evaluate(ctx context.Context, rule storage.ReminderRule, now time.Time) (bool, string, error) {
	switch rule.RuleType {
	case "contact_gap":
		return c.evaluateContactGap(ctx, rule, now)
	case "deadline":
		return c.evaluateDeadline(ctx, rule, now)
	case "periodic":
		return c.evaluatePeriodic(rule, now)
	default:
		return false, "", fmt.Errorf("unknown rule type %q", rule.RuleType)
	}
}

func (c *Checker) evaluateContactGap(ctx context.Context, rule storage.ReminderRule, now time.Time) (bool, string, error) {
	if rule.EntityID == "" {
		return false, "", fmt.Errorf("contact_gap rule %q has no entity_id", rule.ID)
	}

	lastTS, err := c.store.LastMessageTimeForEntity(ctx, rule.EntityID)
	if err != nil {
		return false, "", fmt.Errorf("querying last message for entity %q: %w", rule.EntityID, err)
	}

	if lastTS == "" {
		msg := renderTemplate(rule.MessageTemplate, map[string]string{
			"days": "unknown",
		})
		return true, msg, nil
	}

	lastTime, err := parseTimestamp(lastTS)
	if err != nil {
		return false, "", fmt.Errorf("parsing last message timestamp: %w", err)
	}

	gap := now.Sub(lastTime)
	thresholdDuration := time.Duration(rule.ThresholdDays) * 24 * time.Hour

	if gap >= thresholdDuration {
		days := int(gap.Hours() / 24)
		msg := renderTemplate(rule.MessageTemplate, map[string]string{
			"days": fmt.Sprintf("%d", days),
		})
		return true, msg, nil
	}

	return false, "", nil
}

func (c *Checker) evaluateDeadline(ctx context.Context, rule storage.ReminderRule, now time.Time) (bool, string, error) {
	if rule.ActionItemID == "" {
		return false, "", fmt.Errorf("deadline rule %q has no action_item_id", rule.ID)
	}

	dueDateStr, err := c.store.GetActionItemDueDate(ctx, rule.ActionItemID)
	if err != nil {
		return false, "", fmt.Errorf("looking up action item %q: %w", rule.ActionItemID, err)
	}
	if dueDateStr == "" {
		return false, "", nil
	}

	dueDate, err := parseTimestamp(dueDateStr)
	if err != nil {
		return false, "", fmt.Errorf("parsing due date: %w", err)
	}

	daysUntil := int(dueDate.Sub(now).Hours() / 24)

	if daysUntil <= 7 {
		urgency := "upcoming"
		if daysUntil <= 0 {
			urgency = "overdue"
		} else if daysUntil <= 1 {
			urgency = "tomorrow"
		} else if daysUntil <= 3 {
			urgency = "soon"
		}
		msg := renderTemplate(rule.MessageTemplate, map[string]string{
			"days":    fmt.Sprintf("%d", daysUntil),
			"urgency": urgency,
		})
		return true, msg, nil
	}

	return false, "", nil
}

func (c *Checker) evaluatePeriodic(rule storage.ReminderRule, now time.Time) (bool, string, error) {
	if rule.CronExpr == "" {
		return false, "", fmt.Errorf("periodic rule %q has no cron_expr", rule.ID)
	}

	if rule.LastTriggeredAt == "" {
		return true, rule.MessageTemplate, nil
	}

	lastTriggered, err := parseTimestamp(rule.LastTriggeredAt)
	if err != nil {
		return false, "", fmt.Errorf("parsing last_triggered_at: %w", err)
	}

	interval, err := parseCronInterval(rule.CronExpr)
	if err != nil {
		return false, "", fmt.Errorf("parsing cron expression %q: %w", rule.CronExpr, err)
	}

	if now.Sub(lastTriggered) >= interval {
		return true, rule.MessageTemplate, nil
	}

	return false, "", nil
}

// parseCronInterval provides a simple cron-like interval parser.
// Supports: "daily", "weekly", "monthly", and "@every <duration>" formats.
func parseCronInterval(expr string) (time.Duration, error) {
	switch strings.TrimSpace(strings.ToLower(expr)) {
	case "daily", "@daily":
		return 24 * time.Hour, nil
	case "weekly", "@weekly":
		return 7 * 24 * time.Hour, nil
	case "monthly", "@monthly":
		return 30 * 24 * time.Hour, nil
	default:
		if strings.HasPrefix(expr, "@every ") {
			durStr := strings.TrimPrefix(expr, "@every ")
			d, err := time.ParseDuration(durStr)
			if err != nil {
				return 0, fmt.Errorf("invalid duration %q: %w", durStr, err)
			}
			return d, nil
		}
		return 0, fmt.Errorf("unsupported cron expression %q", expr)
	}
}

// parseTimestamp tries multiple common timestamp formats.
func parseTimestamp(ts string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, ts); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", ts)
}

// renderTemplate replaces {{key}} placeholders in a template string.
func renderTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
