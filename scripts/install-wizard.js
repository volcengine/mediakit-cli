#!/usr/bin/env node

const { spawnSync } = require('node:child_process')
const fs = require('node:fs')
const path = require('node:path')

const pkg = require('../package.json')

const PACKAGE_NAME = pkg.name || '@volcengine/mediakit-cli'
const SKILLS_DIR = path.join(__dirname, '..', 'skills')

function parseArgs(argv) {
  const opts = {
    cliOnly: false,
    skillsOnly: false,
    skills: [],
    yes: false,
  }
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    switch (a) {
      case '--cli-only':
        opts.cliOnly = true
        break
      case '--skills-only':
        opts.skillsOnly = true
        break
      case '-y':
      case '--yes':
        opts.yes = true
        break
      case '-s':
      case '--skill': {
        const next = argv[i + 1]
        if (next && !next.startsWith('-')) {
          opts.skills.push(next)
          i++
        }
        break
      }
      case '--version':
        throw new Error('--version is not supported; install the npm package version you want directly')
      default:
        if (a.startsWith('--version=')) {
          throw new Error('--version is not supported; install the npm package version you want directly')
        } else if (a.startsWith('--skills=')) {
          opts.skills.push(
            ...a
              .slice('--skills='.length)
              .split(',')
              .map((s) => s.trim())
              .filter(Boolean),
          )
        }
        break
    }
  }
  return opts
}

function log(msg) {
  console.log(`[mediakit-cli install] ${msg}`)
}

function whichSync(cmd) {
  const pathValue = process.env.PATH || ''
  const extensions =
    process.platform === 'win32'
      ? (process.env.PATHEXT || '.EXE;.CMD;.BAT;.COM').split(';')
      : ['']
  for (const dir of pathValue.split(path.delimiter)) {
    if (!dir) continue
    for (const ext of extensions) {
      const candidate = path.join(dir, `${cmd}${ext}`)
      if (fs.existsSync(candidate)) {
        return true
      }
    }
  }
  return false
}

function runNpmInstall(target) {
  log(`installing ${target} via npm install -g`)
  const result = spawnSync('npm', ['install', '-g', target], {
    stdio: 'inherit',
  })
  if (result.error) {
    throw result.error
  }
  if (result.status !== 0) {
    throw new Error(
      `npm install -g ${target} failed with exit code ${result.status}`,
    )
  }
}

function runSkillsAdd(opts) {
  const args = ['-y', 'skills', 'add', SKILLS_DIR]
  if (opts.skills.length > 0) {
    for (const skill of opts.skills) {
      args.push('-s', skill)
    }
  }
  args.push('-g')
  if (opts.yes) {
    args.push('-y')
  }
  log(`installing skills via npx ${args.join(' ')}`)
  const result = spawnSync('npx', args, { stdio: 'inherit' })
  if (result.error) {
    throw result.error
  }
  if (result.status !== 0) {
    throw new Error(`npx skills add failed with exit code ${result.status}`)
  }
}

async function runInstallWizard(rawArgs) {
  const opts = parseArgs(rawArgs)

  if (!whichSync('npm')) {
    throw new Error('npm is required but not found in PATH')
  }

  if (!opts.skillsOnly) {
    runNpmInstall(PACKAGE_NAME)
  }

  if (opts.skillsOnly && !opts.cliOnly) {
    if (!whichSync('npx')) {
      throw new Error('npx is required to install skills but not found in PATH')
    }
    runSkillsAdd(opts)
  }

  log('done.')
}

module.exports = { runInstallWizard }

if (require.main === module) {
  runInstallWizard(process.argv.slice(2)).catch((error) => {
    console.error(`[mediakit-cli install] ${error.message}`)
    process.exit(1)
  })
}
