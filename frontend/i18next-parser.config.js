// `npm run i18n:extract`: finds every t('key') in the code and syncs the
// translation files per namespace (useTranslation('pms') -> pms.json).
// New keys are added (English text goes in en/*.json, other languages get
// "" until translated); unused keys are removed.
export default {
  locales: ['en', 'pt-BR', 'nl'],
  input: ['src/**/*.{ts,tsx}', '!src/**/__tests__/**', '!src/**/*.test.{ts,tsx}'],
  output: 'src/i18n/locales/$LOCALE/$NAMESPACE.json',
  defaultNamespace: 'common',
  namespaceSeparator: ':',
  keySeparator: '.',
  createOldCatalogs: false,
  sort: true,
  indentation: 2,
  // Shared texts are used from every module; keep them even when unused.
  // notifications:types.* are looked up by notification type at runtime
  // (see src/notifications/notificationText.ts) and mirror the backend's
  // email texts.
  keepRemoved: [/^common:/, /^notifications:types(\.|$)/],
  defaultValue: (locale, _ns, key, value) => (locale === 'en' ? value || key : ''),
}
