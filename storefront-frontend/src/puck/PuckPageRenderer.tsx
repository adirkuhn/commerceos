import { Render, type Data } from '@puckeditor/core'
import { buildPuckConfig } from './config'
import type { PageSlug } from '../types'

interface Props {
  slug: PageSlug
  data: Data
}

// Default export (not named) so CanvasPage.tsx can React.lazy()-import it --
// keeps @puckeditor/core's whole runtime (dnd-kit, tiptap, etc) out of the
// main bundle every shopper downloads, even on a page that never uses
// editor_mode 'puck'. See CanvasPage.tsx's own Suspense wrapper.
export default function PuckPageRenderer({ slug, data }: Props) {
  return <Render config={buildPuckConfig(slug)} data={data} />
}
