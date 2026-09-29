import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ExternalLink, ArrowRight } from 'lucide-react'
import { RULES_PATH } from '../messagecenter/messageCenterPaths'

// Settings sections whose real controls live elsewhere. They say where,
// instead of showing toggles CommerceOS cannot enforce.

function Card({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-base font-semibold text-ink mb-1">{title}</h2>
        <p className="text-xs text-ink-4">{description}</p>
      </div>
      <div className="bg-raised rounded-xl border border-line p-5 space-y-3">{children}</div>
    </div>
  )
}

const linkClass = 'inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-accent-solid text-white text-sm font-medium hover:bg-accent-solid-hover'

// Sign-in, passwords, two-factor and sessions are lm2's (the identity
// provider every CommerceOS login goes through).
export function SecuritySection({ lm2Url }: { lm2Url: string }) {
  const { t } = useTranslation('settings')
  return (
    <Card title={t('sections.security')} description={t('security.subtitle')}>
      <p className="text-sm text-ink-2">
        {t('security.text')}
      </p>
      <a href={lm2Url} target="_blank" rel="noopener noreferrer" className={linkClass}>
        {t('security.open')} <ExternalLink size={14} aria-hidden />
      </a>
    </Card>
  )
}

// Automations are Message Center rules (event -> message), the one rule
// engine CommerceOS has.
export function AutomationSection() {
  const { t } = useTranslation('settings')
  return (
    <Card title={t('sections.automation')} description={t('automation.subtitle')}>
      <p className="text-sm text-ink-2">
        {t('automation.text')}
      </p>
      <Link to={RULES_PATH} className={linkClass}>
        {t('automation.open')} <ArrowRight size={14} aria-hidden />
      </Link>
    </Card>
  )
}
