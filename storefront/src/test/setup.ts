import '@testing-library/jest-dom/vitest'
import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'
import { resources } from '../i18n/resources'

// Components rendered on their own in tests get the English texts, like a
// store with the default language.
void i18next.use(initReactI18next).init({
  lng: 'en', fallbackLng: 'en', resources, defaultNS: 'shop', initAsync: false,
  returnEmptyString: false, interpolation: { escapeValue: false },
})
