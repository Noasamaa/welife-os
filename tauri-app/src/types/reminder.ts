export interface ReminderRule {
  id: string;
  action_item_id?: string;
  rule_type: "contact_gap" | "deadline" | "periodic";
  entity_id?: string;
  threshold_days?: number;
  cron_expr?: string;
  message_template: string;
  enabled: boolean;
  last_triggered_at?: string;
  created_at: string;
}

export interface Reminder {
  id: string;
  rule_id: string;
  message: string;
  status: "pending" | "read" | "dismissed";
  triggered_at: string;
  read_at?: string;
}
