#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const { install } = require("./install");
const { runInstallWizard } = require("./install-wizard");

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

// Detect whether the CLI is being run via `npx @volcengine/mediakit-cli install`,
// where it should bootstrap a global installation rather than execute the
// embedded binary inside the npx cache. MEDIAKIT_CLI_RUN=1 is a re-entry guard.
function shouldInterceptInstall(args) {
  if (!args || args.length === 0 || args[0] !== "install") {
    return false;
  }
  if (process.env.MEDIAKIT_CLI_RUN === "1" || process.env.MEDIKIT_CLI_RUN === "1") {
    return false;
  }
  return process.env.npm_command === "exec";
}

async function main() {
  const argv = process.argv.slice(2);
  if (shouldInterceptInstall(argv)) {
    process.env.MEDIAKIT_CLI_RUN = "1";
    await runInstallWizard(argv.slice(1));
    return;
  }
  await ensureBinary();
  const result = spawnSync(binary, argv, {
    stdio: "inherit",
    env: { ...process.env, MEDIAKIT_CLI_RUN: "1" },
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
