import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const contractFiles = [
  "contracts/fabric-api.openapi.json",
  "contracts/fabric-event-envelope.schema.json",
  "contracts/fabric-human-gate.schema.json",
  "contracts/fabric-lifecycle-ledger.schema.json",
  "contracts/fabric-operation-receipt.schema.json",
  "contracts/fabric-resource-catalog.schema.json",
  "contracts/fabric-runtime-supervision.schema.json",
  "contracts/fabric-storage-volume.schema.json",
  "contracts/fabric-compute-resource.schema.json",
  "contracts/fabric-storage-attachment.schema.json",
  "contracts/fabric-workspace-entry.schema.json",
  "contracts/fabric-workspace.schema.json"
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
  assert.deepEqual(Object.keys(openapi.paths).sort(), [
    "/api/fabric/catalog",
    "/api/fabric/compute-resources",
    "/api/fabric/compute-resources/{id}",
    "/api/fabric/compute-resources/{id}/destroy",
    "/api/fabric/operations/{id}",
    "/api/fabric/readiness",
    "/api/fabric/storage-attachments",
    "/api/fabric/storage-attachments/{id}",
    "/api/fabric/storage-attachments/{id}/detach",
    "/api/fabric/storage-volumes",
    "/api/fabric/storage-volumes/{id}",
    "/api/fabric/storage-volumes/{id}/destroy",
    "/api/fabric/workspace-entries",
    "/api/fabric/workspace-entries/{id}",
    "/api/fabric/workspaces",
    "/api/fabric/workspaces/{id}"
  ]);
});

test("OpenAPI success responses declare JSON schemas", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      const label = `${method.toUpperCase()} ${path}`;
      const ok = Object.entries(operation.responses || {}).find(([status]) => /^2\d\d$/.test(status))?.[1];
      assert.ok(ok?.content?.["application/json"]?.schema, `${label} must declare 2xx application/json schema`);
    }
  }
});

test("OpenAPI mutating routes require idempotency and correlation headers", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (!["post", "delete", "patch", "put"].includes(method)) continue;
      const parameters = operation.parameters || [];
      assert.ok(
        parameters.some((parameter) => parameter.in === "header" && parameter.name === "Idempotency-Key" && parameter.required === true),
        `${method.toUpperCase()} ${path} must require Idempotency-Key`
      );
      assert.ok(
        parameters.some((parameter) => parameter.in === "header" && parameter.name === "X-Correlation-Id" && parameter.required === true),
        `${method.toUpperCase()} ${path} must require X-Correlation-Id`
      );
      assert.equal(operation.responses?.["202"]?.content?.["application/json"]?.schema?.$ref, "#/components/schemas/OperationReceipt");
    }
  }
});

test("staging live e2e workspace image matches configured TCR namespace", () => {
  const workflow = readFileSync(".github/workflows/fabric-staging-live-e2e.yml", "utf8");
  const workspaceImage = workflow.match(/^\s+OPL_WORKSPACE_IMAGE:\s+(\S+)/m)?.[1];
  const tcrNamespace = workflow.match(/^\s+TENCENT_TCR_NAMESPACE:\s+(\S+)/m)?.[1];

  assert.equal(tcrNamespace, "oplcloud");
  assert.equal(workspaceImage, `uswccr.ccs.tencentyun.com/${tcrNamespace}/one-person-lab-app:latest`);
});
