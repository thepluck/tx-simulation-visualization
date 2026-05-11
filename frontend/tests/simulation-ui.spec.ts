import { expect, test, type Locator, type Page } from "@playwright/test";
import {
  apiURL,
  explorerURL,
  helper,
  longBytes,
  owner,
  recipient,
  routeBaseEndpoints,
  searchOnlyAccount,
  shortBytes,
  simulateResponse,
  spender,
  token
} from "./fixtures";

test("shows validation errors for malformed simulation inputs", async ({ page }) => {
  await routeBaseEndpoints(page);
  let simulateCalls = 0;
  await page.route(`${apiURL}/simulate`, async (route) => {
    simulateCalls += 1;
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({ error: "simulate should not be called" })
    });
  });

  await page.goto("/");
  await page.getByLabel("Block").fill("not-a-block");
  await page.getByLabel("Sender").fill("not-an-address");
  await page.getByLabel("Target").fill(token);
  await page.getByLabel("Calldata").fill("0x123");

  await page.getByRole("button", { name: "Run Simulation" }).click();

  const errorBox = page.locator(".error-box");
  await expect(errorBox).toContainText("request validation failed:");
  await expect(errorBox).toContainText("blockNumber");
  await expect(errorBox).toContainText("sender");
  await expect(errorBox).toContainText("data");
  expect(simulateCalls).toBe(0);
});

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
          trace: "{}"
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
  await addLabel(page, searchOnlyAccount, "SearchOnlyAlias");
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
  await expect(page.getByLabel("Request ID")).toHaveValue("browser-test");

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
  await expect(page.locator(".trace-tree")).not.toContainText("callTarget:");
  await expect(page.locator(".trace-tree")).not.toContainText("WETH:");
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
  await expect(page.locator(".trace-tree > .trace-node > summary").first()).toHaveCSS("align-items", "center");
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-kind").first()).toHaveText("delegatecall");
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-meta").first()).toHaveText("400 gas");
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-meta").first()).toHaveClass(/trace-gas/);
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-meta").first()).not.toHaveCSS("font-weight", "800");
  await expect(page.locator(".trace-tree > .trace-node > summary .trace-meta").first()).toHaveCSS("transform", "none");
  const siblingKindLeftGroups = await page.locator(".trace-children").evaluateAll((groups) =>
    groups
      .map((group) =>
        Array.from(group.children)
          .map((entry) => {
            const row = entry.matches("details") ? entry.querySelector(":scope > summary") : entry.querySelector(":scope > .trace-leaf");
            return row?.querySelector(".trace-kind")?.getBoundingClientRect().left ?? null;
          })
          .filter((left): left is number => left !== null)
      )
      .filter((lefts) => lefts.length > 1)
  );
  expect(siblingKindLeftGroups.length).toBeGreaterThan(0);
  for (const lefts of siblingKindLeftGroups) {
    expect(Math.max(...lefts) - Math.min(...lefts)).toBeLessThanOrEqual(1);
  }
  await expect(page.locator(".trace-tree .address-reference-text").filter({ hasText: "callTarget" })).toHaveCount(0);
  await expect(page.locator(".trace-tree .address-reference-text").filter({ hasText: "srcToken" })).toHaveCount(0);
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(flowScrollTop);
  const transferFunction = page.locator(".trace-tree .function-reference").filter({ hasText: "transferFrom" }).first();
  const functionCard = transferFunction.locator(".function-reference-card");
  await expect(functionCard).toBeHidden();
  await transferFunction.hover();
  await expect(functionCard).toBeHidden();
  await transferFunction.click();
  await expect(functionCard).toBeVisible();
  await expectLocatorInsideViewport(page, functionCard);
  await expect(functionCard.locator("code").first()).toHaveCSS("white-space", "normal");
  await expect(functionCard.locator("code").first()).toHaveCSS("overflow-wrap", "normal");
  await expect(functionCard.locator("code").first().locator("wbr")).toHaveCount(2);
  await expect(functionCard).toContainText("transferFrom(address,address,uint256)");
  await expect(functionCard).toContainText("0x23b872dd");
  await expect(functionCard.getByRole("button", { name: "Copy function signature transferFrom(address,address,uint256)" })).toBeVisible();
  await expect(functionCard.getByRole("button", { name: "Copy function selector 0x23b872dd" })).toBeVisible();
  await functionCard.locator("code").first().click();
  await expect.poll(() => page.locator(".trace-tree > .trace-node").first().evaluate((node) => (node as HTMLDetailsElement).open)).toBe(true);
  await functionCard.getByRole("button", { name: "Copy function selector 0x23b872dd" }).click();
  await expect.poll(() => page.locator(".trace-tree > .trace-node").first().evaluate((node) => (node as HTMLDetailsElement).open)).toBe(true);
  await expect(page.locator(".trace-tree .function-reference").filter({ hasText: "fallback" })).toHaveCount(0);
  await expect(page.locator(".trace-tree .trace-function").filter({ hasText: "fallback" })).toHaveCount(1);
  await expect(page.locator(`.trace-tree .address-reference-card-link[href="${explorerURL}/address/${token}"]`)).toHaveCount(3);
  const tokenReference = page
    .locator(".trace-tree .address-reference")
    .filter({ has: page.locator(`.address-reference-card-link[href="${explorerURL}/address/${token}"]`) })
    .first();
  await expect(tokenReference.locator(".address-reference-text")).toHaveText("WETH");
  await tokenReference.hover();
  const tokenAddressCard = tokenReference.locator(".address-reference-card");
  await expect(tokenAddressCard).toBeHidden();
  await tokenReference.click();
  await expect(tokenAddressCard).toBeVisible();
  await tokenAddressCard.getByRole("button", { name: `Copy ${token}` }).click();
  await expect.poll(() => page.locator(".trace-tree > .trace-node").first().evaluate((node) => (node as HTMLDetailsElement).open)).toBe(true);
  await expect(page.locator(".trace-tree .address-reference").filter({ hasText: "WETHRecipient" }).first()).toBeVisible();
  await expect(page.locator(".trace-tree .address-reference").filter({ hasText: "Sender" }).first()).toBeVisible();
  await expect(page.locator(".trace-tree .address-reference").filter({ hasText: "UnmappedToken" })).toHaveCount(1);
  await expect(page.locator(".trace-tree .address-reference").filter({ hasText: "MetaAggregationRouterV2" })).toHaveCount(1);
  const helperReference = page
    .locator(".trace-tree .address-reference")
    .filter({ has: page.locator(`.address-reference-card-link[href="${explorerURL}/address/${helper}"]`) })
    .first();
  await expect(helperReference.locator(".address-reference-text")).toHaveText("0x11111111...11111111");
  const helperAddressCard = helperReference.locator(".address-reference-card");
  await helperReference.click();
  await expect(helperAddressCard).toBeVisible();
  await expect(helperAddressCard).toContainText(helper);
  await expect(helperReference.locator(".address-reference-text")).toHaveText("0x11111111...11111111");
  await helperReference.click();
  await expect(helperAddressCard).toBeHidden();
  await page.getByLabel("Search trace").fill(searchOnlyAccount);
  await expect(page.locator(".trace-search-count")).toHaveText("1/1");
  await expect(page.locator(".trace-search-match")).toHaveCount(1);
  await expect(page.locator(".trace-search-active")).toContainText("SearchOnlyAlias");
  await expect(page.locator(".trace-search-active .trace-search-highlight")).toHaveText("SearchOnlyAlias");
  await page.getByLabel("Clear trace search").click();
  await page.getByLabel("Search trace").fill("SearchOnly");
  await expect(page.locator(".trace-search-count")).toHaveText("1/1");
  await expect(page.locator(".trace-search-match")).toHaveCount(1);
  await expect(page.locator(".trace-search-active .trace-search-highlight")).toHaveText("SearchOnly");
  await page.getByLabel("Clear trace search").click();
  await expect(page.getByLabel("Search trace")).toHaveValue("");
  await expect(page.locator(".trace-search-match")).toHaveCount(0);
  await expect(page.locator(".trace-search-highlight")).toHaveCount(0);
  await page.getByLabel("Search trace").fill("decode");
  await expect(page.locator(".trace-search-count")).toHaveText("1/2");
  const decodeSummary = page.locator(".trace-node > summary").filter({ hasText: "decode" });
  await expect(decodeSummary).toHaveCount(1);
  await decodeSummary.click();
  await page.getByLabel("Next trace match").click();
  await expect(page.locator(".trace-search-count")).toHaveText("2/2");
  await expect(page.locator(".trace-search-active")).toContainText("helper decode failed");
  await expect(page.locator(".trace-search-active .trace-search-highlight")).toHaveText("decode");
  await page.getByLabel("Clear trace search").click();
  await expectTraceDepth(page, 1, [true, false]);

  const bytesReference = page.locator(".trace-bytes-reference").first();
  const bytesButton = bytesReference.locator(".trace-bytes-toggle");
  const bytesCard = bytesReference.locator(".trace-bytes-card");
  await expect(bytesButton).toHaveText(shortBytes);
  await bytesReference.hover();
  await expect(bytesCard).toBeHidden();
  await bytesButton.click();
  await expect(bytesCard).toBeVisible();
  await expect(bytesCard).toContainText(longBytes);
  await expect(bytesButton).toHaveText(shortBytes);
  await expectLocatorInsideViewport(page, bytesCard);
  await bytesCard.locator("code").click();
  await expect.poll(() => page.locator(".trace-tree > .trace-node").first().evaluate((node) => (node as HTMLDetailsElement).open)).toBe(true);
  const bytesCopyButton = bytesCard.getByRole("button", { name: "Copy bytes argument" });
  await expect(bytesCopyButton).toBeVisible();
  await bytesCopyButton.click();
  await expect(bytesButton).toHaveText(shortBytes);
  await expect.poll(() => page.locator(".trace-tree > .trace-node").first().evaluate((node) => (node as HTMLDetailsElement).open)).toBe(true);
  await bytesButton.click();
  await expect(bytesCard).toBeHidden();

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
  await expect(card).toBeHidden();
  await reference.click();
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
  await reference.click();
  await expect(card).toBeHidden();
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
    await expect(card).toBeHidden();
    await reference.click();
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
