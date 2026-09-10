import { Check, Palette, Sun, Moon, Monitor, Languages } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator,
} from '@/components/ui/DropdownMenu'
import { THEMES } from '@/themes/themes'
import { LOCALES, LOCALE_LABELS } from '@/i18n/dictionaries'
import { useUIStore, type ColorMode } from '@/stores/uiStore'
import { cn } from '@/lib/utils'

const COLOR_MODES: { id: ColorMode; label: string; icon: typeof Sun }[] = [
  { id: 'light', label: 'Light', icon: Sun },
  { id: 'dark', label: 'Dark', icon: Moon },
  { id: 'system', label: 'System', icon: Monitor },
]

// A single settings menu covers theme, dark/light/system, and language --
// each already a real, working Zustand action (see stores/uiStore.ts) and
// ThemeProvider/I18nProvider consumer; this is just the missing UI to
// reach them, so it stays one small menu rather than three separate
// header controls.
export function SettingsMenu() {
  const themeId = useUIStore(s => s.themeId)
  const setThemeId = useUIStore(s => s.setThemeId)
  const colorMode = useUIStore(s => s.colorMode)
  const setColorMode = useUIStore(s => s.setColorMode)
  const locale = useUIStore(s => s.locale)
  const setLocale = useUIStore(s => s.setLocale)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Theme and language settings">
          <Palette className="h-5 w-5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel>Theme</DropdownMenuLabel>
        {THEMES.map(theme => (
          <DropdownMenuItem key={theme.id} onSelect={() => setThemeId(theme.id)}>
            <span className={cn('flex w-full items-center justify-between', themeId === theme.id && 'font-medium')}>
              {theme.name}
              {themeId === theme.id && <Check className="h-3.5 w-3.5" />}
            </span>
          </DropdownMenuItem>
        ))}

        <DropdownMenuSeparator />
        <DropdownMenuLabel>Appearance</DropdownMenuLabel>
        {COLOR_MODES.map(mode => (
          <DropdownMenuItem key={mode.id} onSelect={() => setColorMode(mode.id)}>
            <span className={cn('flex w-full items-center gap-2', colorMode === mode.id && 'font-medium')}>
              <mode.icon className="h-3.5 w-3.5" />
              {mode.label}
              {colorMode === mode.id && <Check className="ml-auto h-3.5 w-3.5" />}
            </span>
          </DropdownMenuItem>
        ))}

        <DropdownMenuSeparator />
        <DropdownMenuLabel className="flex items-center gap-1.5"><Languages className="h-3 w-3" /> Language</DropdownMenuLabel>
        {LOCALES.map(l => (
          <DropdownMenuItem key={l} onSelect={() => setLocale(l)}>
            <span className={cn('flex w-full items-center justify-between', locale === l && 'font-medium')}>
              {LOCALE_LABELS[l]}
              {locale === l && <Check className="h-3.5 w-3.5" />}
            </span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
