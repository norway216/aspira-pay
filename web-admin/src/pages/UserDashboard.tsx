import { useCallback, useEffect, useState } from 'react'
import { usePolling } from '../hooks/usePolling'

export default function UserDashboard({ userId, token }: { userId: string; token: string }) {
  const [accounts, setAccounts] = useState<any[]>([])
  const [cards, setCards] = useState<any[]>([])
  const [payments, setPayments] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

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
    finally { setLoading(false) }
  }, [token])

  useEffect(() => { loadData() }, [loadData])
  usePolling(loadData, 8000, { immediate: false })

  const totalBalance = accounts.reduce((sum: number, a: any) => sum + (a.available_balance || 0), 0)

  if (loading) return (
    <div className="space-y-6 animate-fade-in">
      <div className="skeleton h-8 w-48 rounded-lg" />
      <div className="grid grid-cols-4 gap-4"><div className="skeleton h-28 rounded-2xl" /><div className="skeleton h-28 rounded-2xl" /><div className="skeleton h-28 rounded-2xl" /><div className="skeleton h-28 rounded-2xl" /></div>
    </div>
  )

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-white">Welcome back</h2>
        <p className="text-gray-500 text-sm mt-1">Here's your financial overview</p>
      </div>

      {/* Balance Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {accounts.slice(0, 4).map((a: any) => (
          <div key={a.currency || a.account_id} className="bg-gray-900/80 border border-gray-800/50 rounded-2xl p-5 hover:border-gray-700/50 transition-all duration-200">
            <div className="flex items-center justify-between mb-3">
              <span className="text-xs text-gray-500 uppercase tracking-wider font-medium">{a.currency}</span>
              <span className="w-7 h-7 rounded-lg bg-gray-800 flex items-center justify-center text-xs font-bold">{a.currency?.slice(0,1) || '$'}</span>
            </div>
            <p className="text-2xl font-bold text-white tabular-nums">
              {(a.available_balance / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
            </p>
            <p className="text-xs text-gray-600 mt-1">Available balance</p>
          </div>
        ))}
        {accounts.length === 0 && (
          <div className="col-span-4 bg-gray-900/50 border border-dashed border-gray-800 rounded-2xl p-8 text-center">
            <p className="text-4xl mb-3">💰</p>
            <p className="text-gray-400 font-medium">No accounts yet</p>
            <p className="text-gray-600 text-sm mt-1">Accounts are created automatically when you register</p>
          </div>
        )}
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {[
          { label: 'Total Balance', value: `$${(totalBalance/100).toLocaleString(undefined,{minimumFractionDigits:2})}`, color: 'from-[#C84B4B]/10 to-[#E07373]/5 border-[#C84B4B]/20' },
          { label: 'Active Cards', value: cards.filter((c:any) => c.status === 'ACTIVE').length.toString(), color: 'from-emerald-500/10 to-teal-500/5 border-emerald-500/20' },
          { label: 'Payments', value: payments.length.toString(), color: 'from-violet-500/10 to-purple-500/5 border-violet-500/20' },
          { label: 'Currencies', value: accounts.length.toString(), color: 'from-amber-500/10 to-orange-500/5 border-amber-500/20' },
        ].map(s => (
          <div key={s.label} className={`bg-gradient-to-br ${s.color} border rounded-2xl p-4`}>
            <p className="text-xs text-gray-500 mb-1">{s.label}</p>
            <p className="text-xl font-bold text-white">{s.value}</p>
          </div>
        ))}
      </div>

      {/* Recent Activity */}
      <div>
        <h3 className="text-lg font-semibold text-white mb-4">Recent Activity</h3>
        <div className="bg-gray-900/80 border border-gray-800/50 rounded-2xl overflow-hidden">
          <table className="w-full text-sm">
            <thead><tr className="border-b border-gray-800/50 text-left">
              <th className="px-5 py-3 text-xs text-gray-500 font-medium uppercase tracking-wider">Transaction</th>
              <th className="px-5 py-3 text-xs text-gray-500 font-medium uppercase tracking-wider">Amount</th>
              <th className="px-5 py-3 text-xs text-gray-500 font-medium uppercase tracking-wider">Status</th>
              <th className="px-5 py-3 text-xs text-gray-500 font-medium uppercase tracking-wider hidden sm:table-cell">Date</th>
            </tr></thead>
            <tbody>
              {payments.length === 0 && <tr><td colSpan={4} className="px-5 py-12 text-center text-gray-600">
                <span className="text-3xl block mb-2">📋</span>No transactions yet</td></tr>}
              {payments.slice(0, 8).map((p: any) => (
                <tr key={p.payment_id} className="border-b border-gray-800/30 hover:bg-gray-800/30 transition-colors">
                  <td className="px-5 py-3">
                    <p className="text-white text-sm font-medium">{p.source_currency} → {p.target_currency}</p>
                    <p className="text-xs text-gray-600 font-mono">{p.payment_id?.substring(0, 18)}...</p>
                  </td>
                  <td className="px-5 py-3 font-medium">{(p.source_amount/100).toFixed(2)}</td>
                  <td className="px-5 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${
                      p.status === 'CLOSED' || p.status === 'COMPLETED' ? 'bg-[#3D1A1A]/30 text-[#E07373]' :
                      p.status?.includes('FAILED') || p.status?.includes('REJECTED') ? 'bg-red-900/30 text-red-400' :
                      'bg-[#3D1A1A]/30 text-[#C84B4B]'}`}>{p.status}</span>
                  </td>
                  <td className="px-5 py-3 text-gray-500 text-xs hidden sm:table-cell">{new Date(p.created_at).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
