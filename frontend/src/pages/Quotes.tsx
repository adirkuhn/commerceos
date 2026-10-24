import { useAuth } from '../auth/AuthContext'
import { NEW_RECORD, useRoutedEditor } from '../routing/useRoutedEditor'
import { useStackNavigation } from '../routing/useStackNavigation'
import { QuoteEditor } from '../quotes/QuoteEditor'
import { SavedQuotesList } from '../quotes/SavedQuotesList'

// Quotes: the saved list at /quotes, the editor at /quotes/new and
// /quotes/<id> (lines and discount in the query string).
export default function Quotes() {
  const { token } = useAuth()
  const { editingId, openNew, close } = useRoutedEditor('/quotes')
  const { open } = useStackNavigation()

  return (
    <div className="flex flex-col h-full">
      {editingId === null
        ? <SavedQuotesList token={token ?? ''} onOpen={open} onNew={openNew} />
        : <QuoteEditor key={editingId} token={token ?? ''} quoteId={editingId === NEW_RECORD ? null : editingId} onClose={close} />}
    </div>
  )
}
