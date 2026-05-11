import { useState } from "react";
import { type AddressLabels, isAddress, replaceLabelAliases } from "../../lib/labels";
import AddressReference from "../../components/AddressReference";
import { highlightSearchText } from "../../components/SearchHighlight";

type TraceArgumentsProps = {
  addressLabels: AddressLabels;
  explorerBaseUrl: string;
  highlightTerms?: string[];
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
    };

const longHexPattern = /0x[0-9a-fA-F]{48,}/g;
const addressPattern = /0x[0-9a-fA-F]{40}/g;
const bracketedAddressPattern = /([^,()[\]\n]+?):\s*\[(0x[0-9a-fA-F]{40})\]/g;

export default function TraceArguments(props: TraceArgumentsProps) {
  return (
    <>
      {splitArgumentPieces(props.value).map((piece, index) => {
        if (piece.kind === "bytes") {
          return <CollapsibleBytes highlightTerms={props.highlightTerms} key={`${piece.value}-${index}`} value={piece.value} />;
        }
        if (piece.kind === "address") {
          return (
            <AddressReference
              address={piece.value}
              addressLabels={props.addressLabels}
              explorerBaseUrl={props.explorerBaseUrl}
              highlightTerms={props.highlightTerms}
              key={`${piece.value}-${index}`}
            />
          );
        }
        return highlightSearchText(replaceLabelAliases(piece.value, props.addressLabels), props.highlightTerms);
      })}
    </>
  );
}

function CollapsibleBytes(props: { highlightTerms?: string[]; value: string }) {
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
      {highlightSearchText(display, props.highlightTerms)}
    </button>
  );
}

function splitArgumentPieces(value: string): ArgumentPiece[] {
  const pieces: ArgumentPiece[] = [];
  let cursor = 0;

  for (const match of value.matchAll(longHexPattern)) {
    const start = match.index ?? 0;
    if (start > cursor) {
      pieces.push(...splitBareAddressPieces(value.slice(cursor, start)));
    }
    pieces.push({ kind: "bytes", value: match[0] });
    cursor = start + match[0].length;
  }

  if (cursor < value.length) {
    pieces.push(...splitBareAddressPieces(value.slice(cursor)));
  }
  return pieces.length > 0 ? pieces : [{ kind: "text", value }];
}

function splitBareAddressPieces(value: string): ArgumentPiece[] {
  const pieces: ArgumentPiece[] = [];
  let cursor = 0;

  for (const match of value.matchAll(bracketedAddressPattern)) {
    const start = match.index ?? 0;
    if (start > cursor) {
      pieces.push(...splitPlainAddressPieces(value.slice(cursor, start)));
    }
    const prefix = bracketedAddressPrefix(match[1]);
    if (prefix) {
      pieces.push({ kind: "text", value: prefix });
    }
    pieces.push({ kind: "address", value: match[2] });
    cursor = start + match[0].length;
  }

  if (cursor < value.length) {
    pieces.push(...splitPlainAddressPieces(value.slice(cursor)));
  }
  return pieces;
}

function splitPlainAddressPieces(value: string): ArgumentPiece[] {
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

function bracketedAddressPrefix(value: string): string {
  const leadingSpace = value.match(/^\s*/)?.[0] ?? "";
  const parts = value
    .split(":")
    .map((part) => part.trim())
    .filter(Boolean);
  if (parts.length >= 2) {
    return `${leadingSpace}${parts.slice(0, -1).join(": ")}: `;
  }
  return leadingSpace;
}

function shortenBytes(value: string): string {
  if (value.length <= 22) {
    return value;
  }
  return `${value.slice(0, 10)}...${value.slice(-8)}`;
}
