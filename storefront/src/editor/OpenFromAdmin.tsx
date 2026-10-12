import { useTranslation } from 'react-i18next'

/** Shown when the editor is opened without an edit session. */
export function OpenFromAdmin() {
  const { t } = useTranslation('editor')
  return (
    <main className="mx-auto mt-32 max-w-md px-4 text-center">
      <h1 className="text-xl font-semibold">{t('open.title')}</h1>
      <p className="mt-2 text-ed-ink-3">{t('open.text')}</p>
    </main>
  )
}
