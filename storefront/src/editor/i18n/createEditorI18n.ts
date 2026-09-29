import type { i18n } from 'i18next'
import { createI18n } from '../../i18n/createI18n'
import type { Locale } from '../../i18n/locale'
import { editorResources } from './editorResources'

/** The editor's i18next instance: its own texts plus the shop's. */
export const createEditorI18n = (locale: Locale): i18n => createI18n(locale, editorResources)
