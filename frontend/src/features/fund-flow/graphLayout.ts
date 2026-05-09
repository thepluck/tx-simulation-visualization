import { Graph, layout as dagreLayout, type EdgeLabel } from "@dagrejs/dagre";
import { MarkerType, Position } from "@xyflow/react";
import type { ERC20Transfer } from "../../api/types";
import type { AddressLabels } from "../../lib/labels";
import { estimateEdgeLabelWidth, groupGraphTransfers } from "./transferGrouping";
import { nodeHeight, type FlowEdge, type FlowNode, type TokenMetadata } from "./types";

export function buildGraph(
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
