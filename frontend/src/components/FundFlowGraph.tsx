import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import {
  Background,
  BaseEdge,
  Controls,
  EdgeLabelRenderer,
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
  index: number;
  metadata: Map<string, TokenMetadata>;
  transfer: ERC20Transfer;
};

type FlowNode = Node<FlowNodeData, "address">;
type FlowEdge = Edge<FlowEdgeData, "transfer">;

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
      {href ? (
        <a className="flow-node-link" href={href} rel="noreferrer" target="_blank">
          {content}
        </a>
      ) : (
        content
      )}
    </div>
  );
}

function TransferEdge(props: EdgeProps<FlowEdge>) {
  const [path, labelX, labelY] = getBezierPath(props);
  if (!props.data) {
    return <BaseEdge id={props.id} markerEnd={props.markerEnd} path={path} />;
  }

  const token = tokenInfo(props.data.transfer.token, props.data.metadata);
  const amount = formatFlowAmount(props.data.transfer.normalizedAmount || props.data.transfer.amount);
  return (
    <>
      <BaseEdge id={props.id} markerEnd={props.markerEnd} path={path} />
      <EdgeLabelRenderer>
        <div
          className="edge-label"
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`
          }}
          title={`${amount} ${token.symbol} [${props.data.index + 1}]`}
        >
          <TokenLogo logoUrl={token.logoUrl} symbol={token.symbol} />
          <span>
            {amount} {token.symbol}
          </span>
          <span className="edge-index-label">[{props.data.index + 1}]</span>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

function buildGraph(
  transfers: ERC20Transfer[],
  metadata: Map<string, TokenMetadata>,
  addressLabels: AddressLabels,
  explorerBaseUrl: string,
  containerWidth: number
): { edges: FlowEdge[]; height: number; nodes: FlowNode[] } {
  const userIds = Array.from(new Set(transfers.flatMap((transfer) => [transfer.from, transfer.to])));
  const outgoing = new Set(transfers.map((transfer) => transfer.from));
  const incoming = new Set(transfers.map((transfer) => transfer.to));
  const width = Math.max(560, Math.round(containerWidth || 1100));
  const height = Math.max(460, userIds.length * 92 + 120);
  const left = userIds.filter((id) => outgoing.has(id));
  const right = userIds.filter((id) => !outgoing.has(id) || incoming.has(id));
  const center = userIds.filter((id) => !left.includes(id) && !right.includes(id));
  const nodeWidth = graphNodeWidth(Math.max(left.length, right.length, center.length), userIds.length, width);
  const horizontalPadding = width < 720 ? 28 : 64;
  const leftX = horizontalPadding;
  const rightX = width - horizontalPadding - nodeWidth;
  const centerX = width / 2 - nodeWidth / 2;
  const placed = new Map<string, FlowNode>();

  placeColumn(left, leftX, nodeWidth, addressLabels, explorerBaseUrl, placed);
  placeColumn(right, rightX, nodeWidth, addressLabels, explorerBaseUrl, placed);
  placeColumn(center, centerX, nodeWidth, addressLabels, explorerBaseUrl, placed);

  const edges: FlowEdge[] = transfers.map((transfer, index) => ({
    id: `${transfer.from}-${transfer.to}-${transfer.token}-${index}`,
    source: transfer.from,
    target: transfer.to,
    type: "transfer",
    data: { index, metadata, transfer },
    markerEnd: {
      type: MarkerType.ArrowClosed,
      color: "#8992a1"
    }
  }));

  return { height, nodes: Array.from(placed.values()), edges };
}

function graphNodeWidth(maxColumnSize: number, totalNodes: number, graphWidth: number): number {
  const density = Math.max(maxColumnSize, Math.ceil(totalNodes / 2));
  const maxForViewport = Math.max(120, Math.min(300, (graphWidth - 180) / 2));
  let width = 180;
  if (density <= 2) {
    width = 300;
  } else if (density <= 4) {
    width = 260;
  } else if (density <= 7) {
    width = 220;
  }
  return Math.round(Math.min(width, maxForViewport));
}

function placeColumn(
  ids: string[],
  x: number,
  width: number,
  addressLabels: AddressLabels,
  explorerBaseUrl: string,
  placed: Map<string, FlowNode>
) {
  ids.forEach((id, index) => {
    if (!placed.has(id)) {
      placed.set(id, {
        id,
        type: "address",
        data: { address: id, addressLabels, explorerBaseUrl, width },
        position: { x, y: 70 + index * 92 },
        sourcePosition: Position.Right,
        targetPosition: Position.Left
      });
    }
  });
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
