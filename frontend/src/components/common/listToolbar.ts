// List-page toolbar in the depot design: classic list pages stack their
// toolbar in separate rows (primary action, count + filters, more filters),
// each with its own border and background. Depot merges them into one band
// that wraps only when it doesn't fit -- without touching classic, which
// must stay pixel-identical.
//
//   <div className={LIST_TOOLBAR}>
//     <div className={clsx('...classic row...', LIST_TOOLBAR_ROW)}>
//       <button className={clsx('...', LIST_TOOLBAR_ACTION)}>New</button>
//     </div>
//     <div className={clsx('...classic row...', LIST_TOOLBAR_ROW)}>...</div>
//   </div>
//
// LIST_TOOLBAR is display:contents in classic, so the rows lay out exactly
// as if they were direct children of the page (mind a parent's space-y-*,
// whose child selector would no longer reach them). In depot it is the
// band, and each row dissolves (display:contents) so its items flow in it.
// min-h keeps a band holding only a count as tall as one with controls.

/** The wrapper: invisible in classic, the single toolbar band in depot. */
export const LIST_TOOLBAR =
  'contents depot:flex depot:flex-wrap depot:items-center depot:gap-x-4 depot:gap-y-2 depot:px-5 depot:py-3 depot:border-b depot:border-line depot:bg-surface depot:shrink-0 depot:min-h-[57px]'

/** A page whose classic toolbar is already one row: match the band's surface. */
export const LIST_TOOLBAR_SINGLE = 'depot:bg-surface'

/** A classic toolbar row whose items flow into the depot band. */
export const LIST_TOOLBAR_ROW = 'depot:contents'

/** The primary action: last in the depot band, pushed to the right. */
export const LIST_TOOLBAR_ACTION = 'depot:order-last depot:ml-auto'

/** The record count: first in the depot band, wherever its classic row is. */
export const LIST_TOOLBAR_LEAD = 'depot:order-first'

/** Something classic pushes right (ml-auto) that sits inline in depot. */
export const LIST_TOOLBAR_INLINE = 'depot:ml-0'

/** A page title repeating the topbar's: depot shows it only in the topbar. */
export const LIST_TOOLBAR_TITLE = 'depot:hidden'
