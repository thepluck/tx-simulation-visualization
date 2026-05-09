export function shortAddress(value: string, size = 6): string {
  if (!value) {
    return "";
  }
  if (!value.startsWith("0x") || value.length <= size * 2 + 2) {
    return value;
  }
  return `${value.slice(0, size + 2)}...${value.slice(-size)}`;
}

export function formatUSD(value: number | undefined): string {
  if (value === undefined || Number.isNaN(value)) {
    return "-";
  }
  const sign = value < 0 ? "-" : "";
  const abs = Math.abs(value);
  return `${sign}$${abs.toLocaleString(undefined, {
    maximumFractionDigits: abs >= 100 ? 2 : 6
  })}`;
}

export function formatSignedUSD(value: number | undefined): string {
  const formatted = formatUSD(value);
  if (value === undefined || Number.isNaN(value) || value <= 0) {
    return formatted;
  }
  return `+${formatted}`;
}

export function formatTokenAmount(value: string): string {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return value;
  }
  return numeric.toLocaleString(undefined, {
    maximumFractionDigits: Math.abs(numeric) >= 1 ? 6 : 12
  });
}

export function formatSignedTokenAmount(value: string): string {
  const formatted = formatTokenAmount(value);
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return formatted;
  }
  return `+${formatted}`;
}

export function formatFlowAmount(value: string): string {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return value;
  }
  if (numeric === 0) {
    return "0";
  }
  const abs = Math.abs(numeric);
  if (abs >= 1_000_000_000 || abs < 0.000001) {
    return numeric.toExponential(3).replace(/\.?0+e/, "e");
  }
  return numeric.toLocaleString(undefined, {
    maximumFractionDigits: abs >= 1 ? 6 : 8
  });
}

export function classForSignedNumber(value: string | number | undefined): string {
  const numeric = typeof value === "string" ? Number(value) : value;
  if (numeric === undefined || Number.isNaN(numeric) || numeric === 0) {
    return "";
  }
  return numeric > 0 ? "positive" : "negative";
}
