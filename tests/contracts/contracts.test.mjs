import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const contractFiles = [
  "contracts/fabric-api.openapi.json",
  "contracts/fabric-event-envelope.schema.json",
  "contracts/fabric-human-gate.schema.json",
  "contracts/fabric-lifecycle-ledger.schema.json",
  "contracts/fabric-resource-catalog.schema.json",
  "contracts/fabric-runtime-supervision.schema.json"
];

test("contract files are valid JSON with stable identifiers", () => {
  for (const file of contractFiles) {
    const parsed = JSON.parse(readFileSync(file, "utf8"));
    assert.equal(typeof parsed, "object", file);
    assert.ok(parsed.$id || parsed.openapi, `${file} must expose $id or openapi`);
  }
});
