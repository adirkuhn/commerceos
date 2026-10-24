import i18next from 'i18next'
import { getCompanyProfile } from '../settings/company/api'
import { quoteDocumentHtml, type QuoteDocument } from './quoteDocument'

/**
 * Opens the printable quote in a new tab, leaving the editor as it is. The
 * tab opens before anything is awaited -- a tab opened after an await isn't
 * a direct result of the click and gets blocked as a pop-up.
 */
export async function openQuoteDocument(token: string, doc: Omit<QuoteDocument, 'seller'>): Promise<void> {
  const tab = window.open('', '_blank')
  if (!tab) throw new Error(i18next.t('quotes:print.popupBlocked'))
  tab.document.open()
  tab.document.write(`<p style="font:14px system-ui,sans-serif;padding:24px;color:#4b5563">${i18next.t('quotes:print.preparing')}</p>`)
  let seller
  try {
    seller = await getCompanyProfile(token)
  } catch (e) {
    tab.close()
    throw e
  }
  tab.document.open()
  tab.document.write(quoteDocumentHtml({ ...doc, seller }))
  tab.document.close()
}
