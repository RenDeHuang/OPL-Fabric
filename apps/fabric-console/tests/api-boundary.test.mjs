import { readFile } from "node:fs/promises";
import { test } from "node:test";
import assert from "node:assert/strict";

test("operator token is not read by browser API code", async () => {
  const api = await readFile(new URL("../src/api.ts", import.meta.url), "utf8");

  assert.equal(api.includes("VITE_OPL_OPERATOR_TOKEN"), false);
  assert.equal(api.includes("Authorization:"), false);
});

test("vite dev proxy injects bearer token from server environment", async () => {
  const config = await readFile(new URL("../vite.config.ts", import.meta.url), "utf8");

  assert.match(config, /process\.env\.OPL_OPERATOR_TOKEN/);
  assert.match(config, /proxyReq\.setHeader\("Authorization", `Bearer \$\{token\}`\)/);
  assert.match(config, /127\.0\.0\.1:8787/);
});
