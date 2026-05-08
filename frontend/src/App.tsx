import { FormEvent, useEffect, useMemo, useState } from "react";
import { fetchChainConfig, fetchHealth, simulate } from "./api";
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
import type { SimulateResponse } from "./types";

export default function App() {
  const [form, setForm] = useState<FormState>(() => loadPersistedUIState().form);
  const [chains, setChains] = useState<string[]>(["mainnet"]);
  const [explorerUrls, setExplorerUrls] = useState<Record<string, string>>({});
  const [status, setStatus] = useState<HealthStatus>("offline");
  const [requestTab, setRequestTab] = useState<RequestTab>(() => loadPersistedUIState().requestTab);
  const [outputView, setOutputView] = useState<OutputView>("trace");
  const [response, setResponse] = useState<SimulateResponse | null>(null);
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [expandMode, setExpandMode] = useState<ExpandMode>("depth");
  const [traceExpandDepth, setTraceExpandDepth] = useState(() => loadPersistedUIState().traceExpandDepth);

  useEffect(() => {
    savePersistedUIState({ form, requestTab, traceExpandDepth });
  }, [form, requestTab, traceExpandDepth]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const [ok, chainConfig] = await Promise.all([fetchHealth(form.apiUrl), fetchChainConfig(form.apiUrl)]);
        if (cancelled) {
          return;
        }
        setStatus(ok ? "online" : "error");
        setExplorerUrls(chainConfig.explorerUrls);
        if (chainConfig.chains.length > 0) {
          setChains(chainConfig.chains);
          if (!chainConfig.chains.includes(form.chain)) {
            setForm((current) => ({ ...current, chain: chainConfig.chains[0] }));
          }
        }
      } catch {
        if (!cancelled) {
          setStatus("error");
        }
      }
    };
    const timer = window.setTimeout(load, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [form.apiUrl, form.chain]);

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

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setIsRunning(true);
    try {
      const request = buildRequest(form);
      const result = await simulate(form.apiUrl, request);
      setResponse(result);
      setOutputView(result.erc20Transfers?.length ? "flow" : result.balanceAnalysis ? "balances" : "trace");
      setExpandMode("depth");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setIsRunning(false);
    }
  };

  return (
    <main className="app-shell">
      <RequestForm
        chains={chains}
        error={error}
        form={form}
        isRunning={isRunning}
        requestTab={requestTab}
        status={status}
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
