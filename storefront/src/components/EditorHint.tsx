import { useTranslation } from 'react-i18next'
import { useRenderEnv } from '../runtime/RenderContext'
import type en from '../i18n/locales/en/shop.json'

export type HintKey = keyof (typeof en)['hint']

/** Shown instead of a component in the editor when it has nothing to show
 *  yet; renders nothing on the live store. */
export function EditorHint({ nodeId, className, hint }: { nodeId: string; className: string; hint: HintKey }) {
  const env = useRenderEnv()
  const { t } = useTranslation()
  return env.mode === 'editor' ? <div className={`cos-placeholder ${className}`} data-node={nodeId}>{t(`hint.${hint}`)}</div> : null
}
