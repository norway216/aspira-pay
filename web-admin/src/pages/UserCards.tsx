import { useCallback, useEffect, useState } from 'react'
import { usePolling } from '../hooks/usePolling'

export default function UserCards({ userId, token }: { userId: string; token: string }) {
  const [cards, setCards] = useState<any[]>([])
  const [showApply, setShowApply] = useState(false)
  const [error, setError] = useState('')
  const h = { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' }
  const [app, setApp] = useState({ card_network: 'VISA', default_currency: 'USD', full_name: '', nationality: '', date_of_birth: '', address: '', document_type: 'passport', document_number: '' })

  const loadCards = useCallback(async () => {
    try {
      const r = await fetch('/api/v2/cards', { headers: h })
      const d = await r.json()
      setCards(d.cards || [])
    } catch (e) { console.error(e) }
  }, [token])

  useEffect(() => { loadCards() }, [loadCards])
  const { refresh } = usePolling(loadCards, 8000, { immediate: false })

  const apply = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const r = await fetch('/api/v2/cards/apply', { method: 'POST', headers: h, body: JSON.stringify(app) })
      const d = await r.json()
      if (!r.ok) throw new Error(d.error || 'Failed')
      setShowApply(false)
      refresh()
    } catch (err: any) { setError(err.message) }
  }

  const cancelCard = async (cardId: string) => {
    if (!confirm('Cancel this card? This cannot be undone.')) return
    try {
      await fetch(`/api/v2/cards/${cardId}/cancel`, { method: 'POST', headers: h })
      refresh()
    } catch (err: any) { alert(err.message) }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold">My Cards</h2>
          <p className="text-sm text-gray-500 mt-1">{cards.length}/5 cards issued</p>
        </div>
        <button onClick={() => setShowApply(!showApply)} disabled={cards.length >= 5}
          className="px-4 py-2 bg-emerald-700 hover:bg-emerald-600 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {cards.length >= 5 ? 'Max 5 Cards' : '+ Apply for Card'}
        </button>
      </div>

      {/* Application Form */}
      {showApply && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
          <h3 className="text-lg font-semibold mb-4">Card Application — KYC Required</h3>
          {error && <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-red-400 text-sm mb-4">{error}</div>}
          <form onSubmit={apply} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div><label className="block text-sm text-gray-400 mb-1">Full Legal Name *</label><input type="text" value={app.full_name} onChange={e => setApp({...app, full_name: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
              <div><label className="block text-sm text-gray-400 mb-1">Nationality *</label><input type="text" value={app.nationality} onChange={e => setApp({...app, nationality: e.target.value})} placeholder="US" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
              <div><label className="block text-sm text-gray-400 mb-1">Date of Birth *</label><input type="text" value={app.date_of_birth} onChange={e => setApp({...app, date_of_birth: e.target.value})} placeholder="1990-01-15" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
              <div><label className="block text-sm text-gray-400 mb-1">Address</label><input type="text" value={app.address} onChange={e => setApp({...app, address: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" /></div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Card Network</label>
                <select value={app.card_network} onChange={e => setApp({...app, card_network: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white">
                  <option>VISA</option><option>MASTERCARD</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Currency</label>
                <select value={app.default_currency} onChange={e => setApp({...app, default_currency: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white">
                  <option>USD</option><option>EUR</option><option>GBP</option><option>JPY</option>
                </select>
              </div>
            </div>
            <div className="flex gap-3"><button type="submit" className="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-sm">Submit Application</button><button type="button" onClick={() => setShowApply(false)} className="px-6 py-2 bg-gray-700 rounded-lg text-sm">Cancel</button></div>
          </form>
        </div>
      )}

      {/* Card Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {cards.map((c: any) => (
          <div key={c.card_id} className="relative">
            <div className={`rounded-2xl p-6 bg-gradient-to-br ${c.card_network === 'MASTERCARD' ? 'from-orange-600 via-red-500 to-yellow-500' : 'from-emerald-700 via-teal-600 to-[#E07373]'} shadow-lg min-h-[180px] flex flex-col justify-between`}>
              <div className="flex justify-between items-start">
                <div><p className="text-white/60 text-xs uppercase tracking-widest">Aspira Pay</p><p className="text-white/50 text-[10px]">{c.card_type} · {c.card_form}</p></div>
                <span className="text-white/80 text-lg font-bold italic">{c.card_network}</span>
              </div>
              <div><p className="text-white font-mono text-xl tracking-[0.25em] mb-2">•••• •••• •••• {c.pan_last4}</p>
                <div className="flex gap-4 text-white/60 text-xs"><span>VALID {String(c.expiry_month).padStart(2,'0')}/{String(c.expiry_year).slice(-2)}</span></div>
              </div>
              <div className="flex justify-between items-end">
                <span className={`text-xs px-2 py-0.5 rounded-full ${c.status==='ACTIVE'?'bg-white/20 text-white':'bg-red-500/30 text-red-200'}`}>{c.status}</span>
                <span className="text-white/50 text-xs">{c.default_currency}</span>
              </div>
            </div>
            {c.status === 'ACTIVE' && (
              <button onClick={() => cancelCard(c.card_id)} className="mt-2 w-full py-1.5 bg-red-900/30 hover:bg-red-800/30 text-red-400 rounded-lg text-xs transition-colors">Cancel Card</button>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
