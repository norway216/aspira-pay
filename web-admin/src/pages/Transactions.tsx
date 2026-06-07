import { useCallback, useEffect, useState } from 'react'
import { api, ensureAuth } from '../api/client'
import { usePolling } from '../hooks/usePolling'

export default function Transactions() {
  const [orders, setOrders] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({
    sender_user_id: '',
    receiver_user_id: '',
    source_currency: 'USD',
    target_currency: 'JPY',
    source_amount: 0,
  })

  // Initial auth
  useEffect(() => {
    ensureAuth().then(() => setLoading(false)).catch(e => {
      setError(e.message)
      setLoading(false)
    })
  }, [])

  // Auto-refresh payment list every 2 seconds (payments update asynchronously via Saga)
  const loadPayments = useCallback(async () => {
    try {
      const data = await api.getPayments()
      setOrders(data.orders || [])
      setError('')
    } catch (e: any) {
      if (!orders.length) setError(e.message)
    }
  }, [orders.length])

  const { refresh } = usePolling(loadPayments, 2000, { immediate: true })

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      await api.createPayment({
        ...form,
        source_amount: Math.round(form.source_amount * 100),
      })
      setShowForm(false)
      setForm({ ...form, source_amount: 0 })
      // Immediately refresh to show the new payment
      await refresh()
    } catch (err: any) {
      alert(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  const statusColor = (status: string) => {
    switch (status) {
      case 'COMPLETED': return 'text-green-400'
      case 'FAILED': case 'REJECTED': return 'text-red-400'
      case 'CANCELLED': case 'REFUNDED': return 'text-yellow-400'
      case 'MANUAL_REVIEW': return 'text-orange-400'
      default: return 'text-blue-400'
    }
  }

  if (error && !orders.length) {
    return (
      <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">
        {error}
        <button onClick={refresh} className="ml-3 px-3 py-1 bg-red-800 rounded text-sm">Retry</button>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold">Transactions</h2>
          <span className="text-xs text-gray-600 bg-gray-800 px-2 py-0.5 rounded">
            Auto-refresh 2s
          </span>
        </div>
        <div className="flex gap-2">
          <button
            onClick={refresh}
            className="px-3 py-2 bg-gray-800 hover:bg-gray-700 rounded-lg text-sm text-gray-400 hover:text-white transition-colors"
            title="Manual refresh"
          >
            ⟳
          </button>
          <button
            onClick={() => setShowForm(!showForm)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition-colors"
          >
            + New Payment
          </button>
        </div>
      </div>

      {showForm && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
          <h3 className="text-lg font-semibold mb-4">New Cross-Border Payment</h3>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div><label className="block text-sm text-gray-400 mb-1">Sender User ID</label><input type="text" value={form.sender_user_id} onChange={e => setForm({ ...form, sender_user_id: e.target.value })} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
              <div><label className="block text-sm text-gray-400 mb-1">Receiver User ID</label><input type="text" value={form.receiver_user_id} onChange={e => setForm({ ...form, receiver_user_id: e.target.value })} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
              <div><label className="block text-sm text-gray-400 mb-1">Source</label><select value={form.source_currency} onChange={e => setForm({ ...form, source_currency: e.target.value })} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"><option>USD</option><option>EUR</option><option>GBP</option><option>CNY</option></select></div>
              <div><label className="block text-sm text-gray-400 mb-1">Target</label><select value={form.target_currency} onChange={e => setForm({ ...form, target_currency: e.target.value })} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"><option>JPY</option><option>USD</option><option>EUR</option><option>CNY</option></select></div>
              <div><label className="block text-sm text-gray-400 mb-1">Amount (major unit)</label><input type="number" step="0.01" min="0.01" value={form.source_amount} onChange={e => setForm({ ...form, source_amount: parseFloat(e.target.value) || 0 })} className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white" required /></div>
            </div>
            <div className="flex gap-3">
              <button type="submit" disabled={submitting} className="px-6 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm disabled:opacity-50">{submitting ? 'Creating...' : 'Create'}</button>
              <button type="button" onClick={() => setShowForm(false)} className="px-6 py-2 bg-gray-700 rounded-lg text-sm">Cancel</button>
            </div>
          </form>
        </div>
      )}

      {loading ? <div className="text-gray-500">Loading...</div> : (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <table className="w-full"><thead><tr className="border-b border-gray-800 text-left"><th className="p-4 text-sm text-gray-500">Payment ID</th><th className="p-4 text-sm text-gray-500">From → To</th><th className="p-4 text-sm text-gray-500">Amount</th><th className="p-4 text-sm text-gray-500">Fee</th><th className="p-4 text-sm text-gray-500">Status</th><th className="p-4 text-sm text-gray-500">Created</th></tr></thead>
          <tbody>{orders.length === 0 ? <tr><td colSpan={6} className="p-8 text-center text-gray-600">No transactions yet</td></tr> : orders.map((o: any) => (
            <tr key={o.payment_id} className="border-b border-gray-800/50 hover:bg-gray-800/30"><td className="p-4 text-sm font-mono text-gray-300">{o.payment_id?.substring(0, 20)}...</td><td className="p-4 text-sm">{o.source_currency} → {o.target_currency}</td><td className="p-4 text-sm">{(o.source_amount / 100).toFixed(2)} {o.source_currency}</td><td className="p-4 text-sm text-gray-500">{(o.fee_amount / 100).toFixed(2)}</td><td className={`p-4 text-sm font-medium ${statusColor(o.status)}`}>{o.status}</td><td className="p-4 text-sm text-gray-500">{new Date(o.created_at).toLocaleDateString()}</td></tr>
          ))}</tbody></table>
        </div>
      )}
    </div>
  )
}
