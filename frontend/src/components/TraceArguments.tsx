import { useState } from "react";

type TraceArgumentsProps = {
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
    };

const longHexPattern = /0x[0-9a-fA-F]{48,}/g;

export default function TraceArguments(props: TraceArgumentsProps) {
  return (
    <>
      {splitArgumentPieces(props.value).map((piece, index) =>
        piece.kind === "bytes" ? <CollapsibleBytes key={`${piece.value}-${index}`} value={piece.value} /> : piece.value
      )}
    </>
  );
}

function CollapsibleBytes(props: { value: string }) {
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
      {display}
    </button>
  );
}

function splitArgumentPieces(value: string): ArgumentPiece[] {
  const pieces: ArgumentPiece[] = [];
  let cursor = 0;

  for (const match of value.matchAll(longHexPattern)) {
    const start = match.index ?? 0;
    if (start > cursor) {
      pieces.push({ kind: "text", value: value.slice(cursor, start) });
    }
    pieces.push({ kind: "bytes", value: match[0] });
    cursor = start + match[0].length;
  }

  if (cursor < value.length) {
    pieces.push({ kind: "text", value: value.slice(cursor) });
  }
  return pieces.length > 0 ? pieces : [{ kind: "text", value }];
}

function shortenBytes(value: string): string {
  if (value.length <= 22) {
    return value;
  }
  return `${value.slice(0, 10)}...${value.slice(-8)}`;
}
