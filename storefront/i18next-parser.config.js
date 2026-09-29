// `npm run i18n:extract`: finds every t('key') in the code and syncs the
// translation files -- new keys are added (English text goes in en/*.json,
// other languages get "" until translated), unused keys are removed.
export default {
  locales: ['en', 'pt-BR', 'nl'],
  input: ['src/**/*.{ts,tsx}', '!src/**/__tests__/**'],
  output: 'src/i18n/locales/$LOCALE/$NAMESPACE.json',
  defaultNamespace: 'shop',
  namespaceSeparator: ':',
  keySeparator: '.',
  createOldCatalogs: false,
  sort: true,
  indentation: 2,
  // Keys only reached through a variable, e.g. t(`hint.${name}`), or written
  // from code by `npm run i18n:registry` (component and style texts).
  keepRemoved: [
    // Patterns see the whole key, or just "ns:group" when a group is gone.
    /^shop:hint(\.|$)/, /^shop:product\.(inStock|percentOff|chooseOptions)$/,
    /^editor:(components|categories|style|documents)(\.|$)/,
  ],
  defaultValue: (locale, _ns, key, value) => (locale === 'en' ? value || key : ''),
}
