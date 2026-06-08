import js from '@eslint/js'
import vue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'

export default [
  {
    ignores: ['dist/**', 'node_modules/**'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...vue.configs['flat/essential'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        extraFileExtensions: ['.vue'],
        ecmaVersion: 'latest',
        sourceType: 'module',
      },
    },
  },
  {
    rules: {
      'vue/multi-word-component-names': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
      'no-unused-vars': 'off',
      'no-prototype-builtins': 'off',
      'no-undef': 'off',
      'no-unsafe-optional-chaining': 'off',
      'prefer-const': 'off',
      'no-var': 'off',
      'no-useless-assignment': 'off',
      'no-case-declarations': 'off',
      '@typescript-eslint/no-unused-expressions': 'off',
      'vue/no-mutating-props': 'off',
      'vue/valid-v-for': 'off',
      'vue/require-v-for-key': 'off',
      'vue/no-side-effects-in-computed-properties': 'off',
      'vue/no-v-text-v-html-on-component': 'off',
      'vue/valid-v-slot': 'off',
      'vue/no-unused-components': 'off',
      'vue/no-reserved-component-names': 'off',
      'vue/no-unused-vars': 'off',
      'vue/no-use-v-if-with-v-for': 'off',
    },
  },
]
