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

test("OpenAPI path templates declare matching path parameters", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    const templateParams = [...path.matchAll(/{([^}]+)}/g)].map((match) => match[1]);
    if (templateParams.length === 0) continue;

    for (const [method, operation] of Object.entries(pathItem)) {
      const parameters = operation.parameters || [];
      for (const paramName of templateParams) {
        assert.ok(
          parameters.some((parameter) => parameter.in === "path" && parameter.name === paramName && parameter.required === true),
          `${method.toUpperCase()} ${path} must declare required path parameter ${paramName}`
        );
      }
    }
  }
});

test("OpenAPI operations expose generator-friendly metadata", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  assert.ok(Array.isArray(openapi.servers) && openapi.servers.length > 0, "OpenAPI must declare at least one server");
  assert.ok(Array.isArray(openapi.security) && openapi.security.length > 0, "OpenAPI must declare root security");

  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      const label = `${method.toUpperCase()} ${path}`;
      assert.equal(typeof operation.operationId, "string", `${label} must declare operationId`);
      assert.ok(operation.operationId.length > 0, `${label} operationId must not be empty`);
      assert.equal(typeof operation.summary, "string", `${label} must declare summary`);
      assert.ok(operation.summary.length > 0, `${label} summary must not be empty`);
      assert.ok(
        Object.keys(operation.responses || {}).some((status) => /^4\d\d$/.test(status)),
        `${label} must declare a 4xx response`
      );
    }
  }
});

test("OpenAPI only publishes currently implemented HTTP routes", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  assert.deepEqual(Object.keys(openapi.paths).sort(), ["/api/fabric/catalog", "/api/fabric/readiness"]);
});

test("OpenAPI success responses declare JSON schemas", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      const label = `${method.toUpperCase()} ${path}`;
      const ok = operation.responses?.["200"];
      assert.ok(ok?.content?.["application/json"]?.schema, `${label} must declare 200 application/json schema`);
    }
  }
});
