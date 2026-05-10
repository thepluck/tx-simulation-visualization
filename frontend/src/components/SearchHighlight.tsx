import type { ReactNode } from "react";

export function highlightSearchText(value: string, terms: string[] = []): ReactNode {
  const normalizedTerms = normalizeHighlightTerms(terms);
  if (normalizedTerms.length === 0) {
    return value;
  }

  const lowerValue = value.toLowerCase();
  const pieces: ReactNode[] = [];
  let cursor = 0;

  while (cursor < value.length) {
    const match = nextHighlightMatch(lowerValue, normalizedTerms, cursor);
    if (!match) {
      break;
    }
    const { index: matchIndex, term } = match;
    if (matchIndex > cursor) {
      pieces.push(value.slice(cursor, matchIndex));
    }
    const end = matchIndex + term.length;
    pieces.push(
      <span className="trace-search-highlight" key={`${matchIndex}-${end}`}>
        {value.slice(matchIndex, end)}
      </span>
    );
    cursor = end;
  }

  if (cursor < value.length) {
    pieces.push(value.slice(cursor));
  }
  return pieces.length > 0 ? pieces : value;
}

function normalizeHighlightTerms(terms: string[]): string[] {
  return Array.from(new Set(terms.map((term) => term.trim().toLowerCase()).filter(Boolean))).sort((left, right) => right.length - left.length);
}

function nextHighlightMatch(value: string, terms: string[], cursor: number): { index: number; term: string } | undefined {
  let nextIndex = -1;
  let nextTerm = "";
  for (const term of terms) {
    const index = value.indexOf(term, cursor);
    if (index === -1) {
      continue;
    }
    if (nextIndex === -1 || index < nextIndex || (index === nextIndex && term.length > nextTerm.length)) {
      nextIndex = index;
      nextTerm = term;
    }
  }
  return nextIndex === -1 ? undefined : { index: nextIndex, term: nextTerm };
}
