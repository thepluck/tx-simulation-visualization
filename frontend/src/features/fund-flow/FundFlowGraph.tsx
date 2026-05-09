import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import type { Point } from "@dagrejs/dagre";
import {
  Background,
  BaseEdge,
  Controls,
  EdgeLabelRenderer,
  Handle,
  Position,
  ReactFlow,
  getBezierPath,
  type EdgeProps,
  type NodeProps
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { explorerAddressUrl } from "../../lib/explorer";
import { formatFlowAmount } from "../../lib/format";
import { displayAddress, labelForAddress, type AddressLabels } from "../../lib/labels";
import type { BalanceAnalysis, ERC20Transfer } from "../../api/types";
import AddressReference from "../../components/AddressReference";
import TokenLogo from "../../components/TokenLogo";
import { buildGraph } from "./graphLayout";
import { buildTokenMetadata, formatIndexLabel, tokenInfo } from "./transferGrouping";
import type { FlowEdge, FlowNode, TokenMetadata } from "./types";

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
