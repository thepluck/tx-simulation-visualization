import type { FormState, UpdateForm } from "../../app/form";
import type { ERC20ApprovalOverride, ERC20BalanceOverride, ERC721ApprovalOverride, LabelOverride } from "../../api/types";
import OverrideGroup, { type OverrideField } from "../../components/OverrideGroup";

const labelFields: OverrideField<LabelOverride>[] = [
  { key: "account", label: "Account", placeholder: "0x..." },
  { key: "label", label: "Label", placeholder: "owner" }
];

const erc20BalanceFields: OverrideField<ERC20BalanceOverride>[] = [
  { key: "token", label: "Token", placeholder: "0x..." },
  { key: "account", label: "Account", placeholder: "0x..." },
  { key: "balance", label: "Balance", placeholder: "1000000000000000000" }
];

const erc20ApprovalFields: OverrideField<ERC20ApprovalOverride>[] = [
  { key: "token", label: "Token", placeholder: "0x..." },
  { key: "owner", label: "Owner", placeholder: "0x..." },
  { key: "spender", label: "Spender", placeholder: "0x..." },
  { key: "amount", label: "Amount", placeholder: "1000000000000000000" }
];

const erc721ApprovalFields: OverrideField<ERC721ApprovalOverride>[] = [
  { key: "token", label: "Token", placeholder: "0x..." },
  { key: "owner", label: "Owner", placeholder: "0x..." },
  { key: "spender", label: "Spender", placeholder: "0x..." },
  { key: "tokenId", label: "Token ID", placeholder: "1" }
];

export default function ScriptOverridesTab(props: { form: FormState; onUpdate: UpdateForm }) {
  const { form, onUpdate } = props;
  return (
    <section className="tab-panel active">
      <OverrideGroup
        createRow={emptyLabelOverride}
        fields={labelFields}
        rows={form.labelOverrides}
        title="Labels"
        onRowsChange={(rows) => onUpdate("labelOverrides", rows)}
      />
      <OverrideGroup
        createRow={emptyERC20BalanceOverride}
        fields={erc20BalanceFields}
        rows={form.erc20BalanceOverrides}
        title="ERC20 Balances"
        onRowsChange={(rows) => onUpdate("erc20BalanceOverrides", rows)}
      />
      <OverrideGroup
        createRow={emptyERC20ApprovalOverride}
        fields={erc20ApprovalFields}
        rows={form.erc20ApprovalOverrides}
        title="ERC20 Approvals"
        onRowsChange={(rows) => onUpdate("erc20ApprovalOverrides", rows)}
      />
      <OverrideGroup
        createRow={emptyERC721ApprovalOverride}
        fields={erc721ApprovalFields}
        rows={form.erc721ApprovalOverrides}
        title="ERC721 Approvals"
        onRowsChange={(rows) => onUpdate("erc721ApprovalOverrides", rows)}
      />
    </section>
  );
}

function emptyLabelOverride(): LabelOverride {
  return { account: "", label: "" };
}

function emptyERC20BalanceOverride(): ERC20BalanceOverride {
  return { token: "", account: "", balance: "" };
}

function emptyERC20ApprovalOverride(): ERC20ApprovalOverride {
  return { token: "", owner: "", spender: "", amount: "" };
}

function emptyERC721ApprovalOverride(): ERC721ApprovalOverride {
  return { token: "", owner: "", spender: "", tokenId: "" };
}
