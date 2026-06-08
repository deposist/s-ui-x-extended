import { expect, type Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

export type E2EServerState = {
  baseURL: string
  backendURL: string
  username: string
  password: string
  dbDir: string
}

export const repoRoot = path.resolve(process.cwd(), '..')
export const phase6Dir = path.join(repoRoot, 'tests', 'baseline', 'phase6')
export const serverStatePath = path.join(phase6Dir, 'e2e-server', 'state.json')

export const readServerState = (): E2EServerState => {
  if (fs.existsSync(serverStatePath)) {
    return JSON.parse(fs.readFileSync(serverStatePath, 'utf8')) as E2EServerState
  }
  return {
    baseURL: process.env.SUI_E2E_BASE_URL ?? 'http://127.0.0.1:3000/app/',
    backendURL: process.env.SUI_E2E_BACKEND_URL ?? 'http://127.0.0.1:2095/app/',
    username: process.env.SUI_E2E_USERNAME ?? 'admin',
    password: process.env.SUI_E2E_PASSWORD ?? '',
    dbDir: process.env.SUI_E2E_DB_DIR ?? path.join(phase6Dir, 'e2e-db'),
  }
}

export const setEnglishLocale = async (page: Page) => {
  await page.addInitScript(() => {
    window.localStorage.setItem('locale', 'en')
  })
}

export const login = async (page: Page) => {
  const state = readServerState()
  testPasswordAvailable(state)
  await setEnglishLocale(page)
  await page.goto('login')
  const inputs = page.locator('input')
  await inputs.nth(0).fill(state.username)
  await inputs.nth(1).fill(state.password)
  await page.locator('button[type="submit"]').click()
  await expect.poll(async () => {
    const response = await page.request.get('api/settings')
    const body = await response.json().catch(() => ({ success: false }))
    return body.success === true
  }).toBe(true)
  await page.goto('')
}

export const csrfToken = async (page: Page) => {
  const response = await page.request.get('api/csrf')
  expect(response.ok()).toBeTruthy()
  const body = await response.json()
  expect(body.success).toBeTruthy()
  expect(typeof body.obj?.token).toBe('string')
  return body.obj.token as string
}

export const writeJSONArtifact = (relativePath: string, value: unknown) => {
  const target = path.join(phase6Dir, relativePath)
  fs.mkdirSync(path.dirname(target), { recursive: true })
  fs.writeFileSync(target, JSON.stringify(value, null, 2))
}

export const hasImportFixtures = () => (
  fs.existsSync(path.join(repoRoot, 'test-db', 'x-ui.db')) &&
  fs.existsSync(path.join(repoRoot, 'test-db', 's-ui.db'))
)

const testPasswordAvailable = (state: E2EServerState) => {
  if (!state.password) {
    throw new Error(`E2E password is empty; expected ${serverStatePath} from run-server.js or SUI_E2E_PASSWORD`)
  }
}
