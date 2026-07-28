import { realpathSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const frontendDir = realpathSync.native(fileURLToPath(new URL('..', import.meta.url)))
const vitestEntry = './node_modules/vitest/vitest.mjs'
const args = process.argv.slice(2)
if (args.length === 0) args.push('run')

const result = spawnSync(process.execPath, [vitestEntry, ...args], {
  cwd: frontendDir,
  stdio: 'inherit',
})

if (result.error) throw result.error
process.exit(result.status ?? 1)
