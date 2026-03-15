import type { SystemStatusResponse } from "../types/api";

export const API_BASE_URL = "http://127.0.0.1:18080";

export async function fetchSystemStatus(): Promise<SystemStatusResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/system/status`);
  if (!response.ok) {
    throw new Error(`failed to fetch system status: ${response.status}`);
  }

  return (await response.json()) as SystemStatusResponse;
}
