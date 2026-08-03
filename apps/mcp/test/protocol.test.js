import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import test from "node:test";

test("inicializa y publica la carga segura de comprobantes", async () => {
  const child = spawn(process.execPath, ["src/index.js"], {
    cwd: new URL("..", import.meta.url),
    env: { ...process.env, CONTROL_PROPIEDADES_API_URL: "https://example.invalid", CONTROL_PROPIEDADES_API_TOKEN: "test-token" },
    stdio: ["pipe", "pipe", "pipe"],
  });
  const responses = [];
  child.stdout.setEncoding("utf8");
  child.stdout.on("data", chunk => {
    for (const line of chunk.trim().split("\n")) if (line) responses.push(JSON.parse(line));
  });
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2025-06-18", capabilities: {}, clientInfo: { name: "test", version: "1" } } })}\n`);
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`);
  child.stdin.end();
  await new Promise((resolve, reject) => {
    child.once("error", reject);
    child.once("exit", code => code === 0 ? resolve() : reject(new Error(`MCP terminó con código ${code}`)));
  });
  assert.equal(responses.find(item => item.id === 1)?.result?.serverInfo?.version, "0.2.0");
  const tools = responses.find(item => item.id === 2)?.result?.tools ?? [];
  assert.equal(tools.length, 8);
  const upload = tools.find(tool => tool.name === "subir_comprobante_pago");
  assert.ok(upload);
  assert.deepEqual(upload.inputSchema.properties.confirm, { type: "boolean", const: true });
  const confirm = tools.find(tool => tool.name === "confirmar_pago");
  assert.deepEqual(confirm.inputSchema.properties.confirm, { type: "boolean", const: true });
  const incidentEvidence = tools.find(tool => tool.name === "subir_evidencia_incidente");
  assert.deepEqual(incidentEvidence.inputSchema.properties.confirm, { type: "boolean", const: true });
});
