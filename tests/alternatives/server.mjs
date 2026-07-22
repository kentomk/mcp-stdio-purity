import { spawn } from "node:child_process";
import readline from "node:readline";

const modeIndex = process.argv.indexOf("--mode");
const mode = modeIndex >= 0 ? process.argv[modeIndex + 1] : "clean";
let cleanupScheduled = false;

function send(message) {
  process.stdout.write(`${JSON.stringify(message)}\n`);
}

function contaminateCleanup() {
  if (mode !== "cleanup" || cleanupScheduled) return;
  cleanupScheduled = true;
  const child = spawn(
    process.execPath,
    ["-e", "setTimeout(() => console.log('fixture cleanup log'), 40)"],
    { detached: true, stdio: ["ignore", process.stdout, process.stderr] },
  );
  child.unref();
}

if (mode === "startup") {
  console.log("fixture startup banner");
}

const lines = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
lines.on("line", (line) => {
  let request;
  try {
    request = JSON.parse(line);
  } catch {
    return;
  }

  if (request.method === "notifications/initialized") {
    if (mode === "late") console.log("fixture late log");
    return;
  }
  if (!("id" in request)) return;

  const base = { jsonrpc: "2.0", id: request.id };
  switch (request.method) {
    case "initialize":
      send({
        ...base,
        result: {
          protocolVersion: "2025-11-25",
          capabilities: { tools: {}, resources: {}, prompts: {} },
          serverInfo: { name: "purity-comparison-fixture", version: "0.0.0" },
        },
      });
      break;
    case "ping":
      send({ ...base, result: {} });
      contaminateCleanup();
      break;
    case "tools/list":
      send({ ...base, result: { tools: [] } });
      contaminateCleanup();
      break;
    case "resources/list":
      send({ ...base, result: { resources: [] } });
      break;
    case "resources/templates/list":
      send({ ...base, result: { resourceTemplates: [] } });
      break;
    case "prompts/list":
      send({ ...base, result: { prompts: [] } });
      break;
    case "logging/setLevel":
      send({ ...base, result: {} });
      break;
    default:
      send({ ...base, error: { code: -32601, message: "Method not found" } });
  }
});
