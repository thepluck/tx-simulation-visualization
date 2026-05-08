import { useCallback, useLayoutEffect, useRef, type Dispatch, type SetStateAction } from "react";
import type { ExpandMode, OutputView } from "../form";
import type { AddressLabels } from "../labels";
import type { SimulateResponse } from "../types";
import BalanceAnalysisView from "./BalanceAnalysisView";
import FundFlowGraph from "./FundFlowGraph";
import TraceTree from "./TraceTree";

type OutputPanelProps = {
  addressLabels: AddressLabels;
  expandMode: ExpandMode;
  explorerBaseUrl: string;
  outputView: OutputView;
  response: SimulateResponse | null;
  runMeta: string;
  target: string;
  onExpandModeChange: Dispatch<SetStateAction<ExpandMode>>;
  onOutputViewChange: Dispatch<SetStateAction<OutputView>>;
};

export default function OutputPanel(props: OutputPanelProps) {
  const { addressLabels, expandMode, explorerBaseUrl, outputView, response, runMeta, target, onExpandModeChange, onOutputViewChange } = props;
  const workspaceRef = useRef<HTMLElement | null>(null);
  const scrollPositionsRef = useRef<Partial<Record<OutputView, number>>>({});
  const pendingScrollTopRef = useRef<number | null>(null);

  const handleOutputViewChange = useCallback(
    (nextView: OutputView) => {
      const workspace = workspaceRef.current;
      const currentScrollTop = readScrollTop(workspace);
      scrollPositionsRef.current[outputView] = currentScrollTop;
      pendingScrollTopRef.current = scrollPositionsRef.current[nextView] ?? currentScrollTop;
      onOutputViewChange(nextView);
    },
    [onOutputViewChange, outputView]
  );

  useLayoutEffect(() => {
    const pendingScrollTop = pendingScrollTopRef.current;
    if (pendingScrollTop === null) {
      return;
    }
    pendingScrollTopRef.current = null;
    restoreScrollTop(workspaceRef.current, pendingScrollTop);
  }, [outputView]);

  return (
    <section
      className="workspace"
      aria-label="Simulation output"
      ref={workspaceRef}
      onScroll={(event) => {
        scrollPositionsRef.current[outputView] = event.currentTarget.scrollTop;
      }}
    >
      <div className="workspace-toolbar">
        <div>
          <h2>Output</h2>
          <p>{runMeta}</p>
        </div>
        <div className="view-tabs" role="tablist" aria-label="Output views">
          <ViewButton label="Trace" value="trace" active={outputView} onClick={handleOutputViewChange} />
          <ViewButton label="Flow" value="flow" active={outputView} onClick={handleOutputViewChange} />
          <ViewButton label="Balances" value="balances" active={outputView} onClick={handleOutputViewChange} />
          <ViewButton label="JSON" value="json" active={outputView} onClick={handleOutputViewChange} />
        </div>
      </div>

      {outputView === "trace" && (
        <section className="output-view active">
          <div className="section-bar">
            <h3>Transaction Trace</h3>
            <div className="icon-actions">
              <button type="button" title="Expand trace" onClick={() => onExpandModeChange("expand")}>
                +
              </button>
              <button type="button" title="Collapse trace" onClick={() => onExpandModeChange("collapse")}>
                -
              </button>
            </div>
          </div>
          <TraceTree
            addressLabels={addressLabels}
            explorerBaseUrl={explorerBaseUrl}
            nodes={response?.structuredTrace ?? []}
            target={target}
            expandMode={expandMode}
          />
        </section>
      )}

      {outputView === "flow" && (
        <section className="output-view active">
          <div className="section-bar">
            <h3>Fund Flow</h3>
            <span className="muted">{response?.erc20Transfers?.length ?? 0} transfers</span>
          </div>
          <FundFlowGraph
            addressLabels={addressLabels}
            analysis={response?.balanceAnalysis}
            explorerBaseUrl={explorerBaseUrl}
            transfers={response?.erc20Transfers ?? []}
          />
        </section>
      )}

      {outputView === "balances" && (
        <section className="output-view active">
          <div className="section-bar">
            <h3>Balance Analysis</h3>
            <span className="muted">{response?.balanceAnalysis?.changes?.length ?? 0} changes</span>
          </div>
          <BalanceAnalysisView addressLabels={addressLabels} analysis={response?.balanceAnalysis} explorerBaseUrl={explorerBaseUrl} />
        </section>
      )}

      {outputView === "json" && (
        <section className="output-view active">
          <div className="section-bar">
            <h3>Raw Response</h3>
          </div>
          <pre className="json-output">{JSON.stringify(response ?? {}, null, 2)}</pre>
        </section>
      )}
    </section>
  );
}

function ViewButton(props: { label: string; value: OutputView; active: OutputView; onClick: (value: OutputView) => void }) {
  return (
    <button type="button" className={`view-button ${props.active === props.value ? "active" : ""}`} onClick={() => props.onClick(props.value)}>
      {props.label}
    </button>
  );
}

function readScrollTop(element: HTMLElement | null): number {
  if (element && element.scrollHeight > element.clientHeight) {
    return element.scrollTop;
  }
  return typeof window === "undefined" ? 0 : window.scrollY;
}

function restoreScrollTop(element: HTMLElement | null, scrollTop: number) {
  if (element && element.scrollHeight > element.clientHeight) {
    element.scrollTo({ top: scrollTop });
    return;
  }
  window.scrollTo({ top: scrollTop });
}
