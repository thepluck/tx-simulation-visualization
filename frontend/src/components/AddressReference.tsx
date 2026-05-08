import { useCallback, useEffect, useLayoutEffect, useRef, useState, type CSSProperties } from "react";
import { explorerAddressUrl } from "../explorer";
import { shortAddress } from "../format";
import { labelForAddress, type AddressLabels } from "../labels";

type AddressReferenceProps = {
  address: string;
  addressLabels: AddressLabels;
  className?: string;
  displayLabel?: string;
  explorerBaseUrl: string;
};

export default function AddressReference(props: AddressReferenceProps) {
  const [copied, setCopied] = useState(false);
  const [isActive, setIsActive] = useState(false);
  const [cardStyle, setCardStyle] = useState<CSSProperties>({});
  const cardRef = useRef<HTMLSpanElement | null>(null);
  const closeTimerRef = useRef<number | null>(null);
  const referenceRef = useRef<HTMLSpanElement | null>(null);
  const label = labelForAddress(props.address, props.addressLabels);
  const display = label || props.displayLabel?.trim() || shortAddress(props.address, 8);
  const href = explorerAddressUrl(props.explorerBaseUrl, props.address);
  const className = props.className ?? "";

  const clearCloseTimer = useCallback(() => {
    if (closeTimerRef.current !== null) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
  }, []);

  const openCard = useCallback(() => {
    clearCloseTimer();
    setIsActive(true);
  }, [clearCloseTimer]);

  const scheduleCloseCard = useCallback(() => {
    clearCloseTimer();
    closeTimerRef.current = window.setTimeout(() => {
      setIsActive(false);
      closeTimerRef.current = null;
    }, 120);
  }, [clearCloseTimer]);

  const positionCard = useCallback(() => {
    const card = cardRef.current;
    const reference = referenceRef.current;
    if (!card || !reference) {
      return;
    }

    const margin = 8;
    const gap = 6;
    const referenceBox = reference.getBoundingClientRect();
    const cardBox = card.getBoundingClientRect();
    const maxWidth = Math.max(1, window.innerWidth - margin * 2);
    const maxHeight = Math.max(1, window.innerHeight - margin * 2);
    const cardWidth = Math.min(cardBox.width || maxWidth, maxWidth);
    const cardHeight = Math.min(cardBox.height || maxHeight, maxHeight);
    const maxLeft = Math.max(margin, window.innerWidth - margin - cardWidth);
    const maxTop = Math.max(margin, window.innerHeight - margin - cardHeight);

    let left = referenceBox.left;
    if (left > maxLeft) {
      left = maxLeft;
    }
    left = Math.max(margin, left);

    let top = referenceBox.bottom + gap;
    if (top > maxTop && referenceBox.top - cardHeight - gap >= margin) {
      top = referenceBox.top - cardHeight - gap;
    }
    top = Math.max(margin, Math.min(top, maxTop));

    setCardStyle({
      left: `${Math.round(left)}px`,
      maxHeight: `${maxHeight}px`,
      maxWidth: `${maxWidth}px`,
      top: `${Math.round(top)}px`
    });
  }, []);

  useLayoutEffect(() => {
    if (isActive) {
      positionCard();
    }
  }, [isActive, positionCard]);

  useEffect(() => {
    if (!isActive) {
      return;
    }
    window.addEventListener("resize", positionCard);
    window.addEventListener("scroll", positionCard, true);
    return () => {
      window.removeEventListener("resize", positionCard);
      window.removeEventListener("scroll", positionCard, true);
    };
  }, [isActive, positionCard]);

  useEffect(() => clearCloseTimer, [clearCloseTimer]);

  const copyAddress = async () => {
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
          scheduleCloseCard();
        }
      }}
      onFocus={openCard}
      onPointerEnter={openCard}
      onPointerLeave={scheduleCloseCard}
      ref={referenceRef}
      tabIndex={0}
    >
      <span className={`address-reference-text ${className}`}>{display}</span>
      <span
        className="address-reference-card"
        onPointerEnter={openCard}
        onPointerLeave={scheduleCloseCard}
        ref={cardRef}
        role="tooltip"
        style={cardStyle}
      >
        <span className="address-reference-card-row">
          {href ? (
            <a className="address-reference-card-link" href={href} rel="noreferrer" target="_blank">
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
          />
        </span>
      </span>
    </span>
  );
}
