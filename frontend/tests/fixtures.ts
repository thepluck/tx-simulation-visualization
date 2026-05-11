import type { Page } from "@playwright/test";

export const apiURL = "http://127.0.0.1:8080";
export const explorerURL = "https://explorer.test";
export const token = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
export const owner = "0x0000000000000000000000000000000000000001";
export const spender = "0x0000000000000000000000000000000000000002";
export const recipient = "0x0000000000000000000000000000000000000003";
export const searchOnlyAccount = "0x0000000000000000000000000000000000000004";
export const longBytes = `0x${"a".repeat(64)}`;
export const shortBytes = "0xaaaaaaaa...aaaaaaaa";

const helper = "0x1111111111111111111111111111111111111111";

export async function routeBaseEndpoints(page: Page) {
  await page.route(`${apiURL}/health`, async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ ok: true }) });
  });
  await page.route(`${apiURL}/chains`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        chains: ["mainnet"],
        explorerUrls: {
          mainnet: explorerURL
        }
      })
    });
  });
  await page.route(`${apiURL}/projects`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ projects: ["~/Kyber/ks-dex-aggregator-sc"] })
    });
  });
}

export function simulateResponse() {
  const eventChildren = Array.from({ length: 50 }, (_, index) => ({
    raw: `emit Transfer(from: WETHOwner: [${owner}], to: TraceRecipient: [${recipient}], value: ${index + 1})`,
    kind: "event",
    value: `Transfer(from: WETHOwner: [${owner}], to: TraceRecipient: [${recipient}], value: ${index + 1})`
  }));
  const transfers = [
    ...Array.from({ length: 50 }, () => ({
      token,
      from: owner,
      to: recipient,
      amount: "1000000000000000000",
      normalizedAmount: "1",
      symbol: "WETH",
      logoUrl: ""
    })),
    {
      token,
      from: owner,
      to: helper,
      amount: "1000000000000000000",
      normalizedAmount: "1",
      symbol: "WETH",
      logoUrl: ""
    },
    {
      token,
      from: helper,
      to: spender,
      amount: "1000000000000000000",
      normalizedAmount: "1",
      symbol: "WETH",
      logoUrl: ""
    },
    {
      token,
      from: spender,
      to: recipient,
      amount: "1000000000000000000",
      normalizedAmount: "1",
      symbol: "WETH",
      logoUrl: ""
    }
  ];

  return {
    id: "browser-test",
    success: true,
    exitCode: 0,
    durationMillis: 12,
    trace: "mock trace",
    structuredTrace: [
      {
        raw: "[1000] SimulateTxScript::run()",
        kind: "call",
        gas: 1000,
        target: "SimulateTxScript",
        function: "run",
        children: [
          {
            raw: `[100] ${token}::approve(${spender}, 1000000000000000000)`,
            kind: "call",
            gas: 100,
            target: token,
            function: "approve",
            arguments: `${spender}, 1000000000000000000`
          },
          {
            raw: "[0] VM::recordLogs()",
            kind: "call",
            gas: 0,
            target: "VM",
            function: "recordLogs"
          },
          {
            raw: `[0] VM::prank(Sender: [${spender}])`,
            kind: "call",
            gas: 0,
            target: "VM",
            function: "prank",
            arguments: `Sender: [${spender}]`
          },
          {
            raw: `[400] WETH9::transferFrom(srcToken: WETH9: [${token}], ${owner}, ${recipient}, 1000000000000000000)`,
            kind: "call",
            callType: "delegatecall",
            gas: 400,
            target: "WETH9",
            function: "transferFrom",
            arguments: `srcToken: WETH9: [${token}], ${owner}, ${recipient}, 1000000000000000000, callTarget: [${helper}], ${longBytes}`,
            children: [
              {
                raw: `[40] Sender: [${spender}]::fallback()`,
                kind: "call",
                gas: 40,
                target: `Sender: [${spender}]`,
                function: "fallback"
              },
              {
                raw: `[60] ${helper}::decode(${longBytes})`,
                kind: "call",
                callType: "staticcall",
                gas: 60,
                target: helper,
                function: "decode",
                arguments: longBytes,
                children: [
                  {
                    raw: "← [Revert] helper decode failed",
                    kind: "revert",
                    resultType: "Revert",
                    value: "helper decode failed"
                  }
                ]
              },
              {
                raw: "[70] MetaAggregationRouterV2::swap()",
                kind: "call",
                gas: 70,
                target: "MetaAggregationRouterV2",
                function: "swap"
              },
              {
                raw: "[75] SearchOnlyAlias::inspect()",
                kind: "call",
                gas: 75,
                target: "SearchOnlyAlias",
                function: "inspect"
              },
              {
                raw: `[80] UnmappedToken::transfer(${recipient}, 1000000000000000000)`,
                kind: "call",
                gas: 80,
                target: "UnmappedToken",
                function: "transfer",
                arguments: `${recipient}, 1000000000000000000`
              },
              ...eventChildren,
              {
                raw: "← [Stop]",
                kind: "stop",
                resultType: "Stop",
                value: "← [Stop]"
              },
              {
                raw: "← [Return]",
                kind: "return"
              }
            ]
          },
          {
            raw: "[0] VM::getRecordedLogs()",
            kind: "call",
            gas: 0,
            target: "VM",
            function: "getRecordedLogs"
          },
          {
            raw: "[0] console2::log(TXSIM_LOG|0x0000000000000000000000000000000000000000|3)",
            kind: "call",
            gas: 0,
            target: "console2",
            function: "log",
            arguments: "TXSIM_LOG|0x0000000000000000000000000000000000000000|3"
          }
        ]
      }
    ],
    erc20Transfers: transfers,
    balanceAnalysis: {
      changes: [
        {
          user: owner,
          token,
          symbol: "WETH",
          rawAmount: "-1000000000000000000",
          amount: "-1",
          usdValue: -3500
        },
        {
          user: recipient,
          token,
          symbol: "WETH",
          rawAmount: "1000000000000000000",
          amount: "1",
          usdValue: 3500
        }
      ],
      userTotals: [
        {
          user: owner,
          usdValue: -3500
        },
        {
          user: recipient,
          usdValue: 3500
        }
      ]
    }
  };
}
