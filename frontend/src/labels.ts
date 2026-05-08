import { shortAddress } from "./format";
import type { ERC20Transfer, LabelOverride, SimulateResponse, TokenBalanceChange, TraceNode } from "./types";

export type AddressLabels = {
  byAddress: Map<string, string>;
  byLabel: Map<string, string>;
};
export type AddressReference = {
  address: string;
  label?: string;
};

const traceLabeledAddressPattern = /([^,()[\]\n]+?):\s*\[(0x[0-9a-fA-F]{40})\]/g;

export function buildAddressLabels(overrides: LabelOverride[], sender?: string, response?: SimulateResponse | null): AddressLabels {
  const labels: AddressLabels = {
    byAddress: new Map(),
    byLabel: new Map()
  };
  for (const override of overrides) {
    setAddressLabel(labels, override.account, override.label);
  }
  const senderAddress = (sender ?? "").trim().toLowerCase();
  if (isAddress(senderAddress) && !labels.byAddress.has(senderAddress)) {
    setAddressLabel(labels, senderAddress, "Sender");
  }
  addTraceLabels(labels, response);
  addResponseLabels(labels, response);
  return labels;
}

export function displayAddress(address: string, labels: AddressLabels, size = 8): string {
  return labels.byAddress.get(address.toLowerCase()) || shortAddress(address, size);
}

export function labelForAddress(address: string, labels: AddressLabels): string | undefined {
  return labels.byAddress.get(address.toLowerCase());
}

export function resolveAddressReference(value: string | undefined, labels: AddressLabels): AddressReference | undefined {
  const trimmed = (value ?? "").trim();
  if (!trimmed) {
    return undefined;
  }
  const labeled = parseLabeledAddress(trimmed);
  if (labeled) {
    return {
      address: labeled.address,
      label: resolveLabelAlias(labeled.label ?? "", labels)
    };
  }
  if (isAddress(trimmed)) {
    return {
      address: trimmed,
      label: labels.byAddress.get(trimmed.toLowerCase())
    };
  }

  const labelAddress = labels.byLabel.get(normalizeLabel(trimmed));
  if (labelAddress) {
    return {
      address: labelAddress,
      label: labels.byAddress.get(labelAddress.toLowerCase()) ?? resolveLabelAlias(trimmed, labels)
    };
  }
  return undefined;
}

export function resolveLabelAlias(value: string, labels: AddressLabels): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return value;
  }
  const labelAddress = labels.byLabel.get(normalizeLabel(trimmed));
  if (labelAddress) {
    return labels.byAddress.get(labelAddress.toLowerCase()) ?? trimmed;
  }
  return trimmed;
}

export function replaceLabelAliases(value: string, labels: AddressLabels): string {
  let result = value;
  const aliases = Array.from(labels.byLabel.entries()).sort((left, right) => right[0].length - left[0].length);
  for (const [source, address] of aliases) {
    const target = labels.byAddress.get(address.toLowerCase());
    if (!source || !target || normalizeLabel(target) === source) {
      continue;
    }
    result = result.replace(new RegExp(`(^|[^A-Za-z0-9_$])(${escapeRegExp(source)})(?=$|[^A-Za-z0-9_$])`, "gi"), (_match, prefix) => `${prefix}${target}`);
  }
  return result;
}

export function isAddress(value: string): boolean {
  return /^0x[0-9a-fA-F]{40}$/.test(value.trim());
}

export function parseLabeledAddress(value: string): AddressReference | undefined {
  const match = value.trim().match(/^(.+?):\s*\[(0x[0-9a-fA-F]{40})\]$/);
  if (!match) {
    return undefined;
  }
  const label = lastFoundryLabelSegment(match[1]);
  if (!looksLikeTraceLabel(label)) {
    return undefined;
  }
  return {
    address: match[2],
    label
  };
}

export function looksLikeTraceLabel(value: string): boolean {
  return /^[A-Z]/.test(value.trim());
}

function addResponseLabels(labels: AddressLabels, response: SimulateResponse | null | undefined) {
  for (const transfer of response?.erc20Transfers ?? []) {
    addTokenLabel(labels, transfer);
  }
  for (const change of response?.balanceAnalysis?.changes ?? []) {
    addTokenLabel(labels, change);
  }
}

function addTraceLabels(labels: AddressLabels, response: SimulateResponse | null | undefined) {
  for (const node of response?.structuredTrace ?? []) {
    addTraceNodeLabels(labels, node);
  }
}

function addTraceNodeLabels(labels: AddressLabels, node: TraceNode) {
  addTraceTextLabels(labels, node.raw);
  addTraceTextLabels(labels, node.target);
  addTraceTextLabels(labels, node.arguments);
  addTraceTextLabels(labels, node.value);
  for (const child of node.children ?? []) {
    addTraceNodeLabels(labels, child);
  }
}

function addTraceTextLabels(labels: AddressLabels, value: string | undefined) {
  if (!value) {
    return;
  }
  for (const match of value.matchAll(traceLabeledAddressPattern)) {
    const label = lastFoundryLabelSegment(match[1]);
    if (looksLikeTraceLabel(label)) {
      addAddressLabel(labels, match[2], label, false);
    }
  }
}

function addTokenLabel(labels: AddressLabels, item: ERC20Transfer | TokenBalanceChange) {
  const label = item.symbol?.trim();
  if (label) {
    setAddressLabel(labels, item.token, label);
  }
}

function setAddressLabel(labels: AddressLabels, address: string, label: string) {
  addAddressLabel(labels, address, label, true);
}

function addAddressLabel(labels: AddressLabels, address: string, label: string, prefer: boolean) {
  const canonicalAddress = address.trim();
  const normalizedAddress = canonicalAddress.toLowerCase();
  const trimmedLabel = label.trim();
  if (!isAddress(normalizedAddress) || !trimmedLabel) {
    return;
  }

  if (prefer || !labels.byAddress.has(normalizedAddress)) {
    const previousLabel = labels.byAddress.get(normalizedAddress);
    if (previousLabel && normalizeLabel(previousLabel) !== normalizeLabel(trimmedLabel)) {
      labels.byLabel.set(normalizeLabel(previousLabel), canonicalAddress);
    }
    labels.byAddress.set(normalizedAddress, trimmedLabel);
  }
  labels.byLabel.set(normalizeLabel(trimmedLabel), canonicalAddress);
}

function normalizeLabel(value: string): string {
  return value.trim().toLowerCase();
}

function lastFoundryLabelSegment(value: string): string {
  return (
    value
      .split(":")
      .map((part) => part.trim())
      .filter(Boolean)
      .at(-1) ?? value.trim()
  );
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
