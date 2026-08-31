import globals from 'globals';

/** @type {import('xo').FlatXoConfig} */
const xoConfig = [
  {
    space: true,
    prettier: 'compat',
  },
  {
    ignores: [
      'desktop/node_modules/**',
      'site/**',
      'scripts/**',
      'e2e/**',
      'playwright.config.js',
      'xo.config.js',
    ],
  },
  {
    files: ['desktop/src/**/*.test.js'],
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  },
  {
    files: ['desktop/src/**/*.js'],
    ignores: ['desktop/src/**/*.test.js'],
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
    rules: {
      'no-await-in-loop': 'off',
      'unicorn/catch-error-name': 'off',
      'unicorn/prefer-global-this': 'off',
      'unicorn/prefer-query-selector': 'off',
      'unicorn/prevent-abbreviations': 'off',
    },
  },
];

export default xoConfig;
