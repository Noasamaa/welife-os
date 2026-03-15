export type ActionItemPriority = "high" | "medium" | "low";
export type ActionItemStatus = "pending" | "in_progress" | "completed" | "dismissed";

export interface ActionItem {
  id: string;
  source_agent: string;
  source_session_id?: string;
  title: string;
  description: string;
  priority: ActionItemPriority;
  status: ActionItemStatus;
  category: string;
  related_entity_id?: string;
  due_date?: string;
  completed_at?: string;
  created_at: string;
}

export interface ActionPlanResponse {
  items: ActionItem[];
  count: number;
}
