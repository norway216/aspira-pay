import { useState, useEffect } from 'react'
import { api, ensureAuth } from '../api/client'

export default function Ledger() {
  const [paymentId, setPaymentId] = useState('')
  const [summary, setSummary] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [authOk, setAuthOk] = useState(false)

  useEffect(() => { ensureAuth().then(() => setAuthOk(true)).catch(() => {}) }, [])

  const handleSearch = async () => {
    if (!paymentId.trim()) return
    setLoading(true)
    try { const data = await api.getLedger(paymentId); setSummary(data) } catch (err: any) { alert(err.message) }
    finally { setLoading(false) }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Ledger Explorer</h2>
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 mb-6">
        <div className="flex gap-4"><input type="text" value={paymentId} onChange={e => setPaymentId(e.target.value)} placeholder="Payment ID" className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white font-mono" /><button onClick={handleSearch} disabled={loading || !authOk} className="px-6 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium disabled:opacity-50">{loading ? 'Loading...' : 'Search'}</button></div>
      </div>
      {summary && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-4"><p className="text-sm text-gray-500">Total Debit</p><p className="text-xl font-bold text-red-400">{(summary.total_debit / 100).toFixed(2)}</p></div>
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-4"><p className="text-sm text-gray-500">Total Credit</p><p className="text-xl font-bold text-green-400">{(summary.total_credit / 100).toFixed(2)}</p></div>
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-4"><p className="text-sm text-gray-500">Entries</p><p className="text-xl font-bold">{summary.entry_count}</p></div>
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-4"><p className="text-sm text-gray-500">Balanced</p><p className={`text-xl font-bold ${summary.is_balanced ? 'text-green-400' : 'text-red-400'}`}>{summary.is_balanced ? '✓' : '✗'}</p></div>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden"><table className="w-full"><thead><tr className="border-b border-gray-800 text-left"><th className="p-4 text-sm text-gray-500">Entry</th><th className="p-4 text-sm text-gray-500">Account</th><th className="p-4 text-sm text-gray-500">Currency</th><th className="p-4 text-sm text-gray-500">Direction</th><th className="p-4 text-sm text-gray-500">Amount</th><th className="p-4 text-sm text-gray-500">Balance After</th></tr></thead>
          <tbody>{summary.entries?.map((e: any) => (<tr key={e.entry_id} className="border-b border-gray-800/50"><td className="p-4 text-xs font-mono text-gray-400">{e.entry_id?.substring(0, 30)}</td><td className="p-4 text-xs font-mono">{e.account_id?.substring(0, 16)}</td><td className="p-4 text-sm">{e.currency}</td><td className={`p-4 text-sm font-medium ${e.direction === 'DEBIT' ? 'text-red-400' : 'text-green-400'}`}>{e.direction}</td><td className="p-4 text-sm">{(e.amount / 100).toFixed(2)}</td><td className="p-4 text-sm">{(e.balance_after / 100).toFixed(2)}</td></tr>))}</tbody></table></div>
        </div>
      )}
    </div>
  )
}
