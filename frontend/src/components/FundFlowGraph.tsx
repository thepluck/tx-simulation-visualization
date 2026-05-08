import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import { Graph, layout as dagreLayout, type EdgeLabel, type Point } from "@dagrejs/dagre";
import {
  Background,
  BaseEdge,
  Controls,
  EdgeLabelRenderer,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  getBezierPath,
  type Edge,
  type EdgeProps,
  type Node,
  type NodeProps
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { explorerAddressUrl } from "../explorer";
import { formatFlowAmount, shortAddress } from "../format";
import { displayAddress, labelForAddress, type AddressLabels } from "../labels";
import type { BalanceAnalysis, ERC20Transfer } from "../types";
import AddressReference from "./AddressReference";
import TokenLogo from "./TokenLogo";

type TokenMetadata = {
  logoUrl?: string;
  symbol?: string;
};

type FlowNodeData = {
  address: string;
  addressLabels: AddressLabels;
  explorerBaseUrl: string;
  width: number;
};

type FlowEdgeData = {
  amount: string;
  indices: number[];
  labelX?: number;
  labelY?: number;
  metadata: Map<string, TokenMetadata>;
  points: Point[];
  token: string;
};

type FlowNode = Node<FlowNodeData, "address">;
type FlowEdge = Edge<FlowEdgeData, "transfer">;
type GraphTransfer = {
  amount: string;
  from: string;
  id: string;
  indices: number[];
  to: string;
  token: string;
  transfers: ERC20Transfer[];
};

const nodeHeight = 48;

const nodeTypes = {
  address: AddressNode
};

const edgeTypes = {
  transfer: TransferEdge
};

export default function FundFlowGraph(props: {
  addressLabels: AddressLabels;
  analysis?: BalanceAnalysis;
  explorerBaseUrl: string;
  transfers: ERC20Transfer[];
}) {
  const graphRef = useRef<HTMLDivElement | null>(null);
  const containerWidth = useElementWidth(graphRef);
  const tokenMetadata = useMemo(() => buildTokenMetadata(props.transfers, props.analysis), [props.analysis, props.transfers]);
  const graph = useMemo(
    () => buildGraph(props.transfers, tokenMetadata, props.addressLabels, props.explorerBaseUrl, containerWidth),
    [containerWidth, props.addressLabels, props.explorerBaseUrl, props.transfers, tokenMetadata]
  );

  if (props.transfers.length === 0) {
    return <div className="flow-graph empty-state">No transfers</div>;
  }

  return (
    <div className="flow-graph" ref={graphRef}>
      <div className="flow-canvas" role="img" aria-label="Fund flow graph" style={{ height: graph.height }}>
        <ReactFlow
          className="flow-svg"
          nodes={graph.nodes}
          edges={graph.edges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView
          fitViewOptions={{ padding: 0.16 }}
          minZoom={0.45}
          maxZoom={1.4}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          panOnDrag
          panOnScroll
          preventScrolling={false}
        >
          <Background gap={24} size={1} />
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>
      <EdgeList addressLabels={props.addressLabels} explorerBaseUrl={props.explorerBaseUrl} metadata={tokenMetadata} transfers={props.transfers} />
    </div>
  );
}

function AddressNode(props: NodeProps<FlowNode>) {
  const { address, addressLabels, explorerBaseUrl, width } = props.data;
  const label = labelForAddress(address, addressLabels);
  const primary = label || displayAddress(address, addressLabels);
  const addressText = truncateAddress(address);
  const displayLabel = truncateMiddle(primary, Math.max(8, Math.floor(width / 9)));
  const href = explorerAddressUrl(explorerBaseUrl, address);
  const content = (
    <>
      <span className="flow-node-label">{displayLabel}</span>
      {label && <span className="flow-node-address">{addressText}</span>}
    </>
  );
  return (
    <div className="flow-node" style={{ width }} title={label ? `${label}\n${address}` : address}>
      <Handle className="flow-node-handle" type="target" position={Position.Left} />
      {href ? (
        <a className="flow-node-link" href={href} rel="noreferrer" target="_blank">
          {content}
        </a>
      ) : (
        content
      )}
      <Handle className="flow-node-handle" type="source" position={Position.Right} />
    </div>
  );
}

function TransferEdge(props: EdgeProps<FlowEdge>) {
  const [fallbackPath, fallbackLabelX, fallbackLabelY] = getBezierPath(props);
  if (!props.data) {
    return <BaseEdge id={props.id} markerEnd={props.markerEnd} path={fallbackPath} />;
  }

  const path = props.data.points.length >= 2 ? pathFromPoints(props.data.points) : fallbackPath;
  const labelPoint = edgeLabelPoint(props.data.points);
  const labelX = props.data.labelX ?? labelPoint.x ?? fallbackLabelX;
  const labelY = props.data.labelY ?? labelPoint.y ?? fallbackLabelY;
  const token = tokenInfo(props.data.token, props.data.metadata);
  const amount = formatFlowAmount(props.data.amount);
  const indexLabel = formatIndexLabel(props.data.indices);
  return (
    <>
      <BaseEdge id={props.id} markerEnd={props.markerEnd} path={path} />
      <EdgeLabelRenderer>
        <div
          className="edge-label"
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`
          }}
          title={`${amount} ${token.symbol} ${indexLabel}`}
        >
          <TokenLogo logoUrl={token.logoUrl} symbol={token.symbol} />
          <span>
            {amount} {token.symbol}
          </span>
          <span className="edge-index-label">{indexLabel}</span>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

function pathFromPoints(points: Point[]): string {
  return points.map((point, index) => `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`).join(" ");
}

function edgeLabelPoint(points: Point[]): { x?: number; y?: number } {
  if (points.length === 0) {
    return {};
  }
  return points[Math.floor(points.length / 2)];
}

function buildGraph(
  transfers: ERC20Transfer[],
  metadata: Map<string, TokenMetadata>,
  addressLabels: AddressLabels,
  explorerBaseUrl: string,
  containerWidth: number
): { edges: FlowEdge[]; height: number; nodes: FlowNode[] } {
  const userIds = Array.from(new Set(transfers.flatMap((transfer) => [transfer.from, transfer.to])));
  const graphTransfers = groupGraphTransfers(transfers);
  const nodeWidth = graphNodeWidth(userIds.length, containerWidth);
  const dagreGraph = new Graph({ directed: true, multigraph: true });
  dagreGraph.setGraph({
    acyclicer: "greedy",
    edgesep: 18,
    marginx: 48,
    marginy: 48,
    nodesep: 58,
    rankdir: "LR",
    ranker: "network-simplex",
    ranksep: 150
  });
  dagreGraph.setDefaultEdgeLabel(() => ({}));

  for (const id of userIds) {
    dagreGraph.setNode(id, { width: nodeWidth, height: nodeHeight });
  }
  for (const transfer of graphTransfers) {
    dagreGraph.setEdge(
      transfer.from,
      transfer.to,
      {
        height: 30,
        labelpos: "c",
        weight: 1,
        width: estimateEdgeLabelWidth(transfer, metadata)
      },
      transfer.id
    );
  }
  dagreLayout(dagreGraph);

  const nodes: FlowNode[] = userIds.map((id) => {
    const point = dagreGraph.node(id) as { x?: number; y?: number } | undefined;
    return {
      id,
      type: "address",
      data: { address: id, addressLabels, explorerBaseUrl, width: nodeWidth },
      position: {
        x: Math.max(0, (point?.x ?? 0) - nodeWidth / 2),
        y: Math.max(0, (point?.y ?? 0) - nodeHeight / 2)
      },
      sourcePosition: Position.Right,
      targetPosition: Position.Left
    };
  });

  const edges: FlowEdge[] = graphTransfers.map((transfer) => {
    const route = dagreGraph.edge(transfer.from, transfer.to, transfer.id) as EdgeLabel | undefined;
    return {
      id: transfer.id,
      source: transfer.from,
      target: transfer.to,
      type: "transfer",
      data: {
        amount: transfer.amount,
        indices: transfer.indices,
        labelX: route?.x,
        labelY: route?.y,
        metadata,
        points: route?.points ?? [],
        token: transfer.token
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: "#8992a1"
      }
    };
  });
  const graphLabel = dagreGraph.graph() as { height?: number; width?: number } | undefined;
  const height = Math.max(460, Math.ceil((graphLabel?.height ?? 0) + 40));

  return { height, nodes, edges };
}

function groupGraphTransfers(transfers: ERC20Transfer[]): GraphTransfer[] {
  const groups = new Map<string, GraphTransfer>();
  for (const [index, transfer] of transfers.entries()) {
    const token = transfer.token.toLowerCase();
    const id = `${transfer.from}-${transfer.to}-${token}`;
    const current =
      groups.get(id) ??
      ({
        amount: "0",
        from: transfer.from,
        id,
        indices: [],
        to: transfer.to,
        token,
        transfers: []
      } satisfies GraphTransfer);
    current.indices.push(index + 1);
    current.transfers.push(transfer);
    groups.set(id, current);
  }

  for (const group of groups.values()) {
    group.amount = summedTransferAmount(group.transfers);
  }
  return Array.from(groups.values());
}

function estimateEdgeLabelWidth(transfer: GraphTransfer, metadata: Map<string, TokenMetadata>): number {
  const token = tokenInfo(transfer.token, metadata);
  const text = `${formatFlowAmount(transfer.amount)} ${token.symbol} ${formatIndexLabel(transfer.indices)}`;
  return Math.min(260, Math.max(82, text.length * 8 + 34));
}

function summedTransferAmount(transfers: ERC20Transfer[]): string {
  if (transfers.length === 1) {
    return transfers[0].normalizedAmount || transfers[0].amount;
  }
  const values = transfers.map((transfer) => transfer.normalizedAmount || transfer.amount);
  const decimalTotal = sumDecimalValues(values);
  if (decimalTotal !== undefined) {
    return decimalTotal;
  }
  try {
    return transfers.reduce((sum, transfer) => sum + BigInt(transfer.amount), 0n).toString();
  } catch {
    return values[0] ?? "0";
  }
}

function sumDecimalValues(values: string[]): string | undefined {
  const parts = values.map((value) => value.trim().match(/^([0-9]+)(?:\.([0-9]+))?$/));
  if (parts.some((part) => !part)) {
    return undefined;
  }
  const fractionLength = Math.max(...parts.map((part) => part?.[2]?.length ?? 0));
  const scale = 10n ** BigInt(fractionLength);
  const total = parts.reduce((sum, part) => {
    const integer = BigInt(part?.[1] ?? "0") * scale;
    const fraction = (part?.[2] ?? "").padEnd(fractionLength, "0");
    return sum + integer + BigInt(fraction || "0");
  }, 0n);
  const integer = total / scale;
  const fraction = (total % scale).toString().padStart(fractionLength, "0").replace(/0+$/, "");
  return fraction ? `${integer}.${fraction}` : integer.toString();
}

function formatIndexLabel(indices: number[]): string {
  if (indices.length === 0) {
    return "[]";
  }
  const ranges: string[] = [];
  let start = indices[0];
  let previous = indices[0];
  for (const index of indices.slice(1)) {
    if (index === previous + 1) {
      previous = index;
      continue;
    }
    ranges.push(formatIndexRange(start, previous));
    start = index;
    previous = index;
  }
  ranges.push(formatIndexRange(start, previous));
  return `[${ranges.join(", ")}]`;
}

function formatIndexRange(start: number, end: number): string {
  return start === end ? String(start) : `${start}-${end}`;
}

function graphNodeWidth(totalNodes: number, containerWidth: number): number {
  const graphWidth = Math.max(560, Math.round(containerWidth || 1100));
  const maxForViewport = Math.max(150, Math.min(280, graphWidth / 3.4));
  let width = 210;
  if (totalNodes <= 4) {
    width = 300;
  } else if (totalNodes <= 8) {
    width = 260;
  } else if (totalNodes <= 14) {
    width = 220;
  }
  return Math.round(Math.min(width, maxForViewport));
}

function truncateMiddle(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }
  const keep = Math.max(3, Math.floor((maxLength - 3) / 2));
  return `${value.slice(0, keep)}...${value.slice(-keep)}`;
}

function truncateAddress(value: string): string {
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

function EdgeList(props: { addressLabels: AddressLabels; explorerBaseUrl: string; metadata: Map<string, TokenMetadata>; transfers: ERC20Transfer[] }) {
  return (
    <table className="data-table edge-table">
      <thead>
        <tr>
          <th>#</th>
          <th>From</th>
          <th>To</th>
          <th>Token</th>
          <th>Amount</th>
        </tr>
      </thead>
      <tbody>
        {props.transfers.map((transfer, index) => (
          <tr key={`${transfer.from}-${transfer.to}-${transfer.token}-${index}`}>
            <td>{index + 1}</td>
            <td>
              <AddressReference address={transfer.from} addressLabels={props.addressLabels} explorerBaseUrl={props.explorerBaseUrl} />
            </td>
            <td>
              <AddressReference address={transfer.to} addressLabels={props.addressLabels} explorerBaseUrl={props.explorerBaseUrl} />
            </td>
            <td title={transfer.token}>
              <FlowTokenCell metadata={props.metadata} token={transfer.token} />
            </td>
            <td title={transfer.amount}>{formatFlowAmount(transfer.normalizedAmount || transfer.amount)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function FlowTokenCell(props: { metadata: Map<string, TokenMetadata>; token: string }) {
  const token = tokenInfo(props.token, props.metadata);
  return (
    <div className="flow-token-cell">
      <TokenLogo logoUrl={token.logoUrl} symbol={token.symbol} />
      <span>{token.symbol}</span>
    </div>
  );
}

function tokenInfo(token: string, metadata: Map<string, TokenMetadata>): Required<TokenMetadata> {
  const found = metadata.get(token.toLowerCase());
  return {
    symbol: found?.symbol || shortAddress(token, 4),
    logoUrl: found?.logoUrl || ""
  };
}

function buildTokenMetadata(transfers: ERC20Transfer[], analysis: BalanceAnalysis | undefined): Map<string, TokenMetadata> {
  const metadata = new Map<string, TokenMetadata>();
  for (const transfer of transfers) {
    const token = transfer.token.toLowerCase();
    metadata.set(token, {
      symbol: transfer.symbol,
      logoUrl: transfer.logoUrl
    });
  }
  for (const change of analysis?.changes ?? []) {
    const token = change.token.toLowerCase();
    const current = metadata.get(token) ?? {};
    metadata.set(token, {
      symbol: current.symbol || change.symbol,
      logoUrl: current.logoUrl || change.logoUrl
    });
  }
  return metadata;
}

function useElementWidth(ref: RefObject<HTMLElement | null>): number {
  const [width, setWidth] = useState(1100);

  useEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }

    const updateWidth = () => {
      setWidth(Math.max(0, element.clientWidth));
    };
    updateWidth();

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      setWidth(Math.max(0, Math.floor(entry.contentRect.width)));
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, [ref]);

  return width;
}
