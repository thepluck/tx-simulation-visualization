import { useState, type MouseEvent } from "react";
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
  const { cardRef, cardStyle, closeCard, isActive, referenceRef, toggleCard } = useFloatingCard<HTMLSpanElement, HTMLSpanElement>();
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

  return (
    <span
      className={`address-reference ${isActive ? "active" : ""}`}
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
      <span className={`address-reference-text ${className}`}>{highlightSearchText(display, props.highlightTerms)}</span>
      <span
        className="address-reference-card"
        onClick={blockFloatingCardEvent}
        onPointerDown={blockFloatingCardEvent}
        ref={cardRef}
        role="tooltip"
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
    </span>
  );
}
