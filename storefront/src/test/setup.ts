import '@testing-library/jest-dom/vitest'
import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import { editorResources as resources } from '../editor/i18n/editorResources'

// Components rendered on their own in tests get the English texts, like a
// store with the default language.
void i18next.use(initReactI18next).init({
  lng: 'en', fallbackLng: 'en', resources, ns: ['shop', 'editor'], defaultNS: 'shop', initAsync: false,
  returnEmptyString: false, interpolation: { escapeValue: false },
})
