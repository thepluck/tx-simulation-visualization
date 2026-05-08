import { useEffect, useMemo, useState } from "react";
import type { ExpandMode } from "../form";
import { looksLikeTraceLabel, resolveAddressReference, resolveLabelAlias, type AddressLabels } from "../labels";
import type { TraceNode } from "../types";
import AddressReference from "./AddressReference";
import TraceArguments from "./TraceArguments";

type TraceTreeProps = {
  addressLabels: AddressLabels;
  expandMode: ExpandMode;
  expandDepth: number;
  explorerBaseUrl: string;
  nodes: TraceNode[];
};

export default function TraceTree(props: TraceTreeProps) {
  const visibleNodes = useMemo(() => mainCallTrace(props.nodes), [props.nodes]);

  if (visibleNodes.length === 0) {
    return <div className="trace-tree empty-state">No trace</div>;
  }
  return (
    <div className="trace-tree">
      {visibleNodes.map((node, index) => (
        <TraceNodeView
          addressLabels={props.addressLabels}
          explorerBaseUrl={props.explorerBaseUrl}
          key={`${node.raw}-${index}`}
          node={node}
          expandMode={props.expandMode}
          expandDepth={props.expandDepth}
          depth={0}
        />
      ))}
    </div>
  );
}

function TraceNodeView(props: {
  addressLabels: AddressLabels;
  depth: number;
  expandDepth: number;
  explorerBaseUrl: string;
  node: TraceNode;
  expandMode: ExpandMode;
}) {
  const hasChildren = Boolean(props.node.children?.length);
  const [open, setOpen] = useState(() => shouldOpenAtDepth(props.depth, props.expandDepth));

  useEffect(() => {
    if (props.expandMode === "expand") {
      setOpen(true);
    }
    if (props.expandMode === "collapse") {
      setOpen(false);
    }
    if (props.expandMode === "depth") {
      setOpen(shouldOpenAtDepth(props.depth, props.expandDepth));
    }
  }, [props.depth, props.expandDepth, props.expandMode]);

  const main = traceLabel(props.node, props.addressLabels, props.explorerBaseUrl);
  const kind = traceKindLabel(props.node);
  const meta = props.node.gas ? `${props.node.gas} gas` : "";
  const content = (
    <>
      <span className="trace-kind">{kind}</span>
      <span className="trace-main">{main}</span>
      <span className="trace-meta">{meta}</span>
    </>
  );

  if (!hasChildren) {
    return <div className="trace-leaf">{content}</div>;
  }

  return (
    <details className="trace-node" open={open} onToggle={(event) => setOpen(event.currentTarget.open)}>
      <summary>{content}</summary>
      <div className="trace-children">
        {props.node.children?.map((child, index) => (
          <TraceNodeView
            addressLabels={props.addressLabels}
            explorerBaseUrl={props.explorerBaseUrl}
            key={`${child.raw}-${index}`}
            node={child}
            expandMode={props.expandMode}
            expandDepth={props.expandDepth}
            depth={props.depth + 1}
          />
        ))}
      </div>
    </details>
  );
}

function traceLabel(node: TraceNode, addressLabels: AddressLabels, explorerBaseUrl: string) {
  if (node.kind === "call") {
    const addressRef = resolveAddressReference(node.target, addressLabels);
    const suffix = (
      <>
        ::{node.function ?? "call"}
        {node.arguments ? (
          <>
            (<TraceArguments addressLabels={addressLabels} explorerBaseUrl={explorerBaseUrl} value={node.arguments} />)
          </>
        ) : null}
      </>
    );
    if (addressRef) {
      return (
        <>
          <AddressReference address={addressRef.address} addressLabels={addressLabels} displayLabel={addressRef.label} explorerBaseUrl={explorerBaseUrl} />
          {suffix}
        </>
      );
    }
    const target = resolveLabelAlias(node.target ?? "unknown", addressLabels);
    return (
      <>
        {looksLikeTraceLabel(target) ? <span className="address-reference-text">{target}</span> : target}
        {suffix}
      </>
    );
  }
  if (node.kind === "event") {
    return <TraceArguments addressLabels={addressLabels} explorerBaseUrl={explorerBaseUrl} value={withoutEmitPrefix(node.value ?? node.raw)} />;
  }
  return <TraceArguments addressLabels={addressLabels} explorerBaseUrl={explorerBaseUrl} value={resultDisplayValue(node)} />;
}

function traceKindLabel(node: TraceNode): string {
  if (node.kind === "call" && node.callType) {
    return node.callType;
  }
  return node.kind;
}

function withoutEmitPrefix(value: string): string {
  return value.replace(/^emit\s+/, "");
}

function shouldOpenAtDepth(depth: number, expandDepth: number): boolean {
  return depth < expandDepth;
}

function mainCallTrace(nodes: TraceNode[]): TraceNode[] {
  for (let index = nodes.length - 1; index >= 0; index -= 1) {
    const node = nodes[index];
    if (isScriptWrapperCall(node)) {
      const child = scriptMainCall(node.children ?? []);
      return child ? visibleTrace([child]) : [];
    }
    if (node.kind === "call") {
      return visibleTrace([node]);
    }
  }
  return [];
}

function scriptMainCall(nodes: TraceNode[]): TraceNode | undefined {
  const getRecordedLogsIndex = findLastCallFunction(nodes, "getRecordedLogs");
  if (getRecordedLogsIndex > 0) {
    return lastDirectCall(nodes.slice(0, getRecordedLogsIndex));
  }
  return lastDirectCall(nodes);
}

function findLastCallFunction(nodes: TraceNode[], name: string): number {
  for (let index = nodes.length - 1; index >= 0; index -= 1) {
    if (isCallFunction(nodes[index], name)) {
      return index;
    }
  }
  return -1;
}

function lastDirectCall(nodes: TraceNode[]): TraceNode | undefined {
  for (let index = nodes.length - 1; index >= 0; index -= 1) {
    if (nodes[index].kind === "call") {
      return nodes[index];
    }
  }
  return undefined;
}

function isScriptWrapperCall(node: TraceNode): boolean {
  return node.kind === "call" && node.function === "run" && (node.target ?? "").includes("SimulateTxScript");
}

function isCallFunction(node: TraceNode, name: string): boolean {
  return node.kind === "call" && node.function === name;
}

function visibleTrace(nodes: TraceNode[]): TraceNode[] {
  return nodes.flatMap((node) => {
    const visible = withoutEmptyResult(node);
    return visible ? [visible] : [];
  });
}

function withoutEmptyResult(node: TraceNode): TraceNode | undefined {
  if (isEmptyResult(node)) {
    return undefined;
  }

  const children = (node.children ?? []).flatMap((child) => {
    const visible = withoutEmptyResult(child);
    return visible ? [visible] : [];
  });
  if (children.length === (node.children?.length ?? 0)) {
    return node;
  }

  const next: TraceNode = { ...node };
  if (children.length > 0) {
    next.children = children;
  } else {
    delete next.children;
  }
  return next;
}

function isEmptyResult(node: TraceNode): boolean {
  const kind = node.kind.toLowerCase();
  return isResultKind(kind) && !resultDisplayValue(node);
}

function isResultKind(kind: string): boolean {
  return kind === "return" || kind === "stop" || kind === "result";
}

function resultDisplayValue(node: TraceNode): string {
  const value = node.value?.trim() ?? "";
  if (value) {
    return isResultEcho(value, node) ? "" : value;
  }
  const raw = node.raw.trim();
  if (isResultEcho(raw, node)) {
    return "";
  }
  return raw;
}

function isResultEcho(value: string, node: TraceNode): boolean {
  const resultType = (node.resultType || node.kind).trim();
  if (!resultType) {
    return false;
  }
  const normalized = value.replace(/^←\s*/, "").trim().toLowerCase();
  return normalized === `[${resultType.toLowerCase()}]`;
}
