import { useState } from "react";
import type { MouseEvent } from "react";
import { createPortal } from "react-dom";
import CopyIcon from "./CopyIcon";
import { highlightSearchText } from "./SearchHighlight";
import { blockFloatingCardEvent, stopFloatingCardEvent, useFloatingCard } from "./useFloatingCard";

type FunctionReferenceProps = {
  highlightTerms?: string[];
  name: string;
  selector?: string;
  signature?: string;
};

type CopyTarget = "selector" | "signature";

export default function FunctionReference(props: FunctionReferenceProps) {
  const [copied, setCopied] = useState<CopyTarget | null>(null);
  const { cardRef, cardStyle, closeCard, isActive, keepOpenOnCardPointerDown, referenceRef, shouldCloseOnBlur, toggleCard } =
    useFloatingCard<HTMLButtonElement, HTMLSpanElement>();
  const signature = props.signature?.trim();
  const selector = props.selector?.trim();
  const hasDetails = Boolean(signature || selector);

  const copyValue = async (event: MouseEvent<HTMLButtonElement>, target: CopyTarget, value: string) => {
    event.preventDefault();
    event.stopPropagation();
    try {
      await navigator.clipboard.writeText(value);
      setCopied(target);
      window.setTimeout(() => setCopied(null), 1200);
    } catch {
      setCopied(null);
    }
  };

  if (!hasDetails) {
    return <span className="trace-function">{highlightSearchText(props.name, props.highlightTerms)}</span>;
  }

  const card = (
    <span
      aria-label="Function details"
      className="function-reference-card"
      onClick={blockFloatingCardEvent}
      onPointerDown={keepOpenOnCardPointerDown}
      ref={cardRef}
      role="dialog"
      style={cardStyle}
    >
      {signature && (
        <span className="function-reference-card-row">
          <span className="function-reference-card-label">Signature</span>
          <code>{commaWrappedText(signature)}</code>
          <button
            aria-label={`Copy function signature ${signature}`}
            className={`address-copy-button${copied === "signature" ? " copied" : ""}`}
            onClick={(event) => copyValue(event, "signature", signature)}
            onPointerDown={stopFloatingCardEvent}
            title={copied === "signature" ? "Copied" : "Copy signature"}
            type="button"
          >
            <CopyIcon />
          </button>
        </span>
      )}
      {selector && (
        <span className="function-reference-card-row">
          <span className="function-reference-card-label">Selector</span>
          <code>{selector}</code>
          <button
            aria-label={`Copy function selector ${selector}`}
            className={`address-copy-button${copied === "selector" ? " copied" : ""}`}
            onClick={(event) => copyValue(event, "selector", selector)}
            onPointerDown={stopFloatingCardEvent}
            title={copied === "selector" ? "Copied" : "Copy selector"}
            type="button"
          >
            <CopyIcon />
          </button>
        </span>
      )}
    </span>
  );

  return (
    <span
      className={`function-reference ${isActive ? "active" : ""}`}
      onBlur={(event) => {
        if (shouldCloseOnBlur(event.currentTarget, event.relatedTarget)) {
          closeCard();
        }
      }}
    >
      <button
        aria-expanded={isActive}
        aria-haspopup="dialog"
        className="function-reference-trigger"
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          toggleCard();
        }}
        ref={referenceRef}
        type="button"
      >
        <span className="trace-function">{highlightSearchText(props.name, props.highlightTerms)}</span>
      </button>
      {isActive ? createPortal(card, document.body) : null}
    </span>
  );
}

function commaWrappedText(value: string) {
  return value.split(",").map((part, index, parts) => (
    <span key={`${part}-${index}`}>
      {part}
      {index < parts.length - 1 ? (
        <>
          ,<wbr />
        </>
      ) : null}
    </span>
  ));
}
