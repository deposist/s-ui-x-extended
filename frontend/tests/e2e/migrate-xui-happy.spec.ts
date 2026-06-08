import { expect, test, type Page } from '@playwright/test'

const mockAuthenticatedShell = async (page: Page) => {
  await page.addInitScript(() => {
    window.localStorage.setItem('locale', 'en')
  })
  await page.route('**/api/load**', async route => route.fulfill({
    json: {
      success: true,
      msg: '',
      obj: {
        onlines: { inbound: [], outbound: [], user: [] },
        config: {},
        inbounds: [],
        outbounds: [],
        services: [],
        endpoints: [],
        clients: [],
        tls: [],
      },
    },
  }))
  await page.route('**/api/csrf', async route => route.fulfill({
    json: { success: true, msg: '', obj: { token: 'cluster-h-csrf' } },
  }))
  await page.route('**/api/realtime/ws-token', async route => route.fulfill({
    json: { success: true, msg: '', obj: { token: 'cluster-h-ws-token' } },
  }))
  await page.route('**/api/logout', async route => route.fulfill({
    json: { success: true, msg: '', obj: null },
  }))
}

// XFAIL: пункты 43, 44, 45, 46 реестра; полный happy path требует test-db/x-ui.db и test-db/s-ui.db.
test.skip('upload synthetic db, build plan, apply, download JSON/Markdown report, and rollback', async () => {})

test('Issue43 shows inline apply failure on review step', async ({ page }) => {
  await mockAuthenticatedShell(page)
  await page.route('**/api/import-xui/plan', async route => route.fulfill({
    json: {
      success: true,
      msg: '',
      obj: {
        source: { hash: 'issue43-hash' },
        defaults: {},
        items: [
          {
            kind: 'inbound',
            srcId: '1',
            srcTag: 'demo-inbound',
            dstTag: 'demo-inbound',
            action: 'create',
            conflict: false,
            previewJson: { tag: 'demo-inbound' },
          },
        ],
      },
    },
  }))
  await page.route('**/api/import-xui/apply', async route => route.fulfill({
    json: { success: false, msg: 'synthetic apply failed', obj: null },
  }))

  await page.goto('migrate-xui')
  await expect(page).toHaveURL(/\/migrate-xui$/)
  await expect(page.getByText('Migrate from 3x-ui')).toBeVisible()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'x-ui.db',
    mimeType: 'application/octet-stream',
    buffer: Buffer.from('SQLite format 3\0'),
  })
  await page.getByRole('button', { name: 'Build plan' }).click()
  await page.getByRole('button', { name: 'Apply plan' }).click()

  await expect(page.getByTestId('migrate-xui-apply-error')).toBeVisible()
  await expect(page.getByTestId('migrate-xui-apply-error')).toContainText('synthetic apply failed')
  await expect(page.getByText('Review migration plan')).toBeVisible()
})

test('Issue44 waits for rollback database health before reload', async ({ page }) => {
  let healthCalls = 0
  await mockAuthenticatedShell(page)
  await page.route('**/api/import-xui/plan', async route => route.fulfill({
    json: {
      success: true,
      msg: '',
      obj: {
        source: { hash: 'issue44-hash' },
        defaults: {},
        items: [
          {
            kind: 'inbound',
            srcId: '1',
            srcTag: 'demo-inbound',
            dstTag: 'demo-inbound',
            action: 'create',
            conflict: false,
            previewJson: { tag: 'demo-inbound' },
          },
        ],
      },
    },
  }))
  await page.route('**/api/import-xui/apply', async route => route.fulfill({
    json: {
      success: true,
      msg: '',
      obj: {
        backupPath: 's-ui-pre-xui-import-test.db',
        summary: { inbounds: { created: 1 } },
      },
    },
  }))
  await page.route('**/api/import-xui/rollback', async route => route.fulfill({
    json: { success: true, msg: '', obj: null },
  }))
  await page.route('**/api/status**', async route => {
    healthCalls += 1
    await route.fulfill({
      json: { success: true, msg: '', obj: { db: { clients: 0 } } },
    })
  })

  await page.goto('migrate-xui')
  await expect(page).toHaveURL(/\/migrate-xui$/)
  await expect(page.getByText('Migrate from 3x-ui')).toBeVisible()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'x-ui.db',
    mimeType: 'application/octet-stream',
    buffer: Buffer.from('SQLite format 3\0'),
  })
  await page.getByRole('button', { name: 'Build plan' }).click()
  await page.getByRole('button', { name: 'Apply plan' }).click()
  await expect(page.getByText('Migration result')).toBeVisible()
  await page.getByRole('button', { name: 'Restore previous database' }).click()

  await expect.poll(() => healthCalls).toBeGreaterThan(0)
})

// XFAIL: пункт 45 реестра; generated admin password должен быть скрыт до явного reveal.
test.skip('generated admin password is shown once via reveal pattern, not raw JSON in DOM', async () => {})

test('Issue45 hides generated admin passwords until reveal and auto-clears them', async ({ page }) => {
  await page.addInitScript(() => {
    const nativeSetTimeout = window.setTimeout
    window.setTimeout = ((handler: TimerHandler, timeout?: number, ...args: any[]) => {
      const adjusted = typeof timeout === 'number' && timeout >= 60000 ? 1500 : timeout
      return nativeSetTimeout(handler, adjusted, ...args)
    }) as typeof window.setTimeout
  })
  await mockAuthenticatedShell(page)
  await page.route('**/api/import-xui/plan', async route => route.fulfill({
    json: {
      success: true,
      msg: '',
      obj: {
        source: { hash: 'issue45-hash' },
        defaults: {},
        items: [
          {
            kind: 'admin',
            srcId: '1',
            srcTag: 'migrated-admin',
            dstTag: 'migrated-admin',
            action: 'create',
            conflict: false,
            previewJson: { username: 'migrated-admin' },
          },
        ],
      },
    },
  }))
  await page.route('**/api/import-xui/apply', async route => route.fulfill({
    json: {
      success: true,
      msg: '',
      obj: {
        backupPath: 's-ui-pre-xui-import-test.db',
        summary: { admins: { created: 1 } },
        generatedAdmins: [
          { username: 'migrated-admin', password: 'issue45-secret-password' },
        ],
      },
    },
  }))

  await page.goto('migrate-xui')
  await expect(page).toHaveURL(/\/migrate-xui$/)
  await expect(page.getByText('Migrate from 3x-ui')).toBeVisible()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'x-ui.db',
    mimeType: 'application/octet-stream',
    buffer: Buffer.from('SQLite format 3\0'),
  })
  await page.getByRole('button', { name: 'Build plan' }).click()
  await page.getByRole('button', { name: 'Apply plan' }).click()
  await expect(page.getByText('Migration result')).toBeVisible()
  await expect(page.locator('body')).not.toContainText('issue45-secret-password')
  await expect(page.getByTestId('migrate-xui-generated-admins-hidden')).toBeVisible()

  await page.getByRole('button', { name: 'Reveal passwords' }).click()
  await expect(page.locator('body')).toContainText('issue45-secret-password')
  await expect(page.locator('body')).not.toContainText('issue45-secret-password', { timeout: 5000 })
  await expect(page.getByTestId('migrate-xui-generated-admins')).toBeHidden()
})

test('Issue46 sends reset_required adminMode when building a plan', async ({ page }) => {
  let planRequestBody = ''
  await mockAuthenticatedShell(page)
  await page.route('**/api/import-xui/plan', async route => {
    planRequestBody = route.request().postData() ?? ''
    await route.fulfill({
      json: {
        success: true,
        msg: '',
        obj: {
          source: { hash: 'issue46-hash' },
          defaults: { adminMode: 'reset_required' },
          items: [
            {
              kind: 'admin',
              srcId: '1',
              srcTag: 'migrated-admin',
              dstTag: 'migrated-admin',
              action: 'create',
              conflict: false,
              adminMode: 'reset_required',
              previewJson: { username: 'migrated-admin', mode: 'reset_required' },
            },
          ],
        },
      },
    })
  })

  await page.goto('migrate-xui')
  await expect(page).toHaveURL(/\/migrate-xui$/)
  await page.locator('input[type="file"]').setInputFiles({
    name: 'x-ui.db',
    mimeType: 'application/octet-stream',
    buffer: Buffer.from('SQLite format 3\0'),
  })
  await page.getByLabel('Admin import').click({ force: true })
  await page.getByRole('option', { name: 'Require password reset' }).click()
  await page.getByRole('button', { name: 'Build plan' }).click()

  await expect.poll(() => planRequestBody).toContain('adminMode')
  expect(planRequestBody).toContain('reset_required')
})

