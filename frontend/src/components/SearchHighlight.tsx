import type { ReactNode } from "react";

export function highlightSearchText(value: string, query: string): ReactNode {
  const trimmed = query.trim();
  if (!trimmed) {
    return value;
  }

  const lowerValue = value.toLowerCase();
  const lowerQuery = trimmed.toLowerCase();
  const pieces: ReactNode[] = [];
  let cursor = 0;
  let matchIndex = lowerValue.indexOf(lowerQuery);

  while (matchIndex !== -1) {
    if (matchIndex > cursor) {
      pieces.push(value.slice(cursor, matchIndex));
    }
    const end = matchIndex + trimmed.length;
    pieces.push(
      <span className="trace-search-highlight" key={`${matchIndex}-${end}`}>
        {value.slice(matchIndex, end)}
      </span>
    );
    cursor = end;
    matchIndex = lowerValue.indexOf(lowerQuery, cursor);
  }

  if (cursor < value.length) {
    pieces.push(value.slice(cursor));
  }
  return pieces.length > 0 ? pieces : value;
}
