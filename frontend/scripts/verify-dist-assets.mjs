import { readdir } from 'node:fs/promises'
import { basename, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const distDir = fileURLToPath(new URL('../dist', import.meta.url))

const walk = async (dir) => {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const fullPath = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath))
    } else if (entry.isFile()) {
      files.push(fullPath)
    }
  }
  return files
}

const files = await walk(distDir)
const blocked = files
  .map(file => relative(distDir, file))
  .filter(file => {
    const name = basename(file)
    return name.startsWith('_') || name.startsWith('.')
  })

if (blocked.length > 0) {
  console.error('dist contains files that Go //go:embed directory walks skip:')
  for (const file of blocked) console.error(`- ${file}`)
  process.exitCode = 1
}
