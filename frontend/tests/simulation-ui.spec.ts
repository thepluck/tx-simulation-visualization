import { expect, test, type Page } from "@playwright/test";

const apiURL = "http://127.0.0.1:8080";
const explorerURL = "https://explorer.test";
const token = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const helper = "0x1111111111111111111111111111111111111111";
const owner = "0x0000000000000000000000000000000000000001";
const spender = "0x0000000000000000000000000000000000000002";
const recipient = "0x0000000000000000000000000000000000000003";
const longBytes = `0x${"a".repeat(64)}`;
const shortBytes = "0xaaaaaaaa...aaaaaaaa";

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
  await page.route(`${apiURL}/projects`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ projects: ["~/Kyber/ks-dex-aggregator-sc"] })
    });
  });
  await page.route(`${apiURL}/simulate`, async (route) => {
    const request = route.request().postDataJSON() as { etherscanApiKey?: string; labelOverrides?: Array<{ account: string; label: string }> };
    expect(request.labelOverrides).toContainEqual({ account: spender, label: "Sender" });
    expect(request.etherscanApiKey).toBe("etherscan-test-key");
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
  await expect(page.locator("datalist#project-history option[value='~/Kyber/ks-dex-aggregator-sc']")).toHaveCount(1);
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
  await page.getByLabel("Etherscan API Key").fill("etherscan-test-key");
  await expect(page.getByLabel("Etherscan API Key")).toHaveValue("etherscan-test-key");

  await page.reload();
  await expect(page.getByText("online")).toBeVisible();
  await expect(page.getByLabel("Trace expand depth")).toHaveValue("1");
  await expect(page.getByLabel("Foundry Project")).toHaveValue("/Users/test/foundry-project");
  await expect(page.getByLabel("Block")).toHaveValue("23000000");
  await expect(page.getByLabel("Sender")).toHaveValue(spender);
  await expect(page.getByLabel("Target")).toHaveValue(token);
  await expect(page.getByLabel("Calldata")).toHaveValue("0x23b872dd");
  await expect(page.getByLabel("Etherscan API Key")).toHaveValue("etherscan-test-key");

  await page.getByRole("button", { name: "Run Simulation" }).click();
  await expect(page.getByText("success |")).toBeVisible();

  await expect(page.getByRole("img", { name: "Fund flow graph" })).toBeVisible();
  await expect(page.locator(`.flow-svg a[href="${explorerURL}/address/${owner}"]`)).toHaveCount(1);
  await expect(page.locator(`.flow-svg a[href="${explorerURL}/address/${recipient}"]`)).toHaveCount(1);
  await expect(page.locator(`.edge-table a[href="${explorerURL}/address/${owner}"]`)).toHaveCount(50);
  await expect(page.locator(".edge-table tbody tr").nth(0).locator(`a[href="${explorerURL}/address/${owner}"]`)).toHaveText("WETHOwner");

  await page.evaluate(() => {
    window.scrollTo(0, 360);
  });
  const flowScrollTop = await page.evaluate(() => window.scrollY);
  expect(flowScrollTop).toBeGreaterThan(250);

  await clickOutputTab(page, "Trace");
  await expect(page.locator(".trace-tree")).toContainText("transferFrom");
  await expect(page.locator(".trace-tree")).toContainText("Transfer(from:");
  await expect(page.locator(".trace-tree")).not.toContainText("emit Transfer");
  await expect(page.locator(".trace-tree")).not.toContainText("approve");
  await expect(page.locator(".trace-tree")).not.toContainText("[Return]");
  await expect(page.locator(`.trace-tree a[href="${explorerURL}/address/${token}"]`)).toHaveCount(1);
  await expect(page.locator(`.trace-tree a[href="${explorerURL}/address/${spender}"]`)).toHaveText("Sender");
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
  await expect(page.locator(`.balance-analysis-table a[href="${explorerURL}/address/${recipient}"]`)).toHaveText("WETHRecipient");
  await expectAddressTooltipStaysOpen(page, "WETHRecipient");
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

async function expectAddressTooltipStaysOpen(page: Page, label: string) {
  const reference = page.locator(".address-reference").filter({ has: page.getByRole("link", { name: label }) }).first();
  await reference.hover();

  const card = reference.locator(".address-reference-card");
  await expect(card).toBeVisible();

  const box = await card.boundingBox();
  expect(box).not.toBeNull();
  await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2, { steps: 8 });
  await expect(card).toBeVisible();
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
            arguments: `${owner}, ${recipient}, 1000000000000000000, ${longBytes}`,
            children: [
              {
                raw: `[40] ${spender}::fallback()`,
                kind: "call",
                gas: 40,
                target: spender,
                function: "fallback"
              },
              {
                raw: `[60] ${helper}::decode(${longBytes})`,
                kind: "call",
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
