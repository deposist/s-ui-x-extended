import { expect, test, type Locator, type Page } from '@playwright/test'

import { login } from './helpers'

// The Rules/DNS entity forms now render via FormShell (drawer in Nexus). Their
// views still mount those modals closed, so verify the pages render (the shell
// sidebar stays present => no blank-page crash from a closed FormShell drawer).
test('nexus rules and dns pages render without blanking', async ({ page }) => {
  test.setTimeout(60_000)

  await login(page)

  await page.goto('rules')
  await expect(page.getByRole('link', { name: 'Inbounds' })).toBeVisible()

  await page.goto('dns')
  await expect(page.getByRole('link', { name: 'Inbounds' })).toBeVisible()
})

// Repair flow fixtures. Route and DNS have structurally identical rule trees but
// different hosts: the route form is a FormShell drawer, the DNS form is still a
// raw v-dialog, so the container selector cannot be shared.
type RepairFixture = {
  view: string
  addLabel: string
  container: string
  rootPath: string
  testId: string
}

const routeFixture: RepairFixture = {
  view: 'rules',
  addLabel: 'Add Rule',
  container: '.nexus-drawer.v-navigation-drawer--active',
  rootPath: 'route.rules[0]',
  testId: 'rule-node',
}

const dnsFixture: RepairFixture = {
  view: 'dns',
  addLabel: 'Add Dns Rule',
  container: '.v-overlay--active',
  rootPath: 'dns.rules[0]',
  testId: 'dns-rule-node',
}

const nodeAt = (form: Locator, path: string) => form.locator(`[data-rule-path="${path}"]`)

// A node's own controls, never a descendant's. Shape/invert live in the node's
// first direct v-row and the Add control in its direct v-card-actions; scoping by
// :scope matters because a logical node nests whole child nodes that carry the
// same test ids.
const ownSwitchRoot = (node: Locator, testId: string, which: 'shape' | 'invert') =>
  node.locator(`:scope > .v-row [data-testid="${testId}-${which}"]`)

const ownSwitch = (node: Locator, testId: string, which: 'shape' | 'invert') =>
  ownSwitchRoot(node, testId, which).locator('input')

// Vuetify's switch <input> is a zero-opacity overlay, so a direct check() can be
// intercepted. Click the selection control and assert the bound state instead.
const turnOn = async (root: Locator) => {
  const input = root.locator('input')
  if (!(await input.isChecked())) {
    await root.locator('.v-selection-control__input').click()
  }
  await expect(input).toBeChecked()
}

const ownAddChild = (node: Locator, testId: string) =>
  node.locator(`:scope > .v-card-actions [data-testid="${testId}-add-child"]`)

// The delete control for child `index` sits in that child's wrapper header, ahead
// of the recursive node itself, so `.first()` inside the wrapper is its own.
const deleteChild = (node: Locator, testId: string, index: number) =>
  node
    .locator(':scope > .nested-children > .nested-child')
    .nth(index)
    .locator(`[data-testid="${testId}-delete-child"]`)
    .first()

// Reject needs neither an outbound nor a DNS server. The seeded config ships no
// outbounds and no DNS servers, so the default route/`local` action would leave a
// dangling reference and the whole-config save would fail for a reason unrelated
// to nested-rule repair.
const selectRejectAction = async (page: Page, form: Locator) => {
  // Vuetify's real <input role="combobox"> sits under .v-field__overlay, so a
  // click on the input is intercepted. Both forms label this select "Action" and
  // offer a "Reject" option, so the same helper drives route and DNS.
  await form.locator('.v-select').filter({ hasText: 'Action' }).first().locator('.v-field').click()
  await page.getByRole('option', { name: 'Reject', exact: true }).click()
}

const openAddForm = async (page: Page, fixture: RepairFixture) => {
  await page.goto(fixture.view)
  await page.getByRole('button', { name: fixture.addLabel, exact: true }).first().click()
  const form = page.locator(fixture.container)
  await expect(form).toBeVisible()
  return form
}

const reopenLastRule = async (page: Page, fixture: RepairFixture) => {
  // Rules are the last table on both views (rulesets / DNS servers come first),
  // and the rule just added is appended, so it is the last row.
  await page
    .locator('.nexus-data-table')
    .last()
    .locator('.nexus-data-table__actions')
    .last()
    .getByRole('button', { name: 'Edit' })
    .click()
  const form = page.locator(fixture.container)
  await expect(form).toBeVisible()
  return form
}

// Proves the whole point of the recursive rewrite: a condition-less node deep in
// the tree blocks the save, is reported at its own exact path while the form stays
// open and editable, and is repairable in place. It also pins the fork behaviour
// that a bare {} child is discarded on decode (so it does NOT repair the node)
// while an invert-only child is accepted even though it carries no conditions.
const runRepairFlow = async (page: Page, fixture: RepairFixture) => {
  const form = await openAddForm(page, fixture)
  await selectRejectAction(page, form)

  // A new rule opens simple, so the recursive nodes are not mounted yet. The root
  // shape switch belongs to the modal, not to a node; while the root is still
  // simple it is the only "Logical" switch on screen, so .first() is unambiguous.
  await turnOn(form.locator('.v-switch').filter({ hasText: 'Logical' }).first())

  // Depth 1: the modal seeds a new rule with one {} sub-rule, which now renders.
  const child = nodeAt(form, `${fixture.rootPath}.rules[0]`)
  await expect(child).toBeVisible()

  // Depth 2: converting the sub-rule to logical seeds it with one {} grandchild.
  await turnOn(ownSwitchRoot(child, fixture.testId, 'shape'))
  const grandchild = nodeAt(form, `${fixture.rootPath}.rules[0].rules[0]`)
  await expect(grandchild).toBeVisible()
  await expect(child).toHaveAttribute('data-node-shape', 'logical')

  // Delete the deepest logical node's sole child, leaving it condition-less.
  await deleteChild(child, fixture.testId, 0).click()
  await expect(grandchild).toHaveCount(0)

  // Save is blocked by the core, not by a local guess: the form stays open and the
  // issue is anchored on the exact path the core reported.
  const save = form.getByRole('button', { name: 'Save', exact: true })
  await save.click()
  const anchor = form.locator(`[data-rule-issue="${fixture.rootPath}.rules[0].rules"]`)
  await expect(anchor).toBeVisible()
  await expect(anchor).toContainText('sub-rule list is empty')
  await expect(form).toBeVisible()

  // The invalid node stays editable, so it can be repaired without reopening.
  // A bare {} child is discarded on decode, so it does not clear the issue.
  await ownAddChild(child, fixture.testId).click()
  const repaired = nodeAt(form, `${fixture.rootPath}.rules[0].rules[0]`)
  await expect(repaired).toBeVisible()
  await save.click()
  await expect(anchor).toBeVisible()
  await expect(form).toBeVisible()

  // An invert-only child carries no conditions yet is accepted by the fork, which
  // is exactly the shape a local "has a meaningful field" heuristic would reject.
  await turnOn(ownSwitchRoot(repaired, fixture.testId, 'invert'))
  await save.click()
  await expect(form).toHaveCount(0)

  // Persist the whole config and prove the nested tree survives a reload. Scoped
  // to the page toolbar: the closed form's own Save button stays in the DOM, so an
  // unscoped lookup is ambiguous.
  await page
    .locator('.nexus-toolbar__primary-actions')
    .getByRole('button', { name: 'Save', exact: true })
    .click()
  await expect.poll(async () => {
    const response = await page.request.get('api/config')
    const body = await response.json().catch(() => ({ success: false }))
    return body.success === true
  }).toBe(true)

  await page.reload()
  const reopened = await reopenLastRule(page, fixture)
  await expect(nodeAt(reopened, `${fixture.rootPath}.rules[0]`)).toHaveAttribute('data-node-shape', 'logical')
  const persistedLeaf = nodeAt(reopened, `${fixture.rootPath}.rules[0].rules[0]`)
  await expect(persistedLeaf).toBeVisible()
  await expect(ownSwitch(persistedLeaf, fixture.testId, 'invert')).toBeChecked()
}

test('nexus route rule: nested condition-less node blocks save, repairs in place, persists', async ({ page }) => {
  test.setTimeout(180_000)

  await login(page)
  await runRepairFlow(page, routeFixture)
})

test('nexus dns rule: nested condition-less node blocks save, repairs in place, persists', async ({ page }) => {
  test.setTimeout(180_000)

  await login(page)
  await runRepairFlow(page, dnsFixture)
})
