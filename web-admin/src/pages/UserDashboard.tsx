import { useCallback, useEffect, useState } from 'react'
import { usePolling } from '../hooks/usePolling'

export default function UserDashboard({ userId, token }: { userId: string; token: string }) {
  const [accounts, setAccounts] = useState<any[]>([])
  const [cards, setCards] = useState<any[]>([])
  const [payments, setPayments] = useState<any[]>([])

  const loadData = useCallback(async () => {
    try {
      const h = { 'Authorization': `Bearer ${token}` }
      const [a, c, p] = await Promise.all([
        fetch('/api/v2/accounts', { headers: h }).then(r => r.json()),
        fetch('/api/v2/cards', { headers: h }).then(r => r.json()),
        fetch('/api/v2/payments', { headers: h }).then(r => r.json()),
      ])
      setAccounts(a.accounts || [])
      setCards(c.cards || [])
      setPayments(p.orders || [])
    } catch (e) { console.error(e) }
  }, [token])

  useEffect(() => { loadData() }, [loadData])
  usePolling(loadData, 5000, { immediate: false })

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Welcome</h2>

      {/* Account Balances */}
      <h3 className="text-lg font-semibold mb-3 text-gray-300">💼 Accounts</h3>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-8">
        {accounts.length === 0 && <p className="text-gray-600 col-span-4 text-sm">No accounts yet</p>}
        {accounts.map((a: any) => (
          <div key={a.account_id} className="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <p className="text-xs text-gray-500 mb-1">{a.currency}</p>
            <p className="text-xl font-bold">{(a.available_balance / 100).toFixed(2)}</p>
            <p className="text-xs text-gray-600">Available</p>
          </div>
        ))}
      </div>

      {/* Cards */}
      <h3 className="text-lg font-semibold mb-3 text-gray-300">💳 My Cards ({cards.length}/5)</h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        {cards.length === 0 && <p className="text-gray-600 col-span-2 text-sm">No cards yet — apply for one!</p>}
        {cards.map((c: any) => (
          <div key={c.card_id} className="bg-gradient-to-br from-emerald-700 via-teal-600 to-cyan-500 rounded-xl p-5 text-white">
            <div className="flex justify-between items-start mb-4">
              <span className="text-white/60 text-xs">Aspira Pay</span>
              <span className="font-bold italic text-sm">{c.card_network}</span>
            </div>
            <p className="font-mono text-lg tracking-[0.2em] mb-3">•••• •••• •••• {c.pan_last4}</p>
            <div className="flex justify-between text-xs text-white/60">
              <span>{c.default_currency}</span>
              <span className={`px-2 py-0.5 rounded-full ${c.status === 'ACTIVE' ? 'bg-white/20' : 'bg-red-500/30'}`}>{c.status}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Recent Transactions */}
      <h3 className="text-lg font-semibold mb-3 text-gray-300">📋 Recent Transactions</h3>
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-gray-800 text-left">
            <th className="p-3 text-gray-500 text-xs">ID</th><th className="p-3 text-gray-500 text-xs">Currency</th>
            <th className="p-3 text-gray-500 text-xs">Amount</th><th className="p-3 text-gray-500 text-xs">Status</th>
          </tr></thead>
          <tbody>
            {payments.length === 0 && <tr><td colSpan={4} className="p-6 text-center text-gray-600">No transactions yet</td></tr>}
            {payments.slice(0, 10).map((p: any) => (
              <tr key={p.payment_id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                <td className="p-3 font-mono text-xs text-gray-400">{p.payment_id?.substring(0, 16)}...</td>
                <td className="p-3">{p.source_currency}→{p.target_currency}</td>
                <td className="p-3">{(p.source_amount/100).toFixed(2)}</td>
                <td className={`p-3 text-xs font-medium ${p.status==='CLOSED'||p.status==='COMPLETED'?'text-green-400':p.status?.includes('FAILED')?'text-red-400':'text-blue-400'}`}>{p.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
