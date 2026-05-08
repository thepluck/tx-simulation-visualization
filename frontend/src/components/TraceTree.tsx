import { useEffect, useMemo, useState } from "react";
import type { ExpandMode } from "../form";
import { resolveAddressReference, type AddressLabels } from "../labels";
import type { TraceNode } from "../types";
import AddressReference from "./AddressReference";

type TraceTreeProps = {
  addressLabels: AddressLabels;
  expandMode: ExpandMode;
  explorerBaseUrl: string;
  nodes: TraceNode[];
  target: string;
};

export default function TraceTree(props: TraceTreeProps) {
  const visibleNodes = useMemo(() => mainCallTrace(props.nodes, props.target, props.addressLabels), [props.addressLabels, props.nodes, props.target]);

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
        />
      ))}
    </div>
  );
}

function TraceNodeView(props: { addressLabels: AddressLabels; explorerBaseUrl: string; node: TraceNode; expandMode: ExpandMode }) {
  const hasChildren = Boolean(props.node.children?.length);
  const [open, setOpen] = useState(true);

  useEffect(() => {
    if (props.expandMode === "expand") {
      setOpen(true);
    }
    if (props.expandMode === "collapse") {
      setOpen(false);
    }
  }, [props.expandMode]);

  const main = traceLabel(props.node, props.addressLabels, props.explorerBaseUrl);
  const meta = [props.node.callType, props.node.gas ? `${props.node.gas} gas` : ""].filter(Boolean).join(" | ");

  if (!hasChildren) {
    return (
      <div className="trace-leaf">
        <span className="trace-kind">{props.node.kind}</span>
        <span className="trace-main" title={props.node.raw}>
          {main}
        </span>
        <span className="trace-meta">{meta}</span>
      </div>
    );
  }

  return (
    <details className="trace-node" open={open} onToggle={(event) => setOpen(event.currentTarget.open)}>
      <summary>
        <span className="trace-kind">{props.node.kind}</span>
        <span className="trace-main" title={props.node.raw}>
          {main}
        </span>
        <span className="trace-meta">{meta}</span>
      </summary>
      <div className="trace-children">
        {props.node.children?.map((child, index) => (
          <TraceNodeView
            addressLabels={props.addressLabels}
            explorerBaseUrl={props.explorerBaseUrl}
            key={`${child.raw}-${index}`}
            node={child}
            expandMode={props.expandMode}
          />
        ))}
      </div>
    </details>
  );
}

function traceLabel(node: TraceNode, addressLabels: AddressLabels, explorerBaseUrl: string) {
  if (node.kind === "call") {
    const addressRef = resolveAddressReference(node.target, addressLabels);
    const suffix = `::${node.function ?? "call"}${node.arguments ? `(${node.arguments})` : ""}`;
    if (addressRef) {
      return (
        <>
          <AddressReference address={addressRef.address} addressLabels={addressLabels} explorerBaseUrl={explorerBaseUrl} />
          {suffix}
        </>
      );
    }
    return `${node.target ?? "unknown"}${suffix}`;
  }
  if (node.kind === "event") {
    return `emit ${node.value ?? node.raw}`;
  }
  return node.value || node.raw;
}

function mainCallTrace(nodes: TraceNode[], target: string, addressLabels: AddressLabels): TraceNode[] {
  const targetAddress = resolveAddressReference(target, addressLabels)?.address.toLowerCase() ?? target.trim().toLowerCase();
  if (!targetAddress) {
    return visibleTrace(nodes);
  }

  for (let index = nodes.length - 1; index >= 0; index -= 1) {
    const found = findLastMatchingCall(nodes[index], targetAddress, addressLabels);
    if (found) {
      return visibleTrace([found]);
    }
  }
  return visibleTrace(nodes);
}

function findLastMatchingCall(node: TraceNode, targetAddress: string, addressLabels: AddressLabels): TraceNode | undefined {
  for (let index = (node.children?.length ?? 0) - 1; index >= 0; index -= 1) {
    const found = findLastMatchingCall(node.children![index], targetAddress, addressLabels);
    if (found) {
      return found;
    }
  }

  if (node.kind === "call") {
    const callTarget = resolveAddressReference(node.target, addressLabels)?.address.toLowerCase() ?? node.target?.trim().toLowerCase();
    if (callTarget === targetAddress) {
      return node;
    }
  }
  return undefined;
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
  return (kind === "return" || kind === "stop" || kind === "result") && !node.value?.trim();
}
