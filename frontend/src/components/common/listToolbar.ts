// List-page toolbar: one band holding the count, filters and the primary
// action, wrapping only when it doesn't fit. List pages still mark up their
// toolbar in rows (primary action, count + filters, more filters); each row
// dissolves (display:contents) so its items flow into the band.
//
//   <div className={LIST_TOOLBAR}>
//     <div className={clsx('...row...', LIST_TOOLBAR_ROW)}>
//       <button className={clsx('...', LIST_TOOLBAR_ACTION)}>New</button>
//     </div>
//     <div className={clsx('...row...', LIST_TOOLBAR_ROW)}>...</div>
//   </div>
//
// The ! modifiers win over a row's own conflicting utilities (ml-auto,
// hidden) whatever Tailwind's order. min-h keeps a band holding only a
// count as tall as one with controls.

/** The band. */
export const LIST_TOOLBAR =
  'flex flex-wrap items-center gap-x-4 gap-y-2 px-5 py-3 border-b border-line bg-surface shrink-0 min-h-[57px]'

/** A page whose toolbar is already one row: match the band's surface. */
export const LIST_TOOLBAR_SINGLE = 'bg-surface'

/** A toolbar row whose items flow into the band. */
export const LIST_TOOLBAR_ROW = '!contents'

/** The primary action: last in the band, pushed to the right. */
export const LIST_TOOLBAR_ACTION = 'order-last !ml-auto'

/** The record count: first in the band, wherever its row is. */
export const LIST_TOOLBAR_LEAD = 'order-first'

/** Something a row pushes right (ml-auto) that sits inline in the band. */
export const LIST_TOOLBAR_INLINE = '!ml-0'

/** A page title repeating the topbar's: shown only in the topbar. */
export const LIST_TOOLBAR_TITLE = '!hidden'
