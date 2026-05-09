import { FormEvent, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchChainConfig, fetchHealth, fetchProjects, simulate } from "../api/client";
import OutputPanel from "../features/output/OutputPanel";
import RequestForm from "../features/request/RequestForm";
import { explorerForChain } from "../lib/explorer";
import {
  buildRequest,
  type ExpandMode,
  type FormState,
  type HealthStatus,
  type OutputView,
  type RequestTab,
  type ThemeMode
} from "./form";
import { buildAddressLabels } from "../lib/labels";
import { loadPersistedUIState, savePersistedUIState } from "../lib/persistence";
import type { SimulateRequest, SimulateResponse } from "../api/types";

export default function App() {
  const queryClient = useQueryClient();
  const initialUIState = useMemo(() => loadPersistedUIState(), []);
  const [form, setForm] = useState<FormState>(initialUIState.form);
  const [optimisticProjects, setOptimisticProjects] = useState<string[]>([]);
  const [requestTab, setRequestTab] = useState<RequestTab>(initialUIState.requestTab);
  const [outputView, setOutputView] = useState<OutputView>(initialUIState.outputView);
  const [response, setResponse] = useState<SimulateResponse | null>(initialUIState.response);
  const [theme, setTheme] = useState<ThemeMode>(initialUIState.theme);
  const [error, setError] = useState("");
  const [expandMode, setExpandMode] = useState<ExpandMode>("depth");
  const [traceExpandDepth, setTraceExpandDepth] = useState(initialUIState.traceExpandDepth);
  const simulationAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    savePersistedUIState({ form, outputView, requestTab, response, theme, traceExpandDepth });
  }, [form, outputView, requestTab, response, theme, traceExpandDepth]);

  useLayoutEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

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
  const addressLabels = useMemo(() => buildAddressLabels(form.labelOverrides, form.sender, response), [form.labelOverrides, form.sender, response]);
  const explorerBaseUrl = useMemo(() => explorerForChain(explorerUrls, form.chain), [explorerUrls, form.chain]);

  const update = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((current) => ({ ...current, [key]: value }));
  };

  const simulation = useMutation({
    mutationFn: ({ apiUrl, request, signal }: { apiUrl: string; request: SimulateRequest; signal: AbortSignal }) =>
      simulate(apiUrl, request, signal),
    onSuccess: (result, variables) => {
      setResponse(result);
      setOutputView(result.erc20Transfers?.length ? "flow" : result.balanceAnalysis ? "balances" : "trace");
      setExpandMode("depth");
      setOptimisticProjects([]);
      void queryClient.invalidateQueries({ queryKey: ["projects", variables.apiUrl] });
    },
    onError: (err) => {
      if (isAbortError(err)) {
        setError("Simulation aborted");
        return;
      }
      setError(err instanceof Error ? err.message : String(err));
    },
    onSettled: (_result, _err, variables) => {
      if (simulationAbortRef.current?.signal === variables?.signal) {
        simulationAbortRef.current = null;
      }
    }
  });

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (simulation.isPending) {
      return;
    }
    setError("");
    const controller = new AbortController();
    simulationAbortRef.current = controller;
    simulation.mutate({ apiUrl: form.apiUrl, request: buildRequest(form), signal: controller.signal });
  };

  const abortSimulation = () => {
    simulationAbortRef.current?.abort();
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
        theme={theme}
        onAbort={abortSimulation}
        onProjectBrowsed={(path) => {
          setOptimisticProjects((current) => mergeProjects([path], current).slice(0, 20));
          void queryClient.invalidateQueries({ queryKey: ["projects", form.apiUrl] });
        }}
        onRequestTabChange={setRequestTab}
        onSubmit={submit}
        onThemeChange={setTheme}
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
        onExpandDepthChange={setTraceExpandDepth}
        onExpandModeChange={setExpandMode}
        onOutputViewChange={setOutputView}
      />
    </main>
  );
}

function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === "AbortError";
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
