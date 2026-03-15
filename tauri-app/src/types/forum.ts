export interface ForumSession {
  id: string;
  conversation_id: string;
  task_id: string;
  status: "running" | "completed" | "failed";
  summary?: string;
  created_at: string;
  completed_at?: string;
}

export interface ForumMessage {
  id: string;
  session_id: string;
  agent_name: string;
  round: number;
  stance: string;
  content: string;
  evidence?: string;
  confidence: number;
  created_at: string;
}

export interface ForumSessionDetail {
  session: ForumSession;
  messages: ForumMessage[];
}
