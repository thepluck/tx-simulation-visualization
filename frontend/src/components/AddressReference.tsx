import { explorerAddressUrl } from "../explorer";
import { shortAddress } from "../format";
import { labelForAddress, type AddressLabels } from "../labels";

type AddressReferenceProps = {
  address: string;
  addressLabels: AddressLabels;
  className?: string;
  explorerBaseUrl: string;
};

export default function AddressReference(props: AddressReferenceProps) {
  const label = labelForAddress(props.address, props.addressLabels);
  const display = label || shortAddress(props.address, 8);
  const href = explorerAddressUrl(props.explorerBaseUrl, props.address);
  return (
    <span className="address-reference">
      {href ? (
        <a className={`address-reference-link ${props.className ?? ""}`} href={href} rel="noreferrer" target="_blank">
          {display}
        </a>
      ) : (
        <span className={`address-reference-text ${props.className ?? ""}`}>{display}</span>
      )}
      <span className="address-reference-card" role="tooltip">
        <span>{label || display}</span>
        <code>{props.address}</code>
      </span>
    </span>
  );
}
