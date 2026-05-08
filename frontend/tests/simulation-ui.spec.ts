import { expect, test, type Locator, type Page } from "@playwright/test";

const apiURL = "http://127.0.0.1:8080";
const explorerURL = "https://explorer.test";
const token = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const helper = "0x1111111111111111111111111111111111111111";
const owner = "0x0000000000000000000000000000000000000001";
const spender = "0x0000000000000000000000000000000000000002";
const recipient = "0x0000000000000000000000000000000000000003";
const longBytes = `0x${"a".repeat(64)}`;
const shortBytes = "0xaaaaaaaa...aaaaaaaa";

test("changes the running action to abort and cancels the active request", async ({ page }) => {
  await routeBaseEndpoints(page);
  await page.route(`${apiURL}/simulate`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 750));
    try {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          id: "abort-test",
          success: true,
          exitCode: 0,
          durationMillis: 1,
          trace: "mock trace",
          structuredTrace: []
        })
      });
    } catch {
      // The request is expected to be aborted before the mock response completes.
    }
  });

  await page.goto("/");
  await page.getByLabel("Block").fill("23000000");
  await page.getByLabel("Sender").fill(spender);
  await page.getByLabel("Target").fill(token);
  await page.getByLabel("Calldata").fill("0x23b872dd");

  await page.getByRole("button", { name: "Run Simulation" }).click();
  await expect(page.getByRole("button", { name: "Abort" })).toBeVisible();
  await page.getByRole("button", { name: "Abort" }).click();
  await expect(page.getByText("Simulation aborted")).toBeVisible();
  await expect(page.getByRole("button", { name: "Run Simulation" })).toBeVisible();
});

test("uses configured explorer links and renders only the last main call subtree", async ({ page }) => {
  await routeBaseEndpoints(page);
  await page.route(`${apiURL}/simulate`, async (route) => {
    const request = route.request().postDataJSON() as { etherscanApiKey?: string; labelOverrides?: Array<{ account: string; label: string }> };
    expect(request.labelOverrides).toContainEqual({ account: spender, label: "Sender" });
    expect(request.etherscanApiKey).toBeUndefined();
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(simulateResponse())
    });
  });
  await page.route(`${apiURL}/browse/project`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ path: "/Users/test/foundry-project" })
    });
  });

  await page.goto("/");
  await expect(page.getByText("online")).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.getByRole("button", { name: "Use dark theme" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByRole("button", { name: "Use light theme" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Past projects" })).toBeEnabled();
  await page.getByRole("button", { name: "Past projects" }).click();
  const projectHistoryMenu = page.getByRole("menu");
  await expect(projectHistoryMenu.getByRole("menuitem", { name: "~/Kyber/ks-dex-aggregator-sc" })).toBeVisible();
  await projectHistoryMenu.getByRole("menuitem", { name: "~/Kyber/ks-dex-aggregator-sc" }).click();
  await expect(page.getByLabel("Foundry Project")).toHaveValue("~/Kyber/ks-dex-aggregator-sc");
  await page.getByLabel("Trace expand depth").fill("1");

  await page.getByRole("button", { name: "Browse" }).click();
  await expect(page.getByLabel("Foundry Project")).toHaveValue("/Users/test/foundry-project");

  await page.getByLabel("Block").fill("23000000");
  await page.getByLabel("Sender").fill(spender);
  await page.getByLabel("Target").fill(token);
  await page.getByLabel("Calldata").fill("0x23b872dd");
  await addLabel(page, owner, "WETHOwner");
  await addLabel(page, recipient, "WETHRecipient");
  await page.getByRole("button", { name: "Compiler" }).click();
  await expect(page.getByLabel("Solc")).toBeVisible();

  await page.reload();
  await expect(page.getByText("online")).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByRole("button", { name: "Use light theme" })).toBeVisible();
  await expect(page.getByLabel("Trace expand depth")).toHaveValue("1");
  await expect(page.getByLabel("Foundry Project")).toHaveValue("/Users/test/foundry-project");
  await expect(page.getByLabel("Block")).toHaveValue("23000000");
  await expect(page.getByLabel("Sender")).toHaveValue(spender);
  await expect(page.getByLabel("Target")).toHaveValue(token);
  await expect(page.getByLabel("Calldata")).toHaveValue("0x23b872dd");
  await page.getByRole("button", { name: "Compiler" }).click();
  await expect(page.getByLabel("Solc")).toBeVisible();

  await page.getByRole("button", { name: "Run Simulation" }).click();
  await expect(page.getByText("success |")).toBeVisible();

  await expect(page.getByRole("img", { name: "Fund flow graph" })).toBeVisible();
  await page.reload();
  await expect(page.getByText("success | 12ms | exit 0 | browser-test")).toBeVisible();
  await expect(page.getByRole("img", { name: "Fund flow graph" })).toBeVisible();

  await expect(page.locator(".flow-svg .react-flow__edge-path")).toHaveCount(4);
  await expect(page.locator(".edge-label")).toHaveCount(4);
  await expect(page.locator(".edge-label").filter({ hasText: "[1-50]" })).toBeVisible();
  await expect.poll(() => flowColumnCount(page)).toBeGreaterThanOrEqual(3);
  await expect(page.locator(`.flow-svg a[href="${explorerURL}/address/${owner}"]`)).toHaveCount(1);
  await expect(page.locator(`.flow-svg a[href="${explorerURL}/address/${recipient}"]`)).toHaveCount(1);
  await expect(page.locator(`.edge-table .address-reference-card-link[href="${explorerURL}/address/${owner}"]`)).toHaveCount(51);
  await expect(page.locator(".edge-table tbody tr").nth(0).locator(".address-reference-text").first()).toHaveText("WETHOwner");

  await page.evaluate(() => {
    window.scrollTo(0, 360);
  });
  const flowScrollTop = await page.evaluate(() => window.scrollY);
  expect(flowScrollTop).toBeGreaterThan(250);

  await clickOutputTab(page, "Trace");
  await expect(page.locator(".trace-tree")).toContainText("transferFrom");
  await expect(page.locator(".trace-tree")).toContainText("UnmappedToken");
  await expect(page.locator(".trace-tree")).toContainText("Transfer(from:");
  await expect(page.locator(".trace-tree")).toContainText("callTarget:");
  await expect(page.locator(".trace-tree")).not.toContainText("WETH9");
  await expect(page.locator(".trace-tree")).not.toContainText("TraceRecipient");
  await expect(page.locator(".trace-tree")).not.toContainText(`WETHRecipient: [${recipient}]`);
  await expect(page.locator(".trace-tree")).not.toContainText("emit Transfer");
  await expect(page.locator(".trace-tree")).not.toContainText("approve");
  await expect(page.locator(".trace-tree")).not.toContainText("TXSIM_LOG");
  await expect(page.locator(".trace-tree")).not.toContainText("console2");
  await expect(page.locator(".trace-tree")).not.toContainText("getRecordedLogs");
  await expect(page.locator(".trace-tree")).not.toContainText("SimulateTxScript");
  await expect(page.locator(".trace-tree")).not.toContainText("[Return]");
  await expect(page.locator(".trace-tree")).not.toContainText("[Stop]");
  await expect(page.locator(".trace-tree")).not.toContainText("delegatecall | 400 gas");
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-kind").first()).toHaveText("delegatecall");
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-meta").first()).toHaveText("400 gas");
  await expect(page.locator(".trace-tree .address-reference-text").filter({ hasText: "callTarget" })).toHaveCount(0);
  await expect(page.locator(`.trace-tree .address-reference-card-link[href="${explorerURL}/address/${token}"]`)).toHaveCount(2);
  const tokenReference = page
    .locator(".trace-tree .address-reference")
    .filter({ has: page.locator(`.address-reference-card-link[href="${explorerURL}/address/${token}"]`) })
    .first();
  await expect(tokenReference.locator(".address-reference-text")).toHaveText("WETH");
  await expect(page.locator(".trace-tree .address-reference").filter({ hasText: "WETHRecipient" }).first()).toBeVisible();
  await expect(page.locator(".trace-tree .address-reference").filter({ hasText: "Sender" }).first()).toBeVisible();
  await expect(page.locator(".trace-tree .trace-main > .address-reference-text").filter({ hasText: "UnmappedToken" })).toHaveCount(1);
  await expect(page.locator(".trace-tree .trace-main > .address-reference-text").filter({ hasText: "MetaAggregationRouterV2" })).toHaveCount(1);
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(flowScrollTop);
  await expectTraceDepth(page, 1, [true, false]);

  const bytesButton = page.locator(".trace-bytes-toggle").first();
  await expect(bytesButton).toHaveText(shortBytes);
  await bytesButton.click();
  await expect(bytesButton).toHaveText(longBytes);
  await bytesButton.click();
  await expect(bytesButton).toHaveText(shortBytes);

  await page.getByLabel("Trace expand depth").fill("2");
  await expectTraceDepth(page, 2, [true, true]);

  const traceMain = page.locator(".trace-main").first();
  await expect(traceMain).toHaveCSS("white-space", "normal");
  await expect(traceMain).toHaveCSS("text-align", "left");

  await page.evaluate(() => {
    window.scrollTo(0, 180);
  });
  await clickOutputTab(page, "Flow");
  await expect(page.getByRole("img", { name: "Fund flow graph" })).toBeVisible();
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(flowScrollTop);

  await page.getByRole("button", { name: "Balances" }).click();
  await expect(page.locator(".balance-analysis-table .address-reference").filter({ hasText: "WETHRecipient" }).first()).toBeVisible();
  await expectAddressTooltipStaysOpen(page, "WETHRecipient", recipient);
  await expectAddressTooltipClampsToViewport(page, "WETHRecipient", recipient);
});

async function routeBaseEndpoints(page: Page) {
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

async function clickOutputTab(page: Page, name: string) {
  await page.getByRole("button", { name }).evaluate((element) => {
    if (element instanceof HTMLElement) {
      element.click();
    }
  });
}

async function addLabel(page: Page, account: string, label: string) {
  const labelsGroup = page.locator(".override-group").filter({ has: page.getByRole("heading", { name: "Labels" }) });
  await labelsGroup.getByRole("button", { name: "+" }).click();
  const rows = labelsGroup.locator(".override-row");
  const row = rows.nth((await rows.count()) - 1);
  await row.getByLabel("Account").fill(account);
  await row.getByLabel("Label").fill(label);
}

async function expectAddressTooltipStaysOpen(page: Page, label: string, address: string) {
  const reference = page.locator(".address-reference").filter({ hasText: label }).first();
  await reference.hover();

  const card = reference.locator(".address-reference-card");
  await expect(card).toBeVisible();
  await expect(card.locator(".address-reference-card-row")).toHaveCSS("flex-wrap", "nowrap");
  await expect(card.locator(`a[href="${explorerURL}/address/${address}"]`)).toHaveText(address);
  await expect(card.getByRole("button", { name: `Copy ${address}` })).toBeVisible();
  await expect(card.locator(".address-reference-card-label")).toHaveCount(0);
  await expect(card).not.toContainText(label);

  const box = await card.boundingBox();
  expect(box).not.toBeNull();
  await expectLocatorInsideViewport(page, card);
  await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2, { steps: 8 });
  await expect(card).toBeVisible();
}

async function expectAddressTooltipClampsToViewport(page: Page, label: string, address: string) {
  const originalViewport = page.viewportSize();
  await page.setViewportSize({ width: 260, height: 180 });
  try {
    const reference = page.locator(".address-reference").filter({ hasText: label }).first();
    await reference.evaluate((element) => {
      const target = element as HTMLElement;
      target.style.position = "fixed";
      target.style.right = "2px";
      target.style.bottom = "2px";
      target.style.zIndex = "1000";
    });
    await reference.hover();

    const card = reference.locator(".address-reference-card");
    await expect(card).toBeVisible();
    await expect(card.locator(`a[href="${explorerURL}/address/${address}"]`)).toHaveText(address);
    await expectLocatorInsideViewport(page, card);
  } finally {
    if (originalViewport) {
      await page.setViewportSize(originalViewport);
    }
  }
}

async function expectLocatorInsideViewport(page: Page, locator: Locator) {
  const box = await locator.boundingBox();
  const viewport = page.viewportSize();
  expect(box).not.toBeNull();
  expect(viewport).not.toBeNull();
  expect(box!.x).toBeGreaterThanOrEqual(0);
  expect(box!.y).toBeGreaterThanOrEqual(0);
  expect(box!.x + box!.width).toBeLessThanOrEqual(viewport!.width + 1);
  expect(box!.y + box!.height).toBeLessThanOrEqual(viewport!.height + 1);
}

async function flowColumnCount(page: Page): Promise<number> {
  return page.locator(".flow-svg .react-flow__node").evaluateAll((nodes) => {
    const lefts = nodes.map((node) => Math.round(node.getBoundingClientRect().left / 8) * 8);
    return new Set(lefts).size;
  });
}

async function expectTraceDepth(page: Page, depth: number, expectedOpenStates: boolean[]) {
  await expect(page.getByLabel("Trace expand depth")).toHaveValue(String(depth));
  const nodes = page.locator(".trace-node");
  await expect(nodes).toHaveCount(expectedOpenStates.length);
  for (const [index, expected] of expectedOpenStates.entries()) {
    await expect.poll(() => nodes.nth(index).evaluate((node) => (node as HTMLDetailsElement).open)).toBe(expected);
  }
}

function simulateResponse() {
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
