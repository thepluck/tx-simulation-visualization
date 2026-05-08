import type { ChainConfig, SimulateRequest, SimulateResponse } from "./types";

export async function fetchChainConfig(apiUrl: string): Promise<ChainConfig> {
  const response = await fetch(`${trimSlash(apiUrl)}/chains`);
  if (!response.ok) {
    throw new Error(`chains request failed: ${response.status}`);
  }
  const payload = (await response.json()) as { chains?: string[]; explorerUrls?: Record<string, string> };
  return {
    chains: payload.chains ?? [],
    explorerUrls: payload.explorerUrls ?? {}
  };
}

export async function fetchHealth(apiUrl: string): Promise<boolean> {
  const response = await fetch(`${trimSlash(apiUrl)}/health`);
  return response.ok;
}

export async function simulate(apiUrl: string, request: SimulateRequest): Promise<SimulateResponse> {
  const response = await fetch(`${trimSlash(apiUrl)}/simulate`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(request)
  });
  const payload = (await response.json()) as SimulateResponse;
  if (!response.ok) {
    throw new Error(payload.error || `simulate request failed: ${response.status}`);
  }
  return payload;
}

function trimSlash(value: string): string {
  return value.replace(/\/+$/, "");
}
