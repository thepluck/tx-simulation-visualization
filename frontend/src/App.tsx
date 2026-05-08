import { FormEvent, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchChainConfig, fetchHealth, fetchProjects, simulate } from "./api";
import OutputPanel from "./components/OutputPanel";
import RequestForm from "./components/RequestForm";
import { explorerForChain } from "./explorer";
import {
  buildRequest,
  type ExpandMode,
  type FormState,
  type HealthStatus,
  type OutputView,
  type RequestTab
} from "./form";
import { buildAddressLabels } from "./labels";
import { loadPersistedUIState, savePersistedUIState } from "./persistence";
import type { SimulateRequest, SimulateResponse } from "./types";

export default function App() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<FormState>(() => loadPersistedUIState().form);
  const [optimisticProjects, setOptimisticProjects] = useState<string[]>([]);
  const [requestTab, setRequestTab] = useState<RequestTab>(() => loadPersistedUIState().requestTab);
  const [outputView, setOutputView] = useState<OutputView>("trace");
  const [response, setResponse] = useState<SimulateResponse | null>(null);
  const [error, setError] = useState("");
  const [expandMode, setExpandMode] = useState<ExpandMode>("depth");
  const [traceExpandDepth, setTraceExpandDepth] = useState(() => loadPersistedUIState().traceExpandDepth);

  useEffect(() => {
    savePersistedUIState({ form, requestTab, traceExpandDepth });
  }, [form, requestTab, traceExpandDepth]);

  const healthQuery = useQuery({
    queryKey: ["health", form.apiUrl],
    queryFn: () => fetchHealth(form.apiUrl),
    refetchInterval: 10_000
  });
  const chainQuery = useQuery({
    queryKey: ["chains", form.apiUrl],
    queryFn: () => fetchChainConfig(form.apiUrl)
  });
  const projectQuery = useQuery({
    queryKey: ["projects", form.apiUrl],
    queryFn: () => fetchProjects(form.apiUrl)
  });

  const chains = chainQuery.data?.chains.length ? chainQuery.data.chains : ["mainnet"];
  const explorerUrls = useMemo(() => chainQuery.data?.explorerUrls ?? {}, [chainQuery.data?.explorerUrls]);
  const projectSuggestions = useMemo(
    () => mergeProjects(optimisticProjects, projectQuery.data?.projects ?? []),
    [optimisticProjects, projectQuery.data?.projects]
  );
  const status: HealthStatus = healthQuery.isSuccess ? (healthQuery.data ? "online" : "error") : healthQuery.isError ? "error" : "offline";

  useEffect(() => {
    if (chainQuery.data?.chains.length && !chainQuery.data.chains.includes(form.chain)) {
      setForm((current) => ({ ...current, chain: chainQuery.data.chains[0] }));
    }
  }, [chainQuery.data, form.chain]);

  const runMeta = useMemo(() => {
    if (!response) {
      return "No run yet";
    }
    const state = response.success ? "success" : "failed";
    return `${state} | ${response.durationMillis}ms | exit ${response.exitCode} | ${response.id}`;
  }, [response]);
  const addressLabels = useMemo(() => buildAddressLabels(form.labelOverrides, form.sender), [form.labelOverrides, form.sender]);
  const explorerBaseUrl = useMemo(() => explorerForChain(explorerUrls, form.chain), [explorerUrls, form.chain]);

  const update = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((current) => ({ ...current, [key]: value }));
  };

  const simulation = useMutation({
    mutationFn: ({ apiUrl, request }: { apiUrl: string; request: SimulateRequest }) => simulate(apiUrl, request),
    onSuccess: (result, variables) => {
      setResponse(result);
      setOutputView(result.erc20Transfers?.length ? "flow" : result.balanceAnalysis ? "balances" : "trace");
      setExpandMode("depth");
      setOptimisticProjects([]);
      void queryClient.invalidateQueries({ queryKey: ["projects", variables.apiUrl] });
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : String(err));
    }
  });

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    simulation.mutate({ apiUrl: form.apiUrl, request: buildRequest(form) });
  };

  return (
    <main className="app-shell">
      <RequestForm
        chains={chains}
        error={error}
        form={form}
        isRunning={simulation.isPending}
        projectSuggestions={projectSuggestions}
        requestTab={requestTab}
        status={status}
        onProjectBrowsed={(path) => {
          setOptimisticProjects((current) => mergeProjects([path], current).slice(0, 20));
          void queryClient.invalidateQueries({ queryKey: ["projects", form.apiUrl] });
        }}
        onRequestTabChange={setRequestTab}
        onSubmit={submit}
        onUpdate={update}
      />
      <OutputPanel
        addressLabels={addressLabels}
        expandMode={expandMode}
        expandDepth={traceExpandDepth}
        explorerBaseUrl={explorerBaseUrl}
        outputView={outputView}
        response={response}
        runMeta={runMeta}
        target={form.target}
        onExpandDepthChange={setTraceExpandDepth}
        onExpandModeChange={setExpandMode}
        onOutputViewChange={setOutputView}
      />
    </main>
  );
}

function mergeProjects(primary: string[], secondary: string[]): string[] {
  const seen = new Set<string>();
  const merged: string[] = [];
  for (const path of [...primary, ...secondary]) {
    if (!path || seen.has(path)) {
      continue;
    }
    seen.add(path);
    merged.push(path);
  }
  return merged;
}
