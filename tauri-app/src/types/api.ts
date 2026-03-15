export interface SystemStatusResponse {
  backend: {
    status: string;
    version: string;
  };
  storage: {
    driver: string;
    ready: boolean;
    path: string;
  };
  llm: {
    provider: string;
    reachable: boolean;
    base_url: string;
    model: string;
  };
}

export interface LLMConfig {
  provider: string;
  base_url: string;
  model: string;
  api_key: string;
  embedding_model: string;
}
