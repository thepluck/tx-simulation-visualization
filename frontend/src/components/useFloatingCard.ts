import { useCallback, useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type SyntheticEvent } from "react";

export function useFloatingCard<ReferenceElement extends HTMLElement, CardElement extends HTMLElement>() {
  const [isActive, setIsActive] = useState(false);
  const [cardStyle, setCardStyle] = useState<CSSProperties>({});
  const cardRef = useRef<CardElement | null>(null);
  const cardPointerDownRef = useRef(false);
  const referenceRef = useRef<ReferenceElement | null>(null);

  const openCard = useCallback(() => setIsActive(true), []);
  const closeCard = useCallback(() => setIsActive(false), []);
  const toggleCard = useCallback(() => setIsActive((current) => !current), []);
  const keepOpenOnCardPointerDown = useCallback((event: SyntheticEvent) => {
    cardPointerDownRef.current = true;
    event.stopPropagation();
    window.setTimeout(() => {
      cardPointerDownRef.current = false;
    }, 0);
  }, []);

  const shouldCloseOnBlur = useCallback((currentTarget: HTMLElement, nextTarget: EventTarget | null) => {
    if (cardPointerDownRef.current) {
      return false;
    }
    return !(nextTarget instanceof Node) || (!currentTarget.contains(nextTarget) && !cardRef.current?.contains(nextTarget));
  }, []);

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

  useEffect(() => {
    if (!isActive) {
      return;
    }

    const closeOnOutsidePointerDown = (event: PointerEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) {
        return;
      }
      if (referenceRef.current?.contains(target) || cardRef.current?.contains(target)) {
        return;
      }
      closeCard();
    };

    window.addEventListener("pointerdown", closeOnOutsidePointerDown);
    return () => window.removeEventListener("pointerdown", closeOnOutsidePointerDown);
  }, [closeCard, isActive]);

  return {
    cardRef,
    cardStyle,
    closeCard,
    isActive,
    keepOpenOnCardPointerDown,
    openCard,
    referenceRef,
    shouldCloseOnBlur,
    toggleCard
  };
}

export function stopFloatingCardEvent(event: SyntheticEvent) {
  event.stopPropagation();
}

export function blockFloatingCardEvent(event: SyntheticEvent) {
  event.stopPropagation();
}
