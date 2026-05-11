import { expect, test } from "@playwright/test";
import { apiURL, owner, routeBaseEndpoints, simulateResponse, spender, token } from "./fixtures";

test("loads a saved request by request id", async ({ page }) => {
  await routeBaseEndpoints(page);
  const savedRequest = {
    chain: "mainnet",
    blockNumber: "23000001",
    projectPath: "/Users/test/saved-project",
    labelOverrides: [{ account: owner, label: "SavedOwner" }],
    erc20BalanceOverrides: [],
    erc20ApprovalOverrides: [],
    erc721ApprovalOverrides: [],
    stateOverride: {
      contractName: "SavedOverride",
      source: "pragma solidity ^0.8.0; contract SavedOverride {}"
    },
    compiler: {
      viaIR: false,
      optimize: true,
      optimizerRuns: 300,
      evmVersion: "cancun",
      revertStrings: "debug"
    },
    sender: spender,
    target: token,
    data: "0x23b872dd"
  };
  await page.route(`${apiURL}/requests/saved-run`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        id: "saved-run",
        request: savedRequest,
        response: { ...simulateResponse(), id: "saved-run", durationMillis: 22 }
      })
    });
  });

  await page.goto("/?requestId=saved-run");

  await expect(page.getByLabel("Request ID")).toHaveValue("saved-run");
  await expect(page.getByLabel("Block")).toHaveValue("23000001");
  await expect(page.getByLabel("Foundry Project")).toHaveValue("/Users/test/saved-project");
  await expect(page.getByLabel("Sender")).toHaveValue(spender);
  await expect(page.getByLabel("Target")).toHaveValue(token);
  await expect(page.getByLabel("Calldata")).toHaveValue("0x23b872dd");
  await expect(page.getByText("success | 22ms | exit 0 | saved-run")).toBeVisible();

  await page.getByRole("button", { name: "Override Contract" }).click();
  await expect(page.getByLabel("Override Contract Name")).toHaveValue("SavedOverride");
  await expect(page.getByLabel("Override Contract Source")).toHaveValue("pragma solidity ^0.8.0; contract SavedOverride {}");
  await page.getByRole("button", { name: "Compiler" }).click();
  await expect(page.getByLabel("Optimizer Runs")).toHaveValue("300");
  await expect(page.getByLabel("EVM Version")).toHaveValue("cancun");
  await expect(page.getByLabel("Revert Strings")).toHaveValue("debug");
});

test("loads legacy state override fields from a saved request", async ({ page }) => {
  await routeBaseEndpoints(page);
  await page.route(`${apiURL}/requests/legacy-override-run`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        id: "legacy-override-run",
        request: {
          chain: "mainnet",
          blockNumber: "23000001",
          erc20BalanceOverrides: [],
          erc20ApprovalOverrides: [],
          erc721ApprovalOverrides: [],
          stateOverrideCode: "pragma solidity ^0.8.0; contract LegacyOverride {}",
          stateOverrideContractName: "LegacyOverride",
          sender: spender,
          target: token,
          data: "0x23b872dd"
        },
        response: { ...simulateResponse(), id: "legacy-override-run", durationMillis: 22 }
      })
    });
  });

  await page.goto("/?requestId=legacy-override-run");
  await page.getByRole("button", { name: "Override Contract" }).click();

  await expect(page.getByLabel("Override Contract Name")).toHaveValue("LegacyOverride");
  await expect(page.getByLabel("Override Contract Source")).toHaveValue("pragma solidity ^0.8.0; contract LegacyOverride {}");
});

test("editing the request id clears a stuck lookup", async ({ page }) => {
  await routeBaseEndpoints(page);
  let releaseLookup: () => void = () => {};
  const stalledLookup = new Promise<void>((resolve) => {
    releaseLookup = resolve;
  });
  await page.route(`${apiURL}/requests/stuck-run`, async (route) => {
    await stalledLookup;
    try {
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({ error: "request record not found" })
      });
    } catch {
      // The app aborts the request when the user edits the Request ID.
    }
  });

  await page.goto("/");
  await page.getByLabel("Request ID").fill("stuck-run");
  await page.getByRole("button", { name: "Open" }).click();
  await expect(page.getByRole("button", { name: "Opening..." })).toBeDisabled();

  await page.getByLabel("Request ID").fill("saved-run");
  await expect(page.getByRole("button", { name: "Open" })).toBeEnabled();
  releaseLookup();
});
