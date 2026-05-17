#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const { install } = require("./install");

const packageRoot = path.resolve(__dirname, "..");
const binDir = path.join(packageRoot, "bin");
const binary =
  process.platform === "win32"
    ? path.join(binDir, "mediakit-cli.exe")
    : path.join(binDir, "mediakit-cli");

async function ensureBinary() {
  if (fs.existsSync(binary)) {
    return;
  }
  await install({ force: true });
}

async function main() {
  await ensureBinary();
  const result = spawnSync(binary, process.argv.slice(2), {
    stdio: "inherit",
  });

  if (result.error) {
    throw result.error;
  }
  process.exit(result.status ?? 0);
}

main().catch((error) => {
  console.error(`[mediakit-cli] ${error.message}`);
  process.exit(1);
});
