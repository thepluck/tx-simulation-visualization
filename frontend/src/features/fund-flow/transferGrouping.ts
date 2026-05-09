import type { BalanceAnalysis, ERC20Transfer } from "../../api/types";
import { formatFlowAmount, shortAddress } from "../../lib/format";
import type { GraphTransfer, TokenMetadata } from "./types";

export function buildTokenMetadata(transfers: ERC20Transfer[], analysis: BalanceAnalysis | undefined): Map<string, TokenMetadata> {
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

export function groupGraphTransfers(transfers: ERC20Transfer[]): GraphTransfer[] {
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

export function estimateEdgeLabelWidth(transfer: GraphTransfer, metadata: Map<string, TokenMetadata>): number {
  const token = tokenInfo(transfer.token, metadata);
  const text = `${formatFlowAmount(transfer.amount)} ${token.symbol} ${formatIndexLabel(transfer.indices)}`;
  return Math.min(260, Math.max(82, text.length * 8 + 34));
}

export function tokenInfo(token: string, metadata: Map<string, TokenMetadata>): Required<TokenMetadata> {
  const found = metadata.get(token.toLowerCase());
  return {
    symbol: found?.symbol || shortAddress(token, 4),
    logoUrl: found?.logoUrl || ""
  };
}

export function formatIndexLabel(indices: number[]): string {
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

function formatIndexRange(start: number, end: number): string {
  return start === end ? String(start) : `${start}-${end}`;
}
