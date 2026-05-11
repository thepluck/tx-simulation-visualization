import { useState } from "react";
import type { MouseEvent } from "react";
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
  const { cardRef, cardStyle, closeCard, isActive, referenceRef, toggleCard } = useFloatingCard<HTMLSpanElement, HTMLSpanElement>();
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

  return (
    <span
      className={`function-reference ${isActive ? "active" : ""}`}
      onBlur={(event) => {
        const nextTarget = event.relatedTarget;
        if (!(nextTarget instanceof Node) || !event.currentTarget.contains(nextTarget)) {
          closeCard();
        }
      }}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        toggleCard();
      }}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") {
          return;
        }
        event.preventDefault();
        event.stopPropagation();
        toggleCard();
      }}
      ref={referenceRef}
      tabIndex={0}
    >
      <span className="trace-function">{highlightSearchText(props.name, props.highlightTerms)}</span>
      <span
        className="function-reference-card"
        onClick={blockFloatingCardEvent}
        onPointerDown={blockFloatingCardEvent}
        ref={cardRef}
        role="tooltip"
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
