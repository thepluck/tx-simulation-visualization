import { useState } from "react";
import { type AddressLabels, isAddress, looksLikeTraceLabel, replaceLabelAliases, resolveAddressReference } from "../../lib/labels";
import AddressReference from "../../components/AddressReference";
import { highlightSearchText } from "../../components/SearchHighlight";

type TraceArgumentsProps = {
  addressLabels: AddressLabels;
  explorerBaseUrl: string;
  highlightQuery?: string;
  value: string;
};

type ArgumentPiece =
  | {
      kind: "text";
      value: string;
    }
  | {
      kind: "bytes";
      value: string;
    }
  | {
      kind: "address";
      value: string;
    }
  | {
      address: string;
      kind: "labeledAddress";
      label: string;
    };

const longHexPattern = /0x[0-9a-fA-F]{48,}/g;
const addressPattern = /0x[0-9a-fA-F]{40}/g;
const labeledAddressPattern = /([^,()[\]\n]+?):\s*\[(0x[0-9a-fA-F]{40})\]/g;

export default function TraceArguments(props: TraceArgumentsProps) {
  return (
    <>
      {splitArgumentPieces(props.value, props.addressLabels).map((piece, index) => {
        if (piece.kind === "bytes") {
          return <CollapsibleBytes highlightQuery={props.highlightQuery} key={`${piece.value}-${index}`} value={piece.value} />;
        }
        if (piece.kind === "address") {
          return (
            <AddressReference
              address={piece.value}
              addressLabels={props.addressLabels}
              explorerBaseUrl={props.explorerBaseUrl}
              highlightQuery={props.highlightQuery}
              key={`${piece.value}-${index}`}
            />
          );
        }
        if (piece.kind === "labeledAddress") {
          return (
            <AddressReference
              address={piece.address}
              addressLabels={props.addressLabels}
              displayLabel={piece.label}
              explorerBaseUrl={props.explorerBaseUrl}
              highlightQuery={props.highlightQuery}
              key={`${piece.label}-${piece.address}-${index}`}
            />
          );
        }
        return highlightSearchText(replaceLabelAliases(piece.value, props.addressLabels), props.highlightQuery ?? "");
      })}
    </>
  );
}

function CollapsibleBytes(props: { highlightQuery?: string; value: string }) {
  const [expanded, setExpanded] = useState(false);
  const display = expanded ? props.value : shortenBytes(props.value);

  return (
    <button
      aria-label={expanded ? "Collapse bytes argument" : "Expand bytes argument"}
      className={`trace-bytes-toggle ${expanded ? "expanded" : ""}`}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        setExpanded((current) => !current);
      }}
      type="button"
    >
      {highlightSearchText(display, props.highlightQuery ?? "")}
    </button>
  );
}

function splitArgumentPieces(value: string, labels: AddressLabels): ArgumentPiece[] {
  const pieces: ArgumentPiece[] = [];
  let cursor = 0;

  for (const match of value.matchAll(longHexPattern)) {
    const start = match.index ?? 0;
    if (start > cursor) {
      pieces.push(...splitAddressPieces(value.slice(cursor, start), labels));
    }
    pieces.push({ kind: "bytes", value: match[0] });
    cursor = start + match[0].length;
  }

  if (cursor < value.length) {
    pieces.push(...splitAddressPieces(value.slice(cursor), labels));
  }
  return pieces.length > 0 ? pieces : [{ kind: "text", value }];
}

function splitAddressPieces(value: string, labels: AddressLabels): ArgumentPiece[] {
  const pieces: ArgumentPiece[] = [];
  let cursor = 0;

  for (const match of value.matchAll(labeledAddressPattern)) {
    const start = match.index ?? 0;
    if (start > cursor) {
      pieces.push(...splitBareAddressPieces(value.slice(cursor, start)));
    }
    const labeled = splitFoundryLabelPrefix(match[1], match[2], labels);
    if (labeled.prefix) {
      pieces.push({ kind: "text", value: labeled.prefix });
    }
    pieces.push(labeled.label ? { address: match[2], kind: "labeledAddress", label: labeled.label } : { kind: "address", value: match[2] });
    cursor = start + match[0].length;
  }

  if (cursor < value.length) {
    pieces.push(...splitBareAddressPieces(value.slice(cursor)));
  }
  return pieces;
}

function splitBareAddressPieces(value: string): ArgumentPiece[] {
  const pieces: ArgumentPiece[] = [];
  let cursor = 0;

  for (const match of value.matchAll(addressPattern)) {
    const start = match.index ?? 0;
    if (start > cursor) {
      pieces.push({ kind: "text", value: value.slice(cursor, start) });
    }
    const address = match[0];
    pieces.push(isAddress(address) ? { kind: "address", value: address } : { kind: "text", value: address });
    cursor = start + address.length;
  }

  if (cursor < value.length) {
    pieces.push({ kind: "text", value: value.slice(cursor) });
  }
  return pieces;
}

function splitFoundryLabelPrefix(value: string, address: string, labels: AddressLabels): { label?: string; prefix: string } {
  const leadingSpace = value.match(/^\s*/)?.[0] ?? "";
  const parts = value
    .split(":")
    .map((part) => part.trim())
    .filter(Boolean);
  if (parts.length <= 1) {
    const candidate = parts[0] || value.trim();
    const resolved = resolveAddressReference(candidate, labels);
    if (resolved?.address.toLowerCase() === address.toLowerCase() || looksLikeTraceLabel(candidate)) {
      return { label: candidate, prefix: leadingSpace };
    }
    return { prefix: `${leadingSpace}${candidate}: ` };
  }
  const label = parts[parts.length - 1];
  if (!looksLikeTraceLabel(label)) {
    return { prefix: `${leadingSpace}${parts.join(": ")}: ` };
  }
  return {
    label,
    prefix: `${leadingSpace}${parts.slice(0, -1).join(": ")}: `
  };
}

function shortenBytes(value: string): string {
  if (value.length <= 22) {
    return value;
  }
  return `${value.slice(0, 10)}...${value.slice(-8)}`;
}
