import { useCallback, useEffect, useState } from 'react'
import { usePolling } from '../hooks/usePolling'

export default function UserPayments({ userId, token }: { userId: string; token: string }) {
  const [payments, setPayments] = useState<any[]>([])
  const [showNew, setShowNew] = useState(false)
  const [error, setError] = useState('')
  const h = { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' }
  const [form, setForm] = useState({ sender_user_id: userId, receiver_user_id: '', source_currency: 'USD', target_currency: 'EUR', source_amount: 0 })

  const loadPayments = useCallback(async () => {
    try {
      const r = await fetch('/api/v2/payments', { headers: h })
      const d = await r.json()
      setPayments(d.orders || [])
    } catch (e) { console.error(e) }
  }, [token])

  useEffect(() => { loadPayments() }, [loadPayments])
  const { refresh } = usePolling(loadPayments, 3000, { immediate: false })

  const createPayment = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    try {
      const r = await fetch('/api/v2/payments', {
        method: 'POST', headers: { ...h, 'Idempotency-Key': `upay_${Date.now()}` },
        body: JSON.stringify({ ...form, source_amount: Math.round(form.source_amount * 100), purpose: 'personal', country_from: 'US', country_to: 'DE' }),
      })
      const d = await r.json()
      if (!r.ok) throw new Error(d.error || 'Failed')
      setShowNew(false); setForm({ ...form, source_amount: 0 })
      refresh()
    } catch (err: any) { setError(err.message) }
  }

  const statusColor = (s: string) => {
    if (s === 'CLOSED' || s === 'COMPLETED' || s === 'PAYMENT_CONFIRMED') return 'text-green-400'
    if (s?.includes('FAILED') || s?.includes('REJECTED')) return 'text-red-400'
    if (s === 'CANCELLED' || s === 'REFUNDED') return 'text-yellow-400'
    return 'text-blue-400'
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Payments</h2>
        <button onClick={() => setShowNew(!showNew)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition-colors">+ New Payment</button>
      </div>

      {showNew && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
          <h3 className="text-lg font-semibold mb-4">Send Money</h3>
          {error && <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-red-400 text-sm mb-4">{error}</div>}
          <form onSubmit={createPayment} className="space-y-4">
            <div><label className="block text-sm text-gray-400 mb-1">Recipient ID</label><input type="text" value={form.receiver_user_id} onChange={e => setForm({...form, receiver_user_id: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
            <div className="grid grid-cols-3 gap-4">
              <div><label className="block text-sm text-gray-400 mb-1">From</label><select value={form.source_currency} onChange={e => setForm({...form, source_currency: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"><option>USD</option><option>EUR</option><option>GBP</option><option>JPY</option></select></div>
              <div><label className="block text-sm text-gray-400 mb-1">To</label><select value={form.target_currency} onChange={e => setForm({...form, target_currency: e.target.value})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"><option>EUR</option><option>USD</option><option>GBP</option><option>JPY</option></select></div>
              <div><label className="block text-sm text-gray-400 mb-1">Amount</label><input type="number" step="0.01" value={form.source_amount||''} onChange={e => setForm({...form, source_amount: parseFloat(e.target.value)||0})} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
            </div>
            <div className="flex gap-3"><button type="submit" className="px-6 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm">Send</button><button type="button" onClick={() => setShowNew(false)} className="px-6 py-2 bg-gray-700 rounded-lg text-sm">Cancel</button></div>
          </form>
        </div>
      )}

      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-left">
          <th className="p-3 text-gray-500 text-xs">ID</th><th className="p-3 text-gray-500 text-xs">Pair</th><th className="p-3 text-gray-500 text-xs">Amount</th><th className="p-3 text-gray-500 text-xs">Fee</th><th className="p-3 text-gray-500 text-xs">Status</th><th className="p-3 text-gray-500 text-xs">Date</th>
        </tr></thead><tbody>
          {payments.map((p: any) => (
            <tr key={p.payment_id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
              <td className="p-3 font-mono text-xs text-gray-400">{p.payment_id?.substring(0,16)}...</td>
              <td className="p-3">{p.source_currency}→{p.target_currency}</td>
              <td className="p-3">{(p.source_amount/100).toFixed(2)}</td>
              <td className="p-3 text-gray-500">{(p.fee_amount/100).toFixed(2)}</td>
              <td className={`p-3 text-xs font-medium ${statusColor(p.status)}`}>{p.status}</td>
              <td className="p-3 text-gray-500 text-xs">{new Date(p.created_at).toLocaleDateString()}</td>
            </tr>
          ))}
        </tbody></table>
      </div>
    </div>
  )
}
