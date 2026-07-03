const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const test = require('node:test')

const { install } = require('./install')

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

test('postinstall installs skills from the npm package skills directory when binary already exists', async (t) => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'mediakit-postinstall-'))
  const bin = path.join(temp, 'bin')
  fs.mkdirSync(bin)
  const log = path.join(temp, 'calls.jsonl')

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
  const oldSkip = process.env.MEDIAKIT_CLI_SKIP_DOWNLOAD
  process.env.PATH = `${bin}${path.delimiter}${oldPath || ''}`
  delete process.env.MEDIAKIT_CLI_SKIP_DOWNLOAD
  t.after(() => {
    process.env.PATH = oldPath
    if (oldSkip === undefined) {
      delete process.env.MEDIAKIT_CLI_SKIP_DOWNLOAD
    } else {
      process.env.MEDIAKIT_CLI_SKIP_DOWNLOAD = oldSkip
    }
  })

  const output = await captureLogs(t, () => install())

  const calls = fs
    .readFileSync(log, 'utf8')
    .trim()
    .split('\n')
    .map((line) => JSON.parse(line))
  assert.equal(calls.length, 1)
  assert.deepEqual(calls[0].args, [
    '-y',
    'skills',
    'add',
    path.join(__dirname, '..', 'skills'),
    '-g',
    '-y',
  ])
  assert.match(output, /Installing skills \.\.\./)
  assert.match(output, /✓ Skills installed/)
})
