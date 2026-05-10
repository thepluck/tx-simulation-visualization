import { useEffect, useMemo, useRef, useState } from "react";
import type { ExpandMode } from "../../app/form";
import { isAddress, looksLikeTraceLabel, resolveAddressReference, resolveLabelAlias, type AddressLabels } from "../../lib/labels";
import type { TraceNode } from "../../api/types";
import AddressReference from "../../components/AddressReference";
import { highlightSearchText } from "../../components/SearchHighlight";
import TraceArguments from "./TraceArguments";

type TraceTreeProps = {
  addressLabels: AddressLabels;
  expandMode: ExpandMode;
  expandDepth: number;
  explorerBaseUrl: string;
  nodes: TraceNode[];
  searchMatchIndex: number;
  searchQuery: string;
  onSearchMatchCountChange: (count: number) => void;
};

type SearchNode = {
  children: SearchNode[];
  matchIndex: number | null;
  node: TraceNode;
  selfMatches: boolean;
  subtreeMatches: boolean;
};

export default function TraceTree({
  addressLabels,
  expandDepth,
  expandMode,
  explorerBaseUrl,
  nodes,
  onSearchMatchCountChange,
  searchMatchIndex,
  searchQuery
}: TraceTreeProps) {
  const visibleNodes = useMemo(() => mainCallTrace(nodes), [nodes]);
  const searchTerms = useMemo(() => traceSearchTerms(searchQuery, addressLabels), [addressLabels, searchQuery]);
  const searchResult = useMemo(() => buildSearchTree(visibleNodes, searchTerms, addressLabels), [addressLabels, searchTerms, visibleNodes]);

  useEffect(() => {
    onSearchMatchCountChange(searchResult.matchCount);
  }, [onSearchMatchCountChange, searchResult.matchCount]);

  if (visibleNodes.length === 0) {
    return <div className="trace-tree empty-state">No trace</div>;
  }
  return (
    <div className="trace-tree">
      {searchResult.nodes.map((searchNode, index) => (
        <TraceNodeView
          addressLabels={addressLabels}
          explorerBaseUrl={explorerBaseUrl}
          key={`${searchNode.node.raw}-${index}`}
          searchNode={searchNode}
          expandMode={expandMode}
          expandDepth={expandDepth}
          depth={0}
          hasSearch={searchTerms.length > 0}
          highlightTerms={searchTerms}
          searchMatchIndex={searchMatchIndex}
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
  searchNode: SearchNode;
  expandMode: ExpandMode;
  hasSearch: boolean;
  highlightTerms: string[];
  searchMatchIndex: number;
}) {
  const hasChildren = props.searchNode.children.length > 0;
  const [open, setOpen] = useState(() => shouldOpenAtDepth(props.depth, props.expandDepth));
  const rowRef = useRef<HTMLElement | null>(null);
  const isActiveMatch = props.searchNode.matchIndex === props.searchMatchIndex;

  useEffect(() => {
    if (props.hasSearch && props.searchNode.subtreeMatches) {
      setOpen(true);
      return;
    }
    if (props.hasSearch) {
      return;
    }
    if (props.expandMode === "expand") {
      setOpen(true);
    }
    if (props.expandMode === "collapse") {
      setOpen(false);
    }
    if (props.expandMode === "depth") {
      setOpen(shouldOpenAtDepth(props.depth, props.expandDepth));
    }
  }, [props.depth, props.expandDepth, props.expandMode, props.hasSearch, props.searchNode.subtreeMatches]);

  useEffect(() => {
    if (isActiveMatch) {
      rowRef.current?.scrollIntoView({ block: "center" });
    }
  }, [isActiveMatch]);

  const main = traceLabel(props.searchNode.node, props.addressLabels, props.explorerBaseUrl, props.highlightTerms);
  const kind = traceKindLabel(props.searchNode.node);
  const meta = props.searchNode.node.gas ? `${props.searchNode.node.gas} gas` : "";
  const rowClassName = traceRowClassName(props.searchNode.selfMatches, isActiveMatch);
  const content = (
    <>
      <span className="trace-kind">{highlightSearchText(kind, props.highlightTerms)}</span>
      <span className="trace-main">{main}</span>
      <span className="trace-meta">{highlightSearchText(meta, props.highlightTerms)}</span>
    </>
  );

  if (!hasChildren) {
    return (
      <div
        className={rowClassName("trace-leaf")}
        ref={(element) => {
          rowRef.current = element;
        }}
      >
        {content}
      </div>
    );
  }

  return (
    <details className="trace-node" open={open} onToggle={(event) => setOpen(event.currentTarget.open)}>
      <summary
        className={rowClassName("")}
        ref={(element) => {
          rowRef.current = element;
        }}
      >
        {content}
      </summary>
      <div className="trace-children">
        {props.searchNode.children.map((child, index) => (
          <TraceNodeView
            addressLabels={props.addressLabels}
            explorerBaseUrl={props.explorerBaseUrl}
            key={`${child.node.raw}-${index}`}
            searchNode={child}
            expandMode={props.expandMode}
            expandDepth={props.expandDepth}
            depth={props.depth + 1}
            hasSearch={props.hasSearch}
            highlightTerms={props.highlightTerms}
            searchMatchIndex={props.searchMatchIndex}
          />
        ))}
      </div>
    </details>
  );
}

function traceRowClassName(matches: boolean, active: boolean): (base: string) => string {
  return (base: string) => [base, matches ? "trace-search-match" : "", active ? "trace-search-active" : ""].filter(Boolean).join(" ");
}

function traceLabel(node: TraceNode, addressLabels: AddressLabels, explorerBaseUrl: string, highlightTerms: string[]) {
  if (node.kind === "call") {
    const addressRef = resolveAddressReference(node.target, addressLabels);
    const suffix = (
      <>
        ::{highlightSearchText(node.function ?? "call", highlightTerms)}
        {node.arguments ? (
          <>
            (<TraceArguments addressLabels={addressLabels} explorerBaseUrl={explorerBaseUrl} highlightTerms={highlightTerms} value={node.arguments} />)
          </>
        ) : null}
      </>
    );
    if (addressRef) {
      return (
        <>
          <AddressReference
            address={addressRef.address}
            addressLabels={addressLabels}
            displayLabel={addressRef.label}
            explorerBaseUrl={explorerBaseUrl}
            highlightTerms={highlightTerms}
          />
          {suffix}
        </>
      );
    }
    const target = resolveLabelAlias(node.target ?? "unknown", addressLabels);
    return (
      <>
        {looksLikeTraceLabel(target) ? (
          <span className="address-reference-text">{highlightSearchText(target, highlightTerms)}</span>
        ) : (
          highlightSearchText(target, highlightTerms)
        )}
        {suffix}
      </>
    );
  }
  if (node.kind === "event") {
    return (
      <TraceArguments
        addressLabels={addressLabels}
        explorerBaseUrl={explorerBaseUrl}
        highlightTerms={highlightTerms}
        value={withoutEmitPrefix(node.value ?? node.raw)}
      />
    );
  }
  return <TraceArguments addressLabels={addressLabels} explorerBaseUrl={explorerBaseUrl} highlightTerms={highlightTerms} value={resultDisplayValue(node)} />;
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

function traceSearchTerms(query: string, addressLabels: AddressLabels): string[] {
  const trimmed = query.trim().toLowerCase();
  if (!trimmed) {
    return [];
  }

  const terms = new Set([trimmed]);
  if (isAddress(trimmed)) {
    const label = addressLabels.byAddress.get(trimmed);
    if (label) {
      terms.add(label.toLowerCase());
    }
  }

  const address = addressLabels.byLabel.get(trimmed);
  if (address) {
    terms.add(address.toLowerCase());
  }

  return Array.from(terms);
}

function buildSearchTree(nodes: TraceNode[], terms: string[], addressLabels: AddressLabels): { nodes: SearchNode[]; matchCount: number } {
  let matchCount = 0;
  const searchNodes = nodes.map((node) => visitSearchNode(node));
  return { nodes: searchNodes, matchCount };

  function visitSearchNode(node: TraceNode): SearchNode {
    const text = traceSearchText(node, addressLabels);
    const selfMatches = terms.length > 0 && terms.some((term) => text.includes(term));
    const matchIndex = selfMatches ? matchCount : null;
    if (selfMatches) {
      matchCount += 1;
    }
    const children = (node.children ?? []).map((child) => visitSearchNode(child));
    return {
      children,
      matchIndex,
      node,
      selfMatches,
      subtreeMatches: selfMatches || children.some((child) => child.subtreeMatches)
    };
  }
}

function traceSearchText(node: TraceNode, addressLabels: AddressLabels): string {
  const addressRef = node.kind === "call" ? resolveAddressReference(node.target, addressLabels) : undefined;
  const targetLabel = addressRef?.label ?? resolveLabelAlias(node.target ?? "", addressLabels);
  return [
    node.kind,
    node.callType,
    node.target,
    targetLabel,
    addressRef?.address,
    node.function,
    node.arguments,
    node.value,
    node.raw,
    node.resultType,
    node.gas?.toString()
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
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
