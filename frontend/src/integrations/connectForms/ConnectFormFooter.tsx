import { useTranslation } from 'react-i18next'

// Error line + submit button shared by every connect form in the catalog.
export function ConnectFormFooter({ error, starting }: { error: string; starting: boolean }) {
  const { t } = useTranslation('integrations')
  return (
    <>
      {error && (
        <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>
      )}
      <button
        type="submit"
        disabled={starting}
        className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
      >
        {starting ? t('connect.connecting') : t('connect.submit')}
      </button>
    </>
  )
}
