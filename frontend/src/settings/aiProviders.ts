import type { TFunction } from 'i18next'
import type { AIConnection, AIProvider } from '../integrations/types'

export const AI_PROVIDERS: AIProvider[] = ['openai', 'anthropic', 'gemini', 'ollama', 'nvidia']

// Providers whose free tiers rate-limit hard enough that the adapter
// retries a list of fallback models on HTTP 429.
export const PROVIDERS_WITH_FALLBACK: ReadonlySet<AIProvider> = new Set<AIProvider>(['gemini', 'nvidia'])

export const FALLBACK_PLACEHOLDER: Partial<Record<AIProvider, string>> = {
  gemini: 'gemini-3.5-flash-lite, gemini-3.1-flash-lite',
  nvidia: 'minimaxai/minimax-m3.1-flash, nvidia/nemotron-3.5-super-vl-120b-a12b',
}

export const API_KEY_PLACEHOLDER: Partial<Record<AIProvider, string>> = { nvidia: 'nvapi-...' }

// The adapter's built-in default model, shown when it's worth knowing
// which model an empty field means.
export const DEFAULT_MODEL: Partial<Record<AIProvider, string>> = { nvidia: 'mistralai/mistral-large-4-1t-a52b' }

// Self-hosted: a server URL instead of an API key.
export const SELF_HOSTED: ReadonlySet<AIProvider> = new Set<AIProvider>(['ollama'])

export function providerLabel(t: TFunction<['settings', 'common']>, provider: AIProvider): string {
  switch (provider) {
    case 'openai': return 'OpenAI'
    case 'anthropic': return 'Anthropic (Claude)'
    case 'gemini': return 'Google Gemini'
    case 'ollama': return t('ai.ollama')
    case 'nvidia': return t('ai.nvidia')
  }
}

/** How a connection is named in the list: its label, else the provider. */
export function connectionName(t: TFunction<['settings', 'common']>, c: AIConnection): string {
  return c.label ? `${c.label} · ${providerLabel(t, c.provider)}` : providerLabel(t, c.provider)
}
