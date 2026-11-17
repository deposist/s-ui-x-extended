import { describe, expect, it } from 'vitest'
import { createSSRApp, defineComponent, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import Ssh from './Ssh.vue'

describe('SSH authentication editor', () => {
  it('preserves the remote account when selecting key authentication', () => {
    const data = { user: 'deploy', password: 'test-password' }
    Ssh.computed!.optionKey.set.call({ $props: { data }, usePath: 0 } as any, true)
    expect(data.user).toBe('deploy')
  })

  it('keeps the remote account field editable with key authentication', async () => {
    const app = createSSRApp(Ssh, { data: { user: 'deploy', private_key_path: '/tmp/test-key' } })
    const slot = defineComponent({ inheritAttrs: false, setup: (_, { slots }) => () => slots.default?.() })
    for (const name of ['VCard', 'VRow', 'VCol', 'VBtnToggle', 'VBtn', 'VTextarea', 'VMenu', 'VList', 'VListItem', 'VSwitch', 'VCardActions', 'VSpacer']) {
      app.component(name, slot)
    }
    app.component('VTextField', defineComponent({
      props: ['label', 'modelValue'],
      setup: props => () => h('field', { label: props.label, value: props.modelValue }),
    }))
    app.config.globalProperties.$t = (key: string) => key
    expect(await renderToString(app)).toContain('label="types.un" value="deploy"')
  })
})
