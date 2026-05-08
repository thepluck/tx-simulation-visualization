import type { ChainConfig, ProjectsResponse, SimulateRequest, SimulateResponse } from "./types";

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

export async function fetchProjects(apiUrl: string): Promise<ProjectsResponse> {
  const response = await fetch(`${trimSlash(apiUrl)}/projects`);
  if (!response.ok) {
    throw new Error(`projects request failed: ${response.status}`);
  }
  const payload = (await response.json()) as { projects?: string[] };
  return {
    projects: payload.projects ?? []
  };
}

export async function browseProject(apiUrl: string): Promise<string> {
  const response = await fetch(`${trimSlash(apiUrl)}/browse/project`);
  const payload = (await response.json()) as { error?: string; path?: string };
  if (!response.ok) {
    throw new Error(payload.error || `browse project request failed: ${response.status}`);
  }
  if (!payload.path) {
    throw new Error("browse project response missing path");
  }
  return payload.path;
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
