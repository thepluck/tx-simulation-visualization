import { expect, test, type Page } from "@playwright/test";

const apiURL = "http://127.0.0.1:8080";
const explorerURL = "https://explorer.test";
const token = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const owner = "0x0000000000000000000000000000000000000001";
const spender = "0x0000000000000000000000000000000000000002";
const recipient = "0x0000000000000000000000000000000000000003";

test("uses configured explorer links and renders only the last main call subtree", async ({ page }) => {
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
  await page.route(`${apiURL}/simulate`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(simulateResponse())
    });
  });

  await page.goto("/");
  await expect(page.getByText("online")).toBeVisible();

  await page.getByLabel("Block").fill("23000000");
  await page.getByLabel("Sender").fill(spender);
  await page.getByLabel("Target").fill(token);
  await page.getByLabel("Calldata").fill("0x23b872dd");
  await addLabel(page, owner, "WETHOwner");
  await addLabel(page, spender, "WETHSpender");
  await addLabel(page, recipient, "WETHRecipient");

  await page.getByRole("button", { name: "Run Simulation" }).click();
  await expect(page.getByText("success |")).toBeVisible();

  await expect(page.getByRole("img", { name: "Fund flow graph" })).toBeVisible();
  await expect(page.locator(`svg.flow-svg a[href="${explorerURL}/address/${owner}"]`)).toHaveCount(1);
  await expect(page.locator(`svg.flow-svg a[href="${explorerURL}/address/${recipient}"]`)).toHaveCount(1);
  await expect(page.locator(`.edge-table a[href="${explorerURL}/address/${owner}"]`)).toHaveCount(50);
  await expect(page.locator(".edge-table tbody tr").nth(0).locator(`a[href="${explorerURL}/address/${owner}"]`)).toHaveText("WETHOwner");

  await page.evaluate(() => {
    window.scrollTo(0, 360);
  });
  const flowScrollTop = await page.evaluate(() => window.scrollY);
  expect(flowScrollTop).toBeGreaterThan(250);

  await clickOutputTab(page, "Trace");
  await expect(page.locator(".trace-tree")).toContainText("transferFrom");
  await expect(page.locator(".trace-tree")).not.toContainText("approve");
  await expect(page.locator(".trace-tree")).not.toContainText("[Return]");
  await expect(page.locator(`.trace-tree a[href="${explorerURL}/address/${token}"]`)).toHaveCount(1);
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(flowScrollTop);

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
  await expect(page.locator(`.balance-analysis-table a[href="${explorerURL}/address/${recipient}"]`)).toHaveText("WETHRecipient");
});

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

function simulateResponse() {
  const eventChildren = Array.from({ length: 50 }, (_, index) => ({
    raw: `emit Transfer(from: ${owner}, to: ${recipient}, value: ${index + 1})`,
    kind: "event",
    value: `Transfer(from: ${owner}, to: ${recipient}, value: ${index + 1})`
  }));
  const transfers = Array.from({ length: 50 }, () => ({
    token,
    from: owner,
    to: recipient,
    amount: "1000000000000000000",
    normalizedAmount: "1",
    symbol: "WETH",
    logoUrl: ""
  }));

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
            raw: `[400] ${token}::transferFrom(${owner}, ${recipient}, 1000000000000000000)`,
            kind: "call",
            gas: 400,
            target: token,
            function: "transferFrom",
            arguments: `${owner}, ${recipient}, 1000000000000000000, 0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`,
            children: [
              ...eventChildren,
              {
                raw: "← [Return]",
                kind: "return"
              }
            ]
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
