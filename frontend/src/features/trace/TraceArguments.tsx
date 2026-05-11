import { useState } from "react";
import type { MouseEvent } from "react";
import { createPortal } from "react-dom";
import { type AddressLabels, isAddress, replaceLabelAliases } from "../../lib/labels";
import AddressReference from "../../components/AddressReference";
import CopyIcon from "../../components/CopyIcon";
import { highlightSearchText } from "../../components/SearchHighlight";
import { blockFloatingCardEvent, stopFloatingCardEvent, useFloatingCard } from "../../components/useFloatingCard";

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
          return <BytesReference highlightTerms={props.highlightTerms} key={`${piece.value}-${index}`} value={piece.value} />;
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

function BytesReference(props: { highlightTerms?: string[]; value: string }) {
  const [copied, setCopied] = useState(false);
  const { cardRef, cardStyle, closeCard, isActive, keepOpenOnCardPointerDown, referenceRef, shouldCloseOnBlur, toggleCard } =
    useFloatingCard<HTMLSpanElement, HTMLSpanElement>();
  const display = shortenBytes(props.value);

  const copyBytes = async (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    try {
      await navigator.clipboard.writeText(props.value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  };

  const card = (
    <span
      aria-label="Bytes argument details"
      className="trace-bytes-card"
      onClick={blockFloatingCardEvent}
      onPointerDown={keepOpenOnCardPointerDown}
      ref={cardRef}
      role="dialog"
      style={cardStyle}
    >
      <code>{props.value}</code>
      <button
        aria-label="Copy bytes argument"
        className={`address-copy-button${copied ? " copied" : ""}`}
        onClick={copyBytes}
        onPointerDown={stopFloatingCardEvent}
        title={copied ? "Copied" : "Copy bytes"}
        type="button"
      >
        <CopyIcon />
      </button>
    </span>
  );

  return (
    <span
      className={`trace-bytes-reference ${isActive ? "active" : ""}`}
      onBlur={(event) => {
        if (shouldCloseOnBlur(event.currentTarget, event.relatedTarget)) {
          closeCard();
        }
      }}
      ref={referenceRef}
    >
      <button
        aria-expanded={isActive}
        aria-label={isActive ? "Hide bytes argument" : "Show bytes argument"}
        className="trace-bytes-toggle"
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          toggleCard();
        }}
        type="button"
      >
        {highlightSearchText(display, props.highlightTerms)}
      </button>
      {isActive ? createPortal(card, document.body) : null}
    </span>
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
