export interface ImportJob {
  id: string;
  task_id: string;
  file_name: string;
  format: string;
  status: "pending" | "running" | "succeeded" | "failed";
  conversation_id?: string;
  message_count: number;
  error_message?: string;
  started_at: string;
  completed_at?: string;
}

export interface ImportResult {
  job_id: string;
  task_id: string;
  conversation_id?: string;
  message_count: number;
}

export interface Conversation {
  id: string;
  platform: string;
  conversation_type: string;
  title?: string;
  message_count: number;
  first_message_at?: string;
  last_message_at?: string;
}

export interface GraphNode {
  id: string;
  type: string;
  name: string;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: string;
  weight: number;
}

export interface GraphOverview {
  nodes: GraphNode[];
  edges: GraphEdge[];
  stats: {
    entity_count: number;
    relationship_count: number;
    entity_types: Record<string, number>;
  };
}
