#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const path = require("node:path");

const pkg = require("../package.json");

const PACKAGE_NAME = pkg.name || "@volcengine/mediakit-cli";
const SKILL_REPO = "volcengine/mediakit-cli";

function parseArgs(argv) {
  const opts = {
    cliOnly: false,
    skillsOnly: false,
    skills: [],
    yes: false,
    versionTag: "latest",
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case "--cli-only":
        opts.cliOnly = true;
        break;
      case "--skills-only":
        opts.skillsOnly = true;
        break;
      case "-y":
      case "--yes":
        opts.yes = true;
        break;
      case "-s":
      case "--skill": {
        const next = argv[i + 1];
        if (next && !next.startsWith("-")) {
          opts.skills.push(next);
          i++;
        }
        break;
      }
      case "--version": {
        const next = argv[i + 1];
        if (next && !next.startsWith("-")) {
          opts.versionTag = next;
          i++;
        }
        break;
      }
      default:
        if (a.startsWith("--version=")) {
          opts.versionTag = a.slice("--version=".length);
        } else if (a.startsWith("--skills=")) {
          opts.skills.push(
            ...a.slice("--skills=".length).split(",").map((s) => s.trim()).filter(Boolean)
          );
        }
        break;
    }
  }
  return opts;
}

function log(msg) {
  console.log(`[mediakit-cli install] ${msg}`);
}

function whichSync(cmd) {
  const probe = process.platform === "win32" ? "where" : "command";
  const probeArgs = process.platform === "win32" ? [cmd] : ["-v", cmd];
  const result = spawnSync(probe, probeArgs, { stdio: "ignore", shell: true });
  return result.status === 0;
}

function runNpmInstall(target) {
  log(`installing ${target} via npm install -g`);
  const result = spawnSync("npm", ["install", "-g", target], {
    stdio: "inherit",
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`npm install -g ${target} failed with exit code ${result.status}`);
  }
}

function runSkillsAdd(opts) {
  const args = ["-y", "skills", "add", SKILL_REPO];
  if (opts.skills.length === 0) {
    args.push("-g");
  } else {
    for (const skill of opts.skills) {
      args.push("-s", skill);
    }
  }
  if (opts.yes) {
    args.push("-y");
  }
  log(`installing skills via npx ${args.join(" ")}`);
  const result = spawnSync("npx", args, { stdio: "inherit" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`npx skills add failed with exit code ${result.status}`);
  }
}

function isCliAlreadyInstalled() {
  // If MEDIAKIT_CLI_FORCE_REINSTALL=1, always reinstall.
  if (
    process.env.MEDIAKIT_CLI_FORCE_REINSTALL === "1" ||
    process.env.MEDIKIT_CLI_FORCE_REINSTALL === "1"
  ) {
    return false;
  }
  return whichSync("mediakit-cli");
}

async function runInstallWizard(rawArgs) {
  const opts = parseArgs(rawArgs);

  if (!whichSync("npm")) {
    throw new Error("npm is required but not found in PATH");
  }

  if (!opts.skillsOnly) {
    if (isCliAlreadyInstalled()) {
      log("mediakit-cli already installed; skipping CLI install");
    } else {
      const target = `${PACKAGE_NAME}@${opts.versionTag}`;
      runNpmInstall(target);
    }
  }

  if (!opts.cliOnly) {
    if (!whichSync("npx")) {
      throw new Error("npx is required to install skills but not found in PATH");
    }
    runSkillsAdd(opts);
  }

  log("done.");
}

module.exports = { runInstallWizard };

if (require.main === module) {
  runInstallWizard(process.argv.slice(2)).catch((error) => {
    console.error(`[mediakit-cli install] ${error.message}`);
    process.exit(1);
  });
}
