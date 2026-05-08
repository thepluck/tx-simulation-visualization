export type LabelOverride = {
  account: string;
  label: string;
};

export type ERC20BalanceOverride = {
  token: string;
  account: string;
  balance: string;
};

export type ERC20ApprovalOverride = {
  token: string;
  owner: string;
  spender: string;
  amount: string;
};

export type ERC721ApprovalOverride = {
  token: string;
  owner: string;
  spender: string;
  tokenId: string;
};

export type StateOverride = {
  contractName?: string;
  source: string;
};

export type CompilerConfig = {
  use?: string;
  offline?: boolean;
  noAutoDetect?: boolean;
  viaIR?: boolean;
  useLiteralContent?: boolean;
  noMetadata?: boolean;
  evmVersion?: string;
  optimize?: boolean;
  optimizerRuns?: number;
  revertStrings?: string;
};

export type ChainConfig = {
  chains: string[];
  explorerUrls: Record<string, string>;
};

export type ProjectsResponse = {
  projects: string[];
};

export type SimulateRequest = {
  chain: string;
  blockNumber: string;
  projectPath?: string;
  labelOverrides?: LabelOverride[];
  erc20BalanceOverrides?: ERC20BalanceOverride[];
  erc20ApprovalOverrides?: ERC20ApprovalOverride[];
  erc721ApprovalOverrides?: ERC721ApprovalOverride[];
  stateOverride?: StateOverride;
  compiler?: CompilerConfig;
  etherscanApiKey?: string;
  sender: string;
  target: string;
  data: string;
};

export type TraceNode = {
  raw: string;
  kind: string;
  gas?: number;
  target?: string;
  function?: string;
  arguments?: string;
  callType?: string;
  resultType?: string;
  value?: string;
  children?: TraceNode[];
};

export type ERC20Transfer = {
  token: string;
  from: string;
  to: string;
  amount: string;
  normalizedAmount?: string;
  symbol?: string;
  logoUrl?: string;
};

export type TokenBalanceChange = {
  user: string;
  token: string;
  symbol?: string;
  logoUrl?: string;
  rawAmount: string;
  amount: string;
  usdValue?: number;
};

export type UserUSDChange = {
  user: string;
  usdValue: number;
};

export type BalanceAnalysis = {
  changes?: TokenBalanceChange[];
  userTotals?: UserUSDChange[];
};

export type SimulateResponse = {
  id: string;
  success: boolean;
  exitCode: number;
  durationMillis: number;
  trace: string;
  structuredTrace?: TraceNode[];
  erc20Transfers?: ERC20Transfer[];
  balanceAnalysis?: BalanceAnalysis;
  error?: string;
};
