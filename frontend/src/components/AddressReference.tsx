import { useState, type MouseEvent } from "react";
import { createPortal } from "react-dom";
import { explorerAddressUrl } from "../lib/explorer";
import { shortAddress } from "../lib/format";
import { labelForAddress, type AddressLabels } from "../lib/labels";
import CopyIcon from "./CopyIcon";
import { highlightSearchText } from "./SearchHighlight";
import { blockFloatingCardEvent, stopFloatingCardEvent, useFloatingCard } from "./useFloatingCard";

type AddressReferenceProps = {
  address: string;
  addressLabels: AddressLabels;
  className?: string;
  displayLabel?: string;
  explorerBaseUrl: string;
  highlightTerms?: string[];
};

export default function AddressReference(props: AddressReferenceProps) {
  const [copied, setCopied] = useState(false);
  const { cardRef, cardStyle, closeCard, isActive, keepOpenOnCardPointerDown, referenceRef, shouldCloseOnBlur, toggleCard } =
    useFloatingCard<HTMLButtonElement, HTMLSpanElement>();
  const label = labelForAddress(props.address, props.addressLabels);
  const displayOverride = label || props.displayLabel?.trim();
  const display = displayOverride || shortAddress(props.address, 8);
  const href = explorerAddressUrl(props.explorerBaseUrl, props.address);
  const className = props.className ?? "";

  const copyAddress = async (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    try {
      await navigator.clipboard.writeText(props.address);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  };

  const card = (
    <span
      aria-label="Address details"
      className="address-reference-card"
      onClick={blockFloatingCardEvent}
      onPointerDown={keepOpenOnCardPointerDown}
      ref={cardRef}
      role="dialog"
      style={cardStyle}
    >
      <span className="address-reference-card-row">
        {href ? (
          <a
            className="address-reference-card-link"
            href={href}
            rel="noreferrer"
            target="_blank"
            onClick={stopFloatingCardEvent}
            onPointerDown={stopFloatingCardEvent}
          >
            {props.address}
          </a>
        ) : (
          <code>{props.address}</code>
        )}
        <button
          className={`address-copy-button${copied ? " copied" : ""}`}
          type="button"
          aria-label={`Copy ${props.address}`}
          title={copied ? "Copied" : "Copy address"}
          onClick={copyAddress}
          onPointerDown={stopFloatingCardEvent}
        >
          <CopyIcon />
        </button>
      </span>
    </span>
  );

  return (
    <span
      className={`address-reference ${isActive ? "active" : ""}`}
      onBlur={(event) => {
        if (shouldCloseOnBlur(event.currentTarget, event.relatedTarget)) {
          closeCard();
        }
      }}
    >
      <button
        aria-expanded={isActive}
        aria-haspopup="dialog"
        className="address-reference-trigger"
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          toggleCard();
        }}
        ref={referenceRef}
        type="button"
      >
        <span className={`address-reference-text ${className}`}>{highlightSearchText(display, props.highlightTerms)}</span>
      </button>
      {isActive ? createPortal(card, document.body) : null}
    </span>
  );
}
