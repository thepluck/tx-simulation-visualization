import { useState } from "react";

type TokenLogoProps = {
  logoUrl?: string;
  symbol: string;
};

export default function TokenLogo(props: TokenLogoProps) {
  const [failed, setFailed] = useState(false);
  if (!props.logoUrl || failed) {
    return <span className="token-logo-fallback">{props.symbol.slice(0, 1).toUpperCase()}</span>;
  }
  return (
    <img
      alt=""
      className="token-logo"
      loading="lazy"
      referrerPolicy="no-referrer"
      src={props.logoUrl}
      onError={() => setFailed(true)}
    />
  );
}
