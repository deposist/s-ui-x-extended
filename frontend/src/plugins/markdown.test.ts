import { describe, expect, it } from 'vitest'

import { parseMarkdownBlocks } from './markdown'

describe('parseMarkdownBlocks', () => {
  it('keeps headings, paragraphs, lists and code as structured blocks', () => {
    const blocks = parseMarkdownBlocks(`# Release Notes\n\nPlain **bold** and \`code\`.\n\n- first\n- second\n\n\`\`\`\nraw <b>html</b>\n\`\`\``)

    expect(blocks).toEqual([
      { type: 'heading', level: 1, inline: [{ type: 'text', text: 'Release Notes' }] },
      {
        type: 'paragraph',
        inline: [
          { type: 'text', text: 'Plain ' },
          { type: 'strong', text: 'bold' },
          { type: 'text', text: ' and ' },
          { type: 'code', text: 'code' },
          { type: 'text', text: '.' },
        ],
      },
      {
        type: 'list',
        ordered: false,
        items: [
          [{ type: 'text', text: 'first' }],
          [{ type: 'text', text: 'second' }],
        ],
      },
      { type: 'code', text: 'raw <b>html</b>' },
    ])
  })
})
