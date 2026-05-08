import { shortAddress } from "./format";
import type { LabelOverride } from "./types";

export type AddressLabels = Map<string, string>;
export type AddressReference = {
  address: string;
  label?: string;
};

export function buildAddressLabels(overrides: LabelOverride[]): AddressLabels {
  const labels: AddressLabels = new Map();
  for (const override of overrides) {
    const account = override.account.trim().toLowerCase();
    const label = override.label.trim();
    if (account && label) {
      labels.set(account, label);
    }
  }
  return labels;
}

export function displayAddress(address: string, labels: AddressLabels, size = 8): string {
  return labels.get(address.toLowerCase()) || shortAddress(address, size);
}

export function labelForAddress(address: string, labels: AddressLabels): string | undefined {
  return labels.get(address.toLowerCase());
}

export function resolveAddressReference(value: string | undefined, labels: AddressLabels): AddressReference | undefined {
  const trimmed = (value ?? "").trim();
  if (!trimmed) {
    return undefined;
  }
  if (isAddress(trimmed)) {
    return {
      address: trimmed,
      label: labels.get(trimmed.toLowerCase())
    };
  }

  for (const [address, label] of labels) {
    if (label === trimmed) {
      return { address, label };
    }
  }
  return undefined;
}

export function isAddress(value: string): boolean {
  return /^0x[0-9a-fA-F]{40}$/.test(value.trim());
}
