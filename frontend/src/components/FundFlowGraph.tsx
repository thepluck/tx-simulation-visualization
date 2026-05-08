import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import { displayAddress, labelForAddress, type AddressLabels } from "../labels";
import { formatFlowAmount, shortAddress } from "../format";
import { explorerAddressUrl } from "../explorer";
import type { BalanceAnalysis, ERC20Transfer } from "../types";
import AddressReference from "./AddressReference";
import TokenLogo from "./TokenLogo";

type TokenMetadata = {
  logoUrl?: string;
  symbol?: string;
};

type GraphNode = {
  id: string;
  role: string;
  width: number;
  x: number;
  y: number;
};

type GraphEdge = {
  transfer: ERC20Transfer;
  from: GraphNode;
  to: GraphNode;
};

type GraphModel = {
  edges: GraphEdge[];
  height: number;
  nodes: GraphNode[];
  width: number;
};

export default function FundFlowGraph(props: {
  addressLabels: AddressLabels;
  analysis?: BalanceAnalysis;
  explorerBaseUrl: string;
  transfers: ERC20Transfer[];
}) {
  const graphRef = useRef<HTMLDivElement | null>(null);
  const containerWidth = useElementWidth(graphRef);
  const graph = useMemo(() => buildGraph(props.transfers, containerWidth), [containerWidth, props.transfers]);
  const tokenMetadata = useMemo(() => buildTokenMetadata(props.transfers, props.analysis), [props.analysis, props.transfers]);

  if (props.transfers.length === 0) {
    return <div className="flow-graph empty-state">No transfers</div>;
  }

  return (
    <div className="flow-graph" ref={graphRef}>
      <svg className="flow-svg" style={{ height: graph.height }} viewBox={`0 0 ${graph.width} ${graph.height}`} role="img" aria-label="Fund flow graph">
        <defs>
          <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" />
          </marker>
        </defs>
        {graph.edges.map((edge, index) => (
          <GraphEdgeView edge={edge} index={index} key={`${edge.from.id}-${edge.to.id}-${edge.transfer.token}-${index}`} metadata={tokenMetadata} />
        ))}
        {graph.nodes.map((node) => (
          <GraphNodeView addressLabels={props.addressLabels} explorerBaseUrl={props.explorerBaseUrl} key={node.id} node={node} />
        ))}
      </svg>
      <EdgeList addressLabels={props.addressLabels} explorerBaseUrl={props.explorerBaseUrl} metadata={tokenMetadata} transfers={props.transfers} />
    </div>
  );
}

function GraphEdgeView(props: { edge: GraphEdge; index: number; metadata: Map<string, TokenMetadata> }) {
  const { edge, index, metadata } = props;
  const centerX = (edge.from.x + edge.to.x) / 2;
  const centerY = (edge.from.y + edge.to.y) / 2;
  const token = tokenInfo(edge.transfer.token, metadata);
  const amount = formatFlowAmount(edge.transfer.normalizedAmount || edge.transfer.amount);
  const labelY = centerY - 10;
  const logoX = centerX - 78;
  const amountX = token.logoUrl ? logoX + 24 : centerX - 78;
  return (
    <g className="edge">
      <title>
        {amount} {token.symbol} [{index + 1}]
      </title>
      <path d={edgePath(edge)} />
      {token.logoUrl && <image className="edge-token-logo" height="18" href={token.logoUrl} width="18" x={logoX} y={labelY - 9} />}
      <text className="edge-label" x={amountX} y={labelY}>
        {amount} {token.symbol} <tspan className="edge-index-label">[{index + 1}]</tspan>
      </text>
    </g>
  );
}

function buildGraph(transfers: ERC20Transfer[], containerWidth: number): GraphModel {
  const userIds = Array.from(new Set(transfers.flatMap((transfer) => [transfer.from, transfer.to])));
  const outgoing = new Set(transfers.map((transfer) => transfer.from));
  const incoming = new Set(transfers.map((transfer) => transfer.to));
  const width = Math.max(560, Math.round(containerWidth || 1100));
  const height = Math.max(460, userIds.length * 92 + 80);

  const left = userIds.filter((id) => outgoing.has(id));
  const right = userIds.filter((id) => !outgoing.has(id) || incoming.has(id));
  const center = userIds.filter((id) => !left.includes(id) && !right.includes(id));
  const nodeWidth = graphNodeWidth(Math.max(left.length, right.length, center.length), userIds.length, width);
  const horizontalPadding = width < 720 ? 28 : 64;
  const leftX = horizontalPadding + nodeWidth / 2;
  const rightX = width - horizontalPadding - nodeWidth / 2;
  const centerX = width / 2;
  const placed = new Map<string, GraphNode>();
  placeColumn(left, leftX, "sender", nodeWidth, placed);
  placeColumn(right, rightX, "recipient", nodeWidth, placed);
  placeColumn(center, centerX, "account", nodeWidth, placed);

  const nodes = Array.from(placed.values());
  const edges = transfers.map((transfer) => ({
    transfer,
    from: placed.get(transfer.from)!,
    to: placed.get(transfer.to)!
  }));
  return { width, height, nodes, edges };
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

function placeColumn(ids: string[], x: number, role: string, width: number, placed: Map<string, GraphNode>) {
  ids.forEach((id, index) => {
    if (!placed.has(id)) {
      placed.set(id, { id, role, width, x, y: 90 + index * 92 });
    }
  });
}

function edgePath(edge: GraphEdge): string {
  const startX = edge.from.x + edge.from.width / 2 + 6;
  const endX = edge.to.x - edge.to.width / 2 - 6;
  if (edge.from.id === edge.to.id) {
    return `M ${edge.from.x} ${edge.from.y - 24} C ${edge.from.x + 220} ${edge.from.y - 120}, ${edge.to.x + 220} ${edge.to.y + 120}, ${edge.to.x} ${edge.to.y + 24}`;
  }
  return `M ${startX} ${edge.from.y} C ${edge.from.x + 250} ${edge.from.y}, ${edge.to.x - 250} ${edge.to.y}, ${endX} ${edge.to.y}`;
}

function GraphNodeView(props: { addressLabels: AddressLabels; explorerBaseUrl: string; node: GraphNode }) {
  const label = labelForAddress(props.node.id, props.addressLabels);
  const primary = label || displayAddress(props.node.id, props.addressLabels);
  const address = truncateAddress(props.node.id);
  const displayLabel = truncateMiddle(primary, Math.max(8, Math.floor(props.node.width / 9)));
  const href = explorerAddressUrl(props.explorerBaseUrl, props.node.id);
  const content = (
    <>
      <title>{label ? `${label}\n${props.node.id}` : props.node.id}</title>
      <rect width={props.node.width} height="48" rx="7" />
      <text x={props.node.width / 2} y={label ? "18" : "24"}>
        {displayLabel}
      </text>
      {label && (
        <text x={props.node.width / 2} y="34" className="node-address">
          {address}
        </text>
      )}
    </>
  );
  return (
    <g className="node" transform={`translate(${props.node.x - props.node.width / 2}, ${props.node.y - 24})`}>
      {href ? (
        <a href={href} rel="noreferrer" target="_blank">
          {content}
        </a>
      ) : (
        content
      )}
    </g>
  );
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
