import { classForSignedNumber, formatSignedTokenAmount, formatSignedUSD, shortAddress } from "../../lib/format";
import type { AddressLabels } from "../../lib/labels";
import type { BalanceAnalysis, TokenBalanceChange } from "../../api/types";
import AddressReference from "../../components/AddressReference";
import TokenLogo from "../../components/TokenLogo";

type UserBalanceRow = {
  changes: TokenBalanceChange[];
  totalUSD?: number;
  user: string;
};

export default function BalanceAnalysisView(props: { addressLabels: AddressLabels; analysis: BalanceAnalysis | undefined; explorerBaseUrl: string }) {
  const rows = buildUserRows(props.analysis);
  if (rows.length === 0) {
    return <div className="balance-analysis empty-state">No balance changes</div>;
  }
  return (
    <div className="balance-analysis-table-wrap">
      <table className="data-table balance-analysis-table">
        <thead>
          <tr>
            <th>User</th>
            <th>Balance Changes</th>
            <th>Total USD Change</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.user}>
              <td>
                <AddressReference address={row.user} addressLabels={props.addressLabels} explorerBaseUrl={props.explorerBaseUrl} />
              </td>
              <td>
                <div className="token-change-list">
                  {row.changes.map((change) => (
                    <TokenChangeItem change={change} key={`${change.token}-${change.rawAmount}`} />
                  ))}
                </div>
              </td>
              <td className={`total-usd-cell ${classForSignedNumber(row.totalUSD)}`}>{formatSignedUSD(row.totalUSD)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TokenChangeItem(props: { change: TokenBalanceChange }) {
  const { change } = props;
  const symbol = change.symbol || shortAddress(change.token, 4);
  return (
    <div className="token-change-item">
      <TokenLogo logoUrl={change.logoUrl} symbol={symbol} />
      <div className="token-change-main">
        <div className="token-change-topline">
          <span className="token-symbol" title={change.token}>
            {symbol}
          </span>
          <span className={classForSignedNumber(change.amount)}>{formatSignedTokenAmount(change.amount)}</span>
        </div>
        <div className={`token-change-usd ${classForSignedNumber(change.usdValue)}`}>{formatSignedUSD(change.usdValue)}</div>
      </div>
    </div>
  );
}

function buildUserRows(analysis: BalanceAnalysis | undefined): UserBalanceRow[] {
  const changes = analysis?.changes ?? [];
  const totals = new Map((analysis?.userTotals ?? []).map((total) => [total.user, total.usdValue]));
  const grouped = new Map<string, TokenBalanceChange[]>();

  for (const change of changes) {
    if (!grouped.has(change.user)) {
      grouped.set(change.user, []);
    }
    grouped.get(change.user)!.push(change);
  }

  const users = Array.from(new Set([...grouped.keys(), ...totals.keys()])).sort();
  return users.map((user) => ({
    user,
    changes: grouped.get(user) ?? [],
    totalUSD: totals.get(user)
  }));
}
