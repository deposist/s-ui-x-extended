import { lstat, readdir } from 'node:fs/promises'
import { basename, join, relative, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const distDir = fileURLToPath(new URL('../dist', import.meta.url))

const walk = async (dir) => {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const fullPath = join(dir, entry.name)
    const stat = await lstat(fullPath)
    if (stat.isSymbolicLink()) {
      throw new Error(`dist contains a symbolic link: ${relative(distDir, fullPath)}`)
    }
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath))
    } else if (entry.isFile()) {
      files.push(fullPath)
    } else {
      throw new Error(`dist contains a non-regular entry: ${relative(distDir, fullPath)}`)
    }
  }
  return files
}

const files = await walk(distDir)
const relativeFiles = files.map(file => relative(distDir, file))
const blocked = relativeFiles.filter(file => {
  const name = basename(file)
  return file === '..' || file.startsWith(`..${sep}`) || name.startsWith('_') || name.startsWith('.')
})

if (blocked.length > 0) {
  console.error('dist contains files that Go //go:embed directory walks skip or cannot safely embed:')
  for (const file of blocked) console.error(`- ${file}`)
  process.exitCode = 1
}

if (!relativeFiles.includes('index.html')) {
  console.error('dist does not contain the required index.html entrypoint')
  process.exitCode = 1
}

if (!relativeFiles.some(file => file !== 'index.html')) {
  console.error('dist does not contain any production assets besides index.html')
  process.exitCode = 1
}

if (!process.exitCode) console.log(`verified ${relativeFiles.length} production dist assets`)
