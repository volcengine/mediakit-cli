#!/usr/bin/env node

const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const https = require("node:https");
const http = require("node:http");

const pkg = require("../package.json");

const PROJECT_NAME = "mediakit-cli";

// Deprecated env names (MEDIKIT_*) are still read for back-compat but emit a
// warning. New env names use the MEDIAKIT_* prefix (Rule 22). The 0.3.0
// release will drop the deprecated forms.
function readEnv(currentName, deprecatedName) {
  const current = process.env[currentName];
  if (current !== undefined && current !== "") {
    return current;
  }
  const deprecated = process.env[deprecatedName];
  if (deprecated !== undefined && deprecated !== "") {
    console.warn(
      `[mediakit-cli] env ${deprecatedName} is deprecated, please use ${currentName}`
    );
    return deprecated;
  }
  return undefined;
}

const RELEASE_BASE_URL =
  readEnv("MEDIAKIT_CLI_RELEASE_BASE_URL", "MEDIKIT_CLI_RELEASE_BASE_URL") ||
  "https://github.com/volcengine/mediakit-cli/releases/download";
const TMP_PREFIX = "mediakit-cli-install-";
const packageRoot = path.resolve(__dirname, "..");
const binDir = path.join(packageRoot, "bin");
const skillsDir = path.join(packageRoot, "skills");

const platformMap = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

function resolvePlatform() {
  const platform = platformMap[process.platform];
  if (!platform) {
    throw new Error(`unsupported platform: ${process.platform}`);
  }
  return platform;
}

function resolveArch() {
  const arch = archMap[process.arch];
  if (!arch) {
    throw new Error(`unsupported architecture: ${process.arch}`);
  }
  return arch;
}

function binaryName() {
  return process.platform === "win32" ? `${PROJECT_NAME}.exe` : PROJECT_NAME;
}

function releaseTag(version) {
  return version.startsWith("v") ? version : `v${version}`;
}

function archiveName(version, platform, arch) {
  const ext = platform === "windows" ? "zip" : "tar.gz";
  return `${PROJECT_NAME}_${version}_${platform}_${arch}.${ext}`;
}

function request(url, redirectsLeft = 5) {
  const client = url.startsWith("https:") ? https : http;
  return new Promise((resolve, reject) => {
    const req = client.get(
      url,
      {
        headers: {
          "user-agent": "mediakit-cli-installer",
          accept: "*/*",
        },
      },
      (res) => {
        if (
          res.statusCode &&
          res.statusCode >= 300 &&
          res.statusCode < 400 &&
          res.headers.location
        ) {
          if (redirectsLeft <= 0) {
            reject(new Error(`too many redirects for ${url}`));
            return;
          }
          resolve(request(res.headers.location, redirectsLeft - 1));
          return;
        }

        if (res.statusCode !== 200) {
          reject(new Error(`download failed: ${url} -> ${res.statusCode}`));
          return;
        }

        resolve(res);
      }
    );
    req.on("error", reject);
  });
}

async function downloadToFile(url, destination) {
  await fs.promises.mkdir(path.dirname(destination), { recursive: true });
  const response = await request(url);
  await new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destination);
    response.pipe(file);
    response.on("error", reject);
    file.on("finish", () => file.close(resolve));
    file.on("error", reject);
  });
}

async function downloadText(url) {
  const response = await request(url);
  const chunks = [];
  for await (const chunk of response) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString("utf8");
}

function sha256(filePath) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(filePath));
  return hash.digest("hex");
}

function expectedChecksum(checksums, filename) {
  const line = checksums
    .split(/\r?\n/)
    .map((entry) => entry.trim())
    .find((entry) => entry.endsWith(` ${filename}`) || entry.endsWith(`  ${filename}`));
  if (!line) {
    throw new Error(`checksum not found for ${filename}`);
  }
  return line.split(/\s+/)[0];
}

function ensureSuccess(result, label) {
  if (result.status !== 0) {
    throw new Error(`${label} failed with exit code ${result.status}`);
  }
}

function extractArchive(archivePath, extractDir, platform) {
  fs.mkdirSync(extractDir, { recursive: true });

  if (platform === "windows") {
    const result = spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-Command",
        `Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${extractDir}' -Force`,
      ],
      { stdio: "inherit" }
    );
    ensureSuccess(result, "Expand-Archive");
    return;
  }

  const result = spawnSync("tar", ["-xzf", archivePath, "-C", extractDir], {
    stdio: "inherit",
  });
  ensureSuccess(result, "tar extract");
}

function findBinary(rootDir, targetName) {
  const entries = fs.readdirSync(rootDir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(rootDir, entry.name);
    if (entry.isDirectory()) {
      const nested = findBinary(fullPath, targetName);
      if (nested) {
        return nested;
      }
    } else if (entry.isFile() && entry.name === targetName) {
      return fullPath;
    }
  }
  return null;
}

function installSkills() {
  console.log("[mediakit-cli] Installing skills ...");
  console.log(`[mediakit-cli] skills source: ${skillsDir}`);
  const result = spawnSync(
    "npx",
    ["-y", "skills", "add", skillsDir, "-g", "-y"],
    { stdio: "inherit" }
  );
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`npx skills add failed with exit code ${result.status}`);
  }
  console.log("[mediakit-cli] ✓ Skills installed");
}

async function install(options = {}) {
  if (readEnv("MEDIAKIT_CLI_SKIP_DOWNLOAD", "MEDIKIT_CLI_SKIP_DOWNLOAD") === "1") {
    console.log("[mediakit-cli] skip download because MEDIAKIT_CLI_SKIP_DOWNLOAD=1");
    installSkills();
    return;
  }

  const platform = resolvePlatform();
  const arch = resolveArch();
  const version =
    readEnv("MEDIAKIT_CLI_VERSION", "MEDIKIT_CLI_VERSION") || pkg.version;
  const targetBinary = path.join(binDir, binaryName());

  if (!options.force && fs.existsSync(targetBinary)) {
    installSkills();
    return;
  }

  const tmpDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), TMP_PREFIX));
  const archive = archiveName(version, platform, arch);
  const archivePath = path.join(tmpDir, archive);
  const extractDir = path.join(tmpDir, "extract");
  const tag = releaseTag(version);
  const archiveUrl = `${RELEASE_BASE_URL}/${tag}/${archive}`;
  const checksumsUrl = `${RELEASE_BASE_URL}/${tag}/checksums.txt`;

  console.log(`[mediakit-cli] downloading ${archiveUrl}`);
  await downloadToFile(archiveUrl, archivePath);

  console.log(`[mediakit-cli] downloading ${checksumsUrl}`);
  const checksums = await downloadText(checksumsUrl);
  const expected = expectedChecksum(checksums, archive);
  const actual = sha256(archivePath);
  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${archive}`);
  }

  extractArchive(archivePath, extractDir, platform);
  const extractedBinary = findBinary(extractDir, binaryName());
  if (!extractedBinary) {
    throw new Error(`binary ${binaryName()} not found in ${archive}`);
  }

  await fs.promises.mkdir(binDir, { recursive: true });
  await fs.promises.copyFile(extractedBinary, targetBinary);
  if (platform !== "windows") {
    await fs.promises.chmod(targetBinary, 0o755);
  }

  console.log(`[mediakit-cli] installed ${targetBinary}`);
  installSkills();
}

if (require.main === module) {
  install({ force: process.argv.includes("--force") }).catch((error) => {
    console.error(`[mediakit-cli] ${error.message}`);
    process.exit(1);
  });
}

module.exports = { install };
