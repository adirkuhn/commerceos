import { useEffect, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { useStorefrontAPI } from '../../api/ApiContext'
import type { OrderStatus } from '../../api/storefront'
import { useShopFormat } from '../../i18n/format'
import { orderStatusLabel } from '../../i18n/orderStatus'
import { useRenderEnv } from '../../runtime/RenderContext'
import { rememberedEmailFor, useOrderNumber } from '../../runtime/page/OrderContext'
import { EditorHint } from '../EditorHint'
import type { RendererProps } from '../types'
import type { OrderStatusProps } from './checkout'

/** Shows the order in the URL. The shopper who just placed it sees it at
 *  once; anyone else must give the order's email (the API requires it). */
export function OrderStatusView({ node, props, className }: RendererProps<OrderStatusProps>) {
  const api = useStorefrontAPI()
  const env = useRenderEnv()
  const orderNumber = useOrderNumber()
  const { t } = useTranslation()
  const { price } = useShopFormat()
  const [order, setOrder] = useState<OrderStatus | null>(null)
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')

  const lookup = (address: string) => {
    if (!orderNumber) return
    setError('')
    api.lookupOrder(orderNumber, address).then(setOrder, () => setError(t('order.notFound')))
  }

  useEffect(() => {
    if (env.mode !== 'live' || !orderNumber) return
    const remembered = rememberedEmailFor(orderNumber)
    if (remembered) lookup(remembered)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [orderNumber])

  if (env.mode === 'editor' || !orderNumber) {
    return <EditorHint nodeId={node.id} className={className} hint="order" />
  }
  return (
    <section className={`cos-order-status ${className}`} data-node={node.id}>
      {props.title && <h1 className="cos-heading">{props.title}</h1>}
      <p><Trans i18nKey="order.number" values={{ number: orderNumber }} components={{ strong: <strong /> }} /></p>
      {order ? (
        <dl>
          <div><dt>{t('common.status')}</dt><dd>{orderStatusLabel(t, order.status)}</dd></div>
          <div><dt>{t('common.total')}</dt><dd>{price(order.total_minor_units, order.currency_code)}</dd></div>
        </dl>
      ) : (
        <form onSubmit={e => { e.preventDefault(); lookup(email) }}>
          <label htmlFor={`${node.id}-email`} className="cos-label">{t('order.lookupPrompt')}</label>
          <input id={`${node.id}-email`} type="email" required value={email} onChange={e => setEmail(e.target.value)} />
          <button type="submit" className="cos-button cos-button-outline">{t('order.show')}</button>
        </form>
      )}
      {error && <p role="alert" className="cos-error">{error}</p>}
    </section>
  )
}
