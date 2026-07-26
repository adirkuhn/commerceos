import { useState, useRef, useEffect } from 'react'
import { ScanBarcode, CheckCircle, XCircle, Package, Truck, Volume2, VolumeX, User } from 'lucide-react'
import clsx from 'clsx'

interface PickItem {
  sku: string
  name: string
  barcode: string
  qty: number
  scanned: number
  location: string
}

interface PickOrder {
  id: string
  customer: string
  address: string
  items: PickItem[]
}

const PENDING_ORDERS: PickOrder[] = [
  {
    id: 'ORD-10040',
    customer: 'Sophie Martin',
    address: '14 Rue de la Paix, Paris 75001, FR',
    items: [
      { sku: 'WHP-001', name: 'Wireless Headphones Pro', barcode: '8714574000282', qty: 1, scanned: 0, location: 'A01-B03' },
      { sku: 'UCH-007', name: 'USB-C Hub 7-in-1', barcode: '8714574000299', qty: 2, scanned: 0, location: 'A02-C01' },
      { sku: 'LDL-SMT', name: 'LED Desk Lamp Smart', barcode: '8714574000367', qty: 1, scanned: 0, location: 'A04-B02' },
    ],
  },
  {
    id: 'ORD-10035',
    customer: 'Lars Jensen',
    address: 'Nørrebrogade 45, Copenhagen 2200, DK',
    items: [
      { sku: 'MPX-XL', name: 'Mouse Pad XL Gaming', barcode: '8714574000336', qty: 1, scanned: 0, location: 'A03-B01' },
      { sku: 'MKT-TKL', name: 'Mechanical Keyboard TKL', barcode: '8714574000305', qty: 1, scanned: 0, location: 'B01-C02' },
      { sku: 'WCM-4K', name: 'Webcam 4K Ultra', barcode: '8714574000312', qty: 1, scanned: 0, location: 'B02-C03' },
    ],
  },
]

type ScanState = 'idle' | 'success' | 'error'

export default function Picking() {
  const [activeOrderIdx, setActiveOrderIdx] = useState(0)
  const [orders, setOrders] = useState<PickOrder[]>(PENDING_ORDERS)
  const [scanInput, setScanInput] = useState('')
  const [scanState, setScanState] = useState<ScanState>('idle')
  const [lastScanned, setLastScanned] = useState<string | null>(null)
  const [soundEnabled, setSoundEnabled] = useState(true)
  const scanRef = useRef<HTMLInputElement>(null)

  const activeOrder = orders[activeOrderIdx]
  const allScanned = activeOrder?.items.every(i => i.scanned >= i.qty)

  useEffect(() => {
    scanRef.current?.focus()
  }, [activeOrderIdx])

  const handleScan = (barcode: string) => {
    const trimmed = barcode.trim()
    if (!trimmed) return

    const newOrders = orders.map((order, oi) => {
      if (oi !== activeOrderIdx) return order
      let found = false
      const items = order.items.map(item => {
        if (!found && item.barcode === trimmed && item.scanned < item.qty) {
          found = true
          return { ...item, scanned: item.scanned + 1 }
        }
        return item
      })
      if (found) {
        setScanState('success')
        setLastScanned(trimmed)
        setTimeout(() => setScanState('idle'), 1200)
      } else {
        setScanState('error')
        setTimeout(() => setScanState('idle'), 1200)
      }
      return found ? { ...order, items } : order
    })
    setOrders(newOrders)
    setScanInput('')
  }

  const totalItems = activeOrder?.items.reduce((s, i) => s + i.qty, 0) ?? 0
  const scannedItems = activeOrder?.items.reduce((s, i) => s + i.scanned, 0) ?? 0
  const progress = totalItems > 0 ? (scannedItems / totalItems) * 100 : 0

  return (
    <div className="flex h-full">
      {/* Order queue */}
      <div className="w-60 border-r dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <p className="text-xs font-semibold dark:text-gray-200 text-gray-800">Pending Picks</p>
          <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">{orders.length} orders</p>
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {orders.map((order, i) => {
            const done = order.items.every(x => x.scanned >= x.qty)
            const progress = order.items.reduce((s, x) => s + x.scanned, 0) / order.items.reduce((s, x) => s + x.qty, 0) * 100
            return (
              <button
                key={order.id}
                onClick={() => setActiveOrderIdx(i)}
                className={clsx(
                  'w-full text-left px-4 py-3 border-b dark:border-[#2D3748] border-gray-100 transition-colors',
                  i === activeOrderIdx
                    ? 'dark:bg-blue-600/10 bg-blue-50 border-l-2 border-l-blue-500'
                    : 'hover:dark:bg-[#1F2937] hover:bg-white',
                )}
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-medium dark:text-gray-200 text-gray-800">{order.id}</span>
                  {done && <CheckCircle size={12} className="text-green-400" />}
                </div>
                <p className="text-[10px] dark:text-gray-400 text-gray-500">{order.customer}</p>
                <div className="mt-2 h-1 dark:bg-[#2D3748] bg-gray-200 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 rounded-full transition-all"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Main pick area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className={clsx(
          'flex items-center gap-4 px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 transition-colors shrink-0',
          scanState === 'success' ? 'dark:bg-green-500/10 bg-green-50' : '',
          scanState === 'error' ? 'dark:bg-red-500/10 bg-red-50' : '',
        )}>
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-0.5">
              <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{activeOrder.id}</h2>
              <span className="text-xs dark:text-gray-500 text-gray-400">·</span>
              <span className="text-xs dark:text-gray-400 text-gray-600">{activeOrder.customer}</span>
            </div>
            <p className="text-xs dark:text-gray-500 text-gray-400">{activeOrder.address}</p>
          </div>

          <div className="flex items-center gap-3">
            {scanState === 'success' && (
              <div className="flex items-center gap-1.5 text-green-400 text-xs font-medium">
                <CheckCircle size={14} /> Scanned!
              </div>
            )}
            {scanState === 'error' && (
              <div className="flex items-center gap-1.5 text-red-400 text-xs font-medium">
                <XCircle size={14} /> Not found
              </div>
            )}
            <div className="flex items-center gap-1.5 dark:text-gray-400 text-gray-500 text-xs">
              <User size={12} /> Operator 1
            </div>
            <button onClick={() => setSoundEnabled(!soundEnabled)} className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300">
              {soundEnabled ? <Volume2 size={14} /> : <VolumeX size={14} />}
            </button>
          </div>
        </div>

        {/* Scan input — always prominent */}
        <div className="px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 shrink-0 dark:bg-[#111827]/80 bg-gray-50">
          <div className="flex items-center gap-3">
            <div className={clsx(
              'flex-1 flex items-center gap-3 px-4 py-3 rounded-xl border-2 transition-colors',
              scanState === 'success' ? 'border-green-500 dark:bg-green-500/5' : '',
              scanState === 'error' ? 'border-red-500 dark:bg-red-500/5' : '',
              scanState === 'idle' ? 'dark:border-blue-500 border-blue-400 dark:bg-[#1F2937] bg-white' : '',
            )}>
              <ScanBarcode size={20} className={clsx(
                scanState === 'success' ? 'text-green-400' :
                scanState === 'error' ? 'text-red-400' : 'text-blue-400',
              )} />
              <input
                ref={scanRef}
                type="text"
                value={scanInput}
                onChange={e => setScanInput(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleScan(scanInput)}
                placeholder="Scan barcode or type and press Enter..."
                className="flex-1 bg-transparent text-sm dark:text-gray-100 text-gray-900 outline-none dark:placeholder-gray-500 placeholder-gray-400 font-mono"
                autoFocus
              />
              {scanInput && (
                <button
                  onClick={() => handleScan(scanInput)}
                  className="px-3 py-1 bg-blue-600 text-white text-xs rounded-lg hover:bg-blue-700"
                >
                  Confirm
                </button>
              )}
            </div>
          </div>
          {lastScanned && (
            <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1.5 font-mono">Last: {lastScanned}</p>
          )}
        </div>

        {/* Progress */}
        <div className="px-6 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-xs dark:text-gray-400 text-gray-500">Progress</span>
            <span className="text-xs font-medium dark:text-gray-200 text-gray-800">{scannedItems} / {totalItems} items</span>
          </div>
          <div className="h-2 dark:bg-[#2D3748] bg-gray-200 rounded-full overflow-hidden">
            <div
              className={clsx(
                'h-full rounded-full transition-all duration-500',
                allScanned ? 'bg-green-500' : 'bg-blue-500',
              )}
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        {/* Item list */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {activeOrder.items.map(item => {
            const done = item.scanned >= item.qty
            const partial = item.scanned > 0 && item.scanned < item.qty
            return (
              <div
                key={item.sku}
                className={clsx(
                  'flex items-center gap-4 p-3 rounded-xl border transition-all',
                  done ? 'dark:bg-green-500/5 dark:border-green-500/20 border-green-200 bg-green-50' :
                  partial ? 'dark:bg-amber-500/5 dark:border-amber-500/20 border-amber-200 bg-amber-50' :
                  'dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200',
                )}
              >
                <div className={clsx(
                  'w-10 h-10 rounded-lg flex items-center justify-center shrink-0',
                  done ? 'bg-green-500/20' : 'dark:bg-[#2D3748] bg-gray-100',
                )}>
                  {done
                    ? <CheckCircle size={18} className="text-green-400" />
                    : <Package size={18} className="dark:text-gray-500 text-gray-400" />
                  }
                </div>
                <div className="flex-1 min-w-0">
                  <p className={clsx('text-sm font-medium', done ? 'dark:text-green-300 text-green-700' : 'dark:text-gray-100 text-gray-900')}>
                    {item.name}
                  </p>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{item.sku}</span>
                    <span className="text-[10px] dark:text-gray-600 text-gray-300">·</span>
                    <span className="text-[10px] dark:text-gray-500 text-gray-400">📍 {item.location}</span>
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <p className={clsx(
                    'text-lg font-bold tabular-nums',
                    done ? 'text-green-400' : partial ? 'text-amber-400' : 'dark:text-gray-200 text-gray-800',
                  )}>
                    {item.scanned}/{item.qty}
                  </p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">scanned</p>
                </div>
              </div>
            )
          })}
        </div>

        {/* Action footer */}
        {allScanned && (
          <div className="px-6 py-4 border-t dark:border-[#2D3748] border-gray-200 shrink-0 dark:bg-green-500/5 bg-green-50">
            <div className="flex items-center gap-3">
              <CheckCircle size={16} className="text-green-400" />
              <p className="text-sm font-medium text-green-400 flex-1">All items picked! Ready to pack.</p>
              <button className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-500 text-white text-xs font-medium hover:bg-green-600 transition-colors">
                <Truck size={14} /> Generate Shipment
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
