export function explorerAddressUrl(explorerBaseUrl: string, address: string): string | undefined {
  const baseUrl = explorerBaseUrl.trim().replace(/\/+$/, "");
  if (!baseUrl) {
    return undefined;
  }
  return `${baseUrl}/address/${address}`;
}

export function explorerForChain(explorerUrls: Record<string, string>, chain: string): string {
  return explorerUrls[chain] ?? explorerUrls[chain.trim().toLowerCase()] ?? "";
}
