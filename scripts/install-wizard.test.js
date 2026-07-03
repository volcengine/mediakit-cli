const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const test = require('node:test')

const { runInstallWizard } = require('./install-wizard')

function writeExecutable(file, content) {
  fs.writeFileSync(file, content, { mode: 0o755 })
}

async function captureLogs(t, fn) {
  const oldLog = console.log
  const logs = []
  console.log = (...args) => logs.push(args.join(' '))
  t.after(() => {
    console.log = oldLog
  })
  await fn()
  console.log = oldLog
  return logs.join('\n')
}

test('installs skills from the npm package skills directory', async (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'mediakit-install-wizard-'))
  const bin = path.join(temp, 'bin')
  fs.mkdirSync(bin)
  const log = path.join(temp, 'calls.jsonl')

  writeExecutable(
    path.join(bin, 'npm'),
    `#!/usr/bin/env node
const fs = require('node:fs')
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ cmd: 'npm', args: process.argv.slice(2) }) + '\\n')
process.exit(0)
`,
  )
  writeExecutable(
    path.join(bin, 'npx'),
    `#!/usr/bin/env node
const fs = require('node:fs')
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ cmd: 'npx', args: process.argv.slice(2) }) + '\\n')
process.exit(0)
`,
  )

  t.after(() => fs.rmSync(temp, { recursive: true, force: true }))
  const oldPath = process.env.PATH
  process.env.PATH = `${bin}${path.delimiter}${oldPath || ''}`
  t.after(() => {
    process.env.PATH = oldPath
  })

  await runInstallWizard(['--skills-only', '--skill', 'byted-mediakit-video', '-y'])

  const calls = fs
    .readFileSync(log, 'utf8')
    .trim()
    .split('\n')
    .map((line) => JSON.parse(line))
  assert.equal(calls.length, 1)
  assert.equal(calls[0].cmd, 'npx')
  assert.deepEqual(calls[0].args, [
    '-y',
    'skills',
    'add',
    path.join(__dirname, '..', 'skills'),
    '-s',
    'byted-mediakit-video',
    '-g',
    '-y',
  ])
  assert(!calls[0].args.some((arg) => /^https?:\/\//.test(String(arg))))
})

test('regular install delegates skills installation to npm postinstall', async (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'mediakit-install-wizard-'))
  const bin = path.join(temp, 'bin')
  fs.mkdirSync(bin)
  const log = path.join(temp, 'calls.jsonl')

  writeExecutable(
    path.join(bin, 'npm'),
    `#!/usr/bin/env node
const fs = require('node:fs')
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ cmd: 'npm', args: process.argv.slice(2) }) + '\\n')
process.exit(0)
`,
  )
  writeExecutable(
    path.join(bin, 'npx'),
    `#!/usr/bin/env node
const fs = require('node:fs')
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ cmd: 'npx', args: process.argv.slice(2) }) + '\\n')
process.exit(0)
`,
  )

  t.after(() => fs.rmSync(temp, { recursive: true, force: true }))
  const oldPath = process.env.PATH
  process.env.PATH = `${bin}${path.delimiter}${oldPath || ''}`
  t.after(() => {
    process.env.PATH = oldPath
  })

  const output = await captureLogs(t, () => runInstallWizard(['-y']))

  const calls = fs
    .readFileSync(log, 'utf8')
    .trim()
    .split('\n')
    .map((line) => JSON.parse(line))
  assert.deepEqual(calls, [
    {
      cmd: 'npm',
      args: [
        'install',
        '-g',
        '@volcengine/mediakit-cli@0.1.7',
        '--foreground-scripts',
        '--ignore-scripts=false',
      ],
    },
  ])
  assert.match(output, /installing @volcengine\/mediakit-cli@0\.1\.7 via npm install -g/)
  assert.match(output, /Installing skills \.\.\./)
  assert.match(output, /✓ Successfully installed mediakit-cli 0\.1\.7/)
  assert.match(output, /✓ Skills installed/)
})

test('install wizard rejects unsupported version override', async (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'mediakit-install-wizard-'))
  const bin = path.join(temp, 'bin')
  fs.mkdirSync(bin)
  const log = path.join(temp, 'calls.jsonl')

  writeExecutable(
    path.join(bin, 'npm'),
    `#!/usr/bin/env node
const fs = require('node:fs')
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ cmd: 'npm', args: process.argv.slice(2) }) + '\\n')
process.exit(0)
`,
  )
  writeExecutable(
    path.join(bin, 'npx'),
    `#!/usr/bin/env node
process.exit(0)
`,
  )

  t.after(() => fs.rmSync(temp, { recursive: true, force: true }))
  const oldPath = process.env.PATH
  process.env.PATH = `${bin}${path.delimiter}${oldPath || ''}`
  t.after(() => {
    process.env.PATH = oldPath
  })

  await assert.rejects(
    () => runInstallWizard(['--version', '0.1.7', '-y']),
    /--version is not supported/,
  )
  assert.equal(fs.existsSync(log), false)
})
