import type { RichTextSection as RichTextSectionType } from '@/types'

// dangerouslySetInnerHTML is safe here because every html value comes
// from src/config/demoData.ts (author-controlled fixtures), never from
// unsanitized user input -- a real CMS integration would sanitize on
// write, not on render.
export function RichTextSection({ section }: { section: RichTextSectionType }) {
  return (
    <section
      className="max-w-none space-y-4 text-foreground [&_a]:text-primary [&_a]:underline [&_h1]:text-3xl [&_h1]:font-bold [&_h2]:text-xl [&_h2]:font-semibold [&_p]:text-muted-foreground [&_p]:leading-relaxed"
      dangerouslySetInnerHTML={{ __html: section.html }}
    />
  )
}
