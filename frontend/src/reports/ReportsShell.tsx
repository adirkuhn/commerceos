import type { ReactNode } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useApp } from '../context/AppContext'
import { REPORT_SECTIONS } from '../components/layout/navConfig'

// Frame around every report page. The sidebar has a single Reports link,
// so the reports list themselves here, grouped by subject -- the same
// two-column pattern as Settings.
export function ReportsShell({ children }: { children: ReactNode }) {
  const { activeNav, setActiveNav } = useApp()
  const { t } = useTranslation('layout')

  return (
    <div className="flex h-full overflow-hidden">
      <nav aria-label={t('group.reports')} className="w-56 shrink-0 border-r border-line bg-sunken overflow-y-auto py-3 px-2">
        {REPORT_SECTIONS.map(section => (
          <section key={section.id} className="mb-3">
            <h2 className="px-2.5 pb-1 text-xs font-semibold text-ink-3">{section.label}</h2>
            {section.items.map(report => {
              const active = activeNav === report.id
              return (
                <button
                  key={report.id}
                  onClick={() => setActiveNav(report.id)}
                  aria-current={active ? 'page' : undefined}
                  className={clsx(
                    'w-full px-2.5 py-1.5 rounded-lg text-left text-body transition-colors',
                    active ? 'bg-accent/10 text-accent font-medium' : 'text-ink-2 hover:bg-hover',
                  )}
                >
                  {report.label}
                </button>
              )
            })}
          </section>
        ))}
      </nav>
      <div className="flex-1 min-w-0 overflow-auto">{children}</div>
    </div>
  )
}
