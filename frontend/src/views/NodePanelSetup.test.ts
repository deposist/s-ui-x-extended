import { renderToString } from '@vue/server-renderer'
import { createSSRApp } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { createVuetify } from 'vuetify'
import { VAlert, VBtn, VDivider, VExpansionPanel, VExpansionPanels, VExpansionPanelText, VProgressLinear, VTextField } from 'vuetify/components'
import NodePanelSetup from './NodePanelSetup.vue'
import { createNodeInvitation, parseNodeInvitation } from './nodeInvitation'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/plugins/httputil', () => ({ default: {} }))

describe('node role settings', () => {
  it('preserves an HTTPS reverse-proxy prefix and rejects unsupported or unsafe invitations', () => {
    const address = 'https://master.example/panel/branch/'
    const code = 'a'.repeat(32)
    const invitation = createNodeInvitation(address, code)
    expect(parseNodeInvitation(invitation)).toEqual({ masterUrl: address, code })
    expect(parseNodeInvitation(invitation.replace('v1 ', 'v2 '))).toBeNull()
    expect(parseNodeInvitation(invitation.replace('https:', 'http:'))).toBeNull()
    expect(parseNodeInvitation(invitation.replace('/branch/', '/branch/?secret=1'))).toBeNull()
    expect(parseNodeInvitation(invitation.replace('master.example', 'user@master.example'))).toBeNull()
  })
  it('renders a collapsed control with the localized panel title', async () => {
    const app = createSSRApp(NodePanelSetup)
    app.use(createVuetify({
      components: { VAlert, VBtn, VDivider, VExpansionPanel, VExpansionPanels, VExpansionPanelText, VProgressLinear, VTextField },
    }))
    app.config.globalProperties.$t = (key: string) => key

    const html = await renderToString(app)
    expect(html).toMatch(/<button[^>]*aria-expanded="false"[^>]*>[\s\S]*nodes\.setup\.title/)
    expect(html).not.toContain('actions.update')
  })
})
