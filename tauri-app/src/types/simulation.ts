export interface PersonProfile {
  id: string;
  entity_id: string;
  name: string;
  personality: string;
  relationship_to_self: string;
  behavioral_patterns?: string;
  source_conversation_ids?: string;
  created_at: string;
  updated_at: string;
}

export interface SimulationSession {
  id: string;
  conversation_id: string;
  task_id: string;
  fork_description: string;
  status: "running" | "completed" | "failed";
  step_count: number;
  original_graph_snapshot?: string;
  final_graph_snapshot?: string;
  narrative?: string;
  created_at: string;
  completed_at?: string;
}

export interface SimulationStep {
  id: string;
  session_id: string;
  step_number: number;
  description: string;
  entity_changes: string;
  reactions: string;
  created_at: string;
}

export interface SimulationDetail {
  session: SimulationSession;
  steps: SimulationStep[];
}
