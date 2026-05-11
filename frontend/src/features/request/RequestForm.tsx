import { useState, type FormEventHandler } from "react";
import { browseProject } from "../../api/client";
import type { FormState, HealthStatus, RequestTab, ThemeMode, UpdateForm } from "../../app/form";
import ProjectHistoryDropdown from "./ProjectHistoryDropdown";
import ScriptOverridesTab from "./ScriptOverridesTab";

type RequestFormProps = {
  chains: string[];
  error: string;
  form: FormState;
  isOpeningRequest: boolean;
  isRunning: boolean;
  projectSuggestions: string[];
  requestLookupId: string;
  requestTab: RequestTab;
  status: HealthStatus;
  theme: ThemeMode;
  onAbort: () => void;
  onOpenRequest: () => void;
  onProjectBrowsed: (path: string) => void;
  onRequestTabChange: (value: RequestTab) => void;
  onRequestLookupIdChange: (value: string) => void;
  onSubmit: FormEventHandler<HTMLFormElement>;
  onThemeChange: (value: ThemeMode) => void;
  onUpdate: UpdateForm;
};

export default function RequestForm(props: RequestFormProps) {
  const {
    chains,
    error,
    form,
    isOpeningRequest,
    isRunning,
    projectSuggestions,
    requestLookupId,
    requestTab,
    status,
    theme,
    onAbort,
    onOpenRequest,
    onProjectBrowsed,
    onRequestTabChange,
    onRequestLookupIdChange,
    onSubmit,
    onThemeChange,
    onUpdate
  } = props;
  const [browseError, setBrowseError] = useState("");
  const [isBrowsingProject, setIsBrowsingProject] = useState(false);

  const handleBrowseProject = async () => {
    setBrowseError("");
    setIsBrowsingProject(true);
    try {
      const path = await browseProject(form.apiUrl);
      onUpdate("projectPath", path);
      onProjectBrowsed(path);
    } catch (err) {
      setBrowseError(err instanceof Error ? err.message : String(err));
    } finally {
      setIsBrowsingProject(false);
    }
  };

  return (
    <section className="control-panel" aria-label="Simulation request">
      <div className="panel-header">
        <h1>Foundry Tx Simulator</h1>
        <div className="panel-header-actions">
          <button
            aria-label={theme === "dark" ? "Use light theme" : "Use dark theme"}
            className="theme-toggle"
            type="button"
            onClick={() => onThemeChange(theme === "dark" ? "light" : "dark")}
          >
            {theme === "dark" ? "Light" : "Dark"}
          </button>
          <span className={`status-pill ${status}`}>{status}</span>
        </div>
      </div>

      <form className="request-form" onSubmit={onSubmit}>
        <label>
          API URL
          <input value={form.apiUrl} onChange={(event) => onUpdate("apiUrl", event.target.value)} />
        </label>

        <div className="field-block">
          <label htmlFor="request-id">Request ID</label>
          <span className="request-id-field">
            <input
              id="request-id"
              value={requestLookupId}
              placeholder="20260511T120000.000000000-deadbeef"
              onChange={(event) => onRequestLookupIdChange(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  onOpenRequest();
                }
              }}
            />
            <button
              className="lookup-button"
              type="button"
              disabled={isRunning || isOpeningRequest || !requestLookupId.trim()}
              onClick={onOpenRequest}
            >
              {isOpeningRequest ? "Opening..." : "Open"}
            </button>
          </span>
        </div>

        <div className="two-col">
          <label>
            Chain
            <select value={form.chain} onChange={(event) => onUpdate("chain", event.target.value)}>
              {chains.map((chain) => (
                <option key={chain} value={chain}>
                  {chain}
                </option>
              ))}
            </select>
          </label>
          <label>
            Block
            <input
              value={form.blockNumber}
              placeholder="23000000"
              onChange={(event) => onUpdate("blockNumber", event.target.value)}
            />
          </label>
        </div>

        <div className="field-block">
          <label htmlFor="foundry-project">Foundry Project</label>
          <span className="browse-field">
            <input
              className="project-path-input"
              id="foundry-project"
              value={form.projectPath}
              placeholder="~/foundry-project"
              onChange={(event) => onUpdate("projectPath", event.target.value)}
            />
            <ProjectHistoryDropdown projects={projectSuggestions} onSelect={(path) => onUpdate("projectPath", path)} />
            <button className="browse-button" type="button" disabled={isBrowsingProject} onClick={handleBrowseProject}>
              {isBrowsingProject ? "Choosing..." : "Browse"}
            </button>
          </span>
          {browseError && <span className="field-error">{browseError}</span>}
        </div>

        <label>
          Sender
          <input value={form.sender} placeholder="0x..." onChange={(event) => onUpdate("sender", event.target.value)} />
        </label>

        <label>
          Target
          <input value={form.target} placeholder="0x..." onChange={(event) => onUpdate("target", event.target.value)} />
        </label>

        <label>
          Calldata
          <textarea
            value={form.data}
            rows={3}
            spellCheck={false}
            placeholder="0x"
            onChange={(event) => onUpdate("data", event.target.value)}
          />
        </label>

        <div className="tabs" role="tablist" aria-label="Request sections">
          <TabButton label="Override Options" value="overrides" active={requestTab} onClick={onRequestTabChange} />
          <TabButton label="Override Contract" value="state" active={requestTab} onClick={onRequestTabChange} />
          <TabButton label="Compiler" value="compiler" active={requestTab} onClick={onRequestTabChange} />
        </div>

        {requestTab === "overrides" && <ScriptOverridesTab form={form} onUpdate={onUpdate} />}
        {requestTab === "state" && <StateTab form={form} onUpdate={onUpdate} />}
        {requestTab === "compiler" && <CompilerTab form={form} onUpdate={onUpdate} />}

        {error && <div className="error-box">{error}</div>}
        <button
          className={`primary-action ${isRunning ? "abort-action" : ""}`}
          type={isRunning ? "button" : "submit"}
          onClick={isRunning ? onAbort : undefined}
        >
          {isRunning ? "Abort" : "Run Simulation"}
        </button>
      </form>
    </section>
  );
}

function StateTab(props: { form: FormState; onUpdate: UpdateForm }) {
  const { form, onUpdate } = props;
  return (
    <section className="tab-panel active">
      <label>
        Override Contract Name
        <input
          value={form.stateContractName}
          placeholder="WETHStateOverride"
          onChange={(event) => onUpdate("stateContractName", event.target.value)}
        />
      </label>
      <label>
        Override Contract Source
        <textarea
          value={form.stateSource}
          rows={10}
          spellCheck={false}
          placeholder={"// SPDX-License-Identifier: UNLICENSED\npragma solidity ^0.8.0;"}
          onChange={(event) => onUpdate("stateSource", event.target.value)}
        />
      </label>
    </section>
  );
}

function CompilerTab(props: { form: FormState; onUpdate: UpdateForm }) {
  const { form, onUpdate } = props;
  return (
    <section className="tab-panel active">
      <div className="toggle-grid">
        <Checkbox label="viaIR" checked={form.viaIR} onChange={(value) => onUpdate("viaIR", value)} />
        <Checkbox label="optimize" checked={form.optimize} onChange={(value) => onUpdate("optimize", value)} />
        <Checkbox label="offline" checked={form.offline} onChange={(value) => onUpdate("offline", value)} />
        <Checkbox label="no metadata" checked={form.noMetadata} onChange={(value) => onUpdate("noMetadata", value)} />
      </div>
      <div className="two-col">
        <label>
          Solc
          <input value={form.compilerUse} onChange={(event) => onUpdate("compilerUse", event.target.value)} />
        </label>
        <label>
          Optimizer Runs
          <input
            value={form.optimizerRuns}
            inputMode="numeric"
            placeholder="200"
            onChange={(event) => onUpdate("optimizerRuns", event.target.value)}
          />
        </label>
      </div>
      <div className="two-col">
        <label>
          EVM Version
          <input value={form.evmVersion} onChange={(event) => onUpdate("evmVersion", event.target.value)} />
        </label>
        <label>
          Revert Strings
          <select value={form.revertStrings} onChange={(event) => onUpdate("revertStrings", event.target.value)}>
            <option value=""></option>
            <option value="default">default</option>
            <option value="strip">strip</option>
            <option value="debug">debug</option>
            <option value="verboseDebug">verboseDebug</option>
          </select>
        </label>
      </div>
    </section>
  );
}

function TabButton(props: { label: string; value: RequestTab; active: RequestTab; onClick: (value: RequestTab) => void }) {
  return (
    <button type="button" className={`tab-button ${props.active === props.value ? "active" : ""}`} onClick={() => props.onClick(props.value)}>
      {props.label}
    </button>
  );
}

function Checkbox(props: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <label className="checkbox-row">
      <input type="checkbox" checked={props.checked} onChange={(event) => props.onChange(event.target.checked)} />
      {props.label}
    </label>
  );
}
