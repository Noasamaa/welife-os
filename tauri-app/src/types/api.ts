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
