import { useCallback, useEffect, useState } from 'react'
import { api, ensureAuth } from '../api/client'
import { usePolling } from '../hooks/usePolling'

export default function Cards() {
  const [cards, setCards] = useState<any[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [network, setNetwork] = useState('VISA')
  const [currency, setCurrency] = useState('USD')

  // Spend quote
  const [quoteCardId, setQuoteCardId] = useState('')
  const [quoteAmount, setQuoteAmount] = useState(0)
  const [quoteCurrency, setQuoteCurrency] = useState('EUR')
  const [quoteResult, setQuoteResult] = useState<any>(null)

  useEffect(() => { ensureAuth().then(() => loadCards()).catch(setError) }, [])

  const loadCards = useCallback(async () => {
    try {
      const data = await api.request('/cards')
      setCards(data.cards || [])
    } catch (e: any) {
      if (!cards.length) setError(e.message)
    } finally { setLoading(false) }
  }, [cards.length])

  const { refresh } = usePolling(loadCards, 10000, { immediate: false })

  const createCard = async () => {
    try {
      await api.request('/cards/virtual', {
        method: 'POST',
        body: JSON.stringify({ card_network: network, default_currency: currency }),
      })
      setShowCreate(false)
      refresh()
    } catch (err: any) { alert(err.message) }
  }

  const toggleFreeze = async (cardId: string, frozen: boolean) => {
    try {
      await api.request(`/cards/${cardId}/${frozen ? 'unfreeze' : 'freeze'}`, { method: 'POST' })
      refresh()
    } catch (err: any) { alert(err.message) }
  }

  const getQuote = async (cardId: string) => {
    try {
      const q = await api.request(`/cards/${cardId}/quote-spend`, {
        method: 'POST',
        body: JSON.stringify({
          transaction_amount: Math.round(quoteAmount * 100),
          transaction_currency: quoteCurrency,
          merchant_country: quoteCurrency === 'EUR' ? 'DE' : quoteCurrency === 'JPY' ? 'JP' : 'US',
          merchant_category_code: '5812',
        }),
      })
      setQuoteResult(q)
    } catch (err: any) { alert(err.message) }
  }

  const cardGradient = (c: any) => {
    if (c.card_network === 'MASTERCARD') return 'from-orange-600 via-red-500 to-yellow-500'
    if (c.card_network === 'UNIONPAY') return 'from-blue-600 via-cyan-500 to-teal-400'
    return 'from-emerald-700 via-teal-600 to-cyan-500' // VISA / default
  }

  if (error && !cards.length) {
    return <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">{error}</div>
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">💳 Cards</h2>
        <button onClick={() => setShowCreate(!showCreate)}
          className="px-4 py-2 bg-emerald-700 hover:bg-emerald-600 rounded-lg text-sm font-medium transition-colors">
          + New Card
        </button>
      </div>

      {/* Create Card Form */}
      {showCreate && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
          <h3 className="text-lg font-semibold mb-4">Issue New Virtual Card</h3>
          <div className="flex gap-4 items-end">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Network</label>
              <select value={network} onChange={e => setNetwork(e.target.value)}
                className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white">
                <option>VISA</option><option>MASTERCARD</option>
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Default Currency</label>
              <select value={currency} onChange={e => setCurrency(e.target.value)}
                className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white">
                <option>USD</option><option>EUR</option><option>GBP</option><option>JPY</option><option>CNY</option>
              </select>
            </div>
            <button onClick={createCard}
              className="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-sm">
              Issue Card
            </button>
            <button onClick={() => setShowCreate(false)}
              className="px-6 py-2 bg-gray-700 rounded-lg text-sm">Cancel</button>
          </div>
        </div>
      )}

      {/* Spend Quote Calculator */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
        <h3 className="text-lg font-semibold mb-4">🧮 Spend Quote Calculator</h3>
        <div className="flex gap-3 items-end flex-wrap">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Card</label>
            <select value={quoteCardId} onChange={e => setQuoteCardId(e.target.value)}
              className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white text-sm">
              <option value="">Select card...</option>
              {cards.map(c => (
                <option key={c.card_id} value={c.card_id}>****{c.pan_last4} ({c.card_network})</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Amount</label>
            <input type="number" value={quoteAmount || ''} onChange={e => setQuoteAmount(parseFloat(e.target.value) || 0)}
              className="w-32 bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white text-sm" placeholder="100.00" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Currency</label>
            <select value={quoteCurrency} onChange={e => setQuoteCurrency(e.target.value)}
              className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white text-sm">
              <option>EUR</option><option>JPY</option><option>GBP</option><option>USD</option><option>CNY</option>
            </select>
          </div>
          <button onClick={() => getQuote(quoteCardId)}
            disabled={!quoteCardId || !quoteAmount}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm disabled:opacity-50">
            Calculate
          </button>
        </div>
        {quoteResult && (
          <div className="mt-4 p-4 bg-gray-800/50 rounded-lg grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
            <div><span className="text-gray-500">Transaction</span>
              <p className="text-lg font-bold">{(quoteResult.transaction_amount/100).toFixed(2)} {quoteResult.transaction_currency}</p></div>
            <div><span className="text-gray-500">You'll Pay</span>
              <p className="text-lg font-bold text-white">{(quoteResult.debit_amount/100).toFixed(2)} {quoteResult.debit_currency}</p></div>
            <div><span className="text-gray-500">FX Rate</span>
              <p className="text-lg font-mono">{parseFloat(quoteResult.fx_rate || '1').toFixed(6)}</p></div>
            <div><span className="text-gray-500">Fee</span>
              <p className="text-lg font-bold text-amber-400">{(quoteResult.total_fee/100).toFixed(2)} {quoteResult.debit_currency}</p></div>
          </div>
        )}
      </div>

      {/* Card Grid */}
      {loading ? <div className="text-gray-500">Loading cards...</div> : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {cards.length === 0 && (
            <div className="col-span-2 text-center py-16 text-gray-600">
              <p className="text-4xl mb-4">💳</p>
              <p className="text-lg">No cards yet. Issue your first virtual card!</p>
            </div>
          )}
          {cards.map((c: any) => (
            <div key={c.card_id} className="relative group">
              {/* Wise-style Card */}
              <div className={`rounded-2xl p-6 bg-gradient-to-br ${cardGradient(c)} shadow-lg shadow-teal-900/20 min-h-[200px] flex flex-col justify-between`}>
                {/* Network */}
                <div className="flex justify-between items-start">
                  <div>
                    <p className="text-white/60 text-xs uppercase tracking-widest">Aspira Pay</p>
                    <p className="text-white/50 text-[10px]">{c.card_type} · {c.card_form}</p>
                  </div>
                  <span className="text-white/80 text-lg font-bold italic">{c.card_network}</span>
                </div>

                {/* Card Number */}
                <div>
                  <p className="text-white font-mono text-xl tracking-[0.25em] mb-2">
                    •••• •••• •••• {c.pan_last4}
                  </p>
                  <div className="flex gap-6 text-white/60 text-xs">
                    <span>VALID {String(c.expiry_month).padStart(2,'0')}/{String(c.expiry_year).slice(-2)}</span>
                  </div>
                </div>

                {/* Status badge */}
                <div className="flex justify-between items-end">
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                    c.status === 'ACTIVE' ? 'bg-white/20 text-white' :
                    c.status === 'FROZEN' ? 'bg-red-500/30 text-red-200' :
                    'bg-gray-500/30 text-gray-300'
                  }`}>{c.status}</span>
                  <span className="text-white/50 text-xs">{c.default_currency}</span>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex gap-2 mt-3">
                <button
                  onClick={() => { setQuoteCardId(c.card_id); setQuoteCurrency('EUR'); setQuoteAmount(100) }}
                  className="flex-1 px-3 py-1.5 bg-gray-800 hover:bg-gray-700 rounded-lg text-xs text-gray-300 transition-colors">
                  🧮 Quote
                </button>
                <button
                  onClick={() => toggleFreeze(c.card_id, c.status === 'FROZEN')}
                  className={`flex-1 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                    c.status === 'FROZEN'
                      ? 'bg-emerald-800/50 hover:bg-emerald-700/50 text-emerald-300'
                      : 'bg-red-900/30 hover:bg-red-800/30 text-red-400'
                  }`}>
                  {c.status === 'FROZEN' ? '🔓 Unfreeze' : '🔒 Freeze'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
