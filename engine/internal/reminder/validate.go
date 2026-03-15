package reminder

import (
	"fmt"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/storage"
)

// ValidateRule checks whether a reminder rule has the fields required by its type.
func ValidateRule(rule storage.ReminderRule) error {
	if strings.TrimSpace(rule.RuleType) == "" {
		return fmt.Errorf("rule_type is required")
	}
	if strings.TrimSpace(rule.MessageTemplate) == "" {
		return fmt.Errorf("message_template is required")
	}

	switch rule.RuleType {
	case "contact_gap":
		if strings.TrimSpace(rule.EntityID) == "" {
			return fmt.Errorf("entity_id is required for contact_gap rules")
		}
		if rule.ThresholdDays <= 0 {
			return fmt.Errorf("threshold_days must be greater than 0 for contact_gap rules")
		}
	case "deadline":
		if strings.TrimSpace(rule.ActionItemID) == "" {
			return fmt.Errorf("action_item_id is required for deadline rules")
		}
	case "periodic":
		if strings.TrimSpace(rule.CronExpr) == "" {
			return fmt.Errorf("cron_expr is required for periodic rules")
		}
		if _, err := parseCronInterval(rule.CronExpr); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown rule type %q", rule.RuleType)
	}

	return nil
}
