import { getThemePage, saveThemePage } from './api'
import { blocksToPuckData } from './puckConversion'
import { storefrontVisitURL } from './storefrontLink'
import { PAGE_SLUGS } from './types'
import type { PageSlug } from './types'
import type { SalesChannel } from '../integrations/types'

export interface ChannelMigrationResult {
  channelId: string
  channelName: string
  migratedSlugs: PageSlug[]
  previewUrl: string
}

// migrateChannelToPuck converts every one of a channel's own pages that
// still has real freeform content and isn't already on Puck -- a page
// nothing was ever saved to (the backend's own empty default, blocks: [])
// is skipped, there's nothing to bring over. Same conversion
// (blocksToPuckData) and non-blocking-per-page convention seedStarterTheme
// already uses for a brand-new channel; this is the same operation run
// once, by hand, against a channel that predates Puck-first seeding.
// previewUrl is the one, single, canonical link for this channel (see
// storefrontVisitURL) -- always returned so the caller has something to
// spot-check the result with, migrated or not.
export async function migrateChannelToPuck(token: string, channel: SalesChannel): Promise<ChannelMigrationResult> {
  const migratedSlugs: PageSlug[] = []
  for (const slug of PAGE_SLUGS) {
    try {
      const page = await getThemePage(token, channel.id, slug)
      if (page.editor_mode !== 'freeform' || page.blocks.length === 0) continue
      await saveThemePage(token, channel.id, slug, {
        canvas_width: page.canvas_width,
        blocks: page.blocks,
        custom_css: page.custom_css,
        editor_mode: 'puck',
        puck_data: blocksToPuckData(page.blocks),
      })
      migratedSlugs.push(slug)
    } catch {
      // Best-effort per page -- one page's failure (a transient network
      // error, say) must not stop the rest of this channel or any other
      // channel from migrating.
    }
  }
  return { channelId: channel.id, channelName: channel.name, migratedSlugs, previewUrl: storefrontVisitURL(channel.id) }
}

// migrateAllChannelsToPuck runs migrateChannelToPuck over every internal
// channel (the only type with a storefront/theme of its own) sequentially
// -- one at a time, not Promise.all, so this stays a small, predictable
// burst of requests against the same tenant rather than PAGE_SLUGS.length
// * channels.length requests all in flight together.
export async function migrateAllChannelsToPuck(token: string, channels: SalesChannel[]): Promise<ChannelMigrationResult[]> {
  const results: ChannelMigrationResult[] = []
  for (const channel of channels.filter(c => c.type === 'internal')) {
    results.push(await migrateChannelToPuck(token, channel))
  }
  return results
}
