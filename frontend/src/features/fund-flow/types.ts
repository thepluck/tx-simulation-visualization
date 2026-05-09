import type { Point } from "@dagrejs/dagre";
import type { Edge, Node } from "@xyflow/react";
import type { ERC20Transfer } from "../../api/types";
import type { AddressLabels } from "../../lib/labels";

export type TokenMetadata = {
  logoUrl?: string;
  symbol?: string;
};

export type FlowNodeData = {
  address: string;
  addressLabels: AddressLabels;
  explorerBaseUrl: string;
  width: number;
};

export type FlowEdgeData = {
  amount: string;
  indices: number[];
  labelX?: number;
  labelY?: number;
  metadata: Map<string, TokenMetadata>;
  points: Point[];
  token: string;
};

export type FlowNode = Node<FlowNodeData, "address">;
export type FlowEdge = Edge<FlowEdgeData, "transfer">;

export type GraphTransfer = {
  amount: string;
  from: string;
  id: string;
  indices: number[];
  to: string;
  token: string;
  transfers: ERC20Transfer[];
};

export const nodeHeight = 48;
