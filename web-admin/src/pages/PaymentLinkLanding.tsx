// V5 Payment Link Landing Page (§5)
// Public, mobile-first page for payers to view and pay a payment link.
import { useEffect, useState } from 'react'

export default function PaymentLinkLanding() {
  const [link, setLink] = useState<any>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [paying, setPaying] = useState(false)
  const [token, setToken] = useState(localStorage.getItem('auth_token') || '')
  const [loginForm, setLoginForm] = useState({ username: '', password: '' })

  // Extract token from URL path: /pay/:token
  const urlToken = window.location.pathname.split('/pay/')[1] || ''

  useEffect(() => {
    if (!urlToken) { setError('Invalid payment link'); setLoading(false); return }
    fetch(`/api/v2/v4/payment-links/public/${urlToken}`)
      .then(r => r.json())
      .then(d => {
        if (d.error) throw new Error(d.error)
        setLink(d)
      })
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [urlToken])

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const r = await fetch('/api/v2/auth/login', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginForm),
      })
      const d = await r.json()
      if (!r.ok) throw new Error(d.error || 'Login failed')
      localStorage.setItem('auth_token', d.token)
      setToken(d.token)
    } catch (err: any) { alert(err.message) }
  }

  const handlePay = async () => {
    if (!link || !token) return
    setPaying(true)
    try {
      // Get user's account for this currency
      const meResp = await fetch('/api/v2/users/me', { headers: { Authorization: `Bearer ${token}` } })
      const me = await meResp.json()
      const acctsResp = await fetch('/api/v2/accounts', { headers: { Authorization: `Bearer ${token}` } })
      const accts = await acctsResp.json()
      const srcAcct = (accts.accounts || []).find((a: any) => a.currency === link.currency)

      if (!srcAcct) throw new Error(`No ${link.currency} account found`)

      const payResp = await fetch(`/api/v2/v4/payment-links/${link.payment_link_id}/pay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ source_account_id: srcAcct.account_id }),
      })
      const payResult = await payResp.json()
      if (!payResp.ok) throw new Error(payResult.error || 'Payment failed')

      setLink({ ...link, status: 'paid', payResult })
    } catch (err: any) { alert(err.message) }
    finally { setPaying(false) }
  }

  // Loading skeleton
  if (loading) return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md space-y-4">
        <div className="skeleton h-48 rounded-2xl" />
        <div className="skeleton h-16 rounded-2xl" />
        <div className="skeleton h-14 rounded-xl" />
      </div>
    </div>
  )

  // Error state
  if (error || !link) return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md text-center space-y-4">
        <div className="text-6xl">🔗</div>
        <h2 className="text-xl font-semibold text-gray-800">{error || 'Link not found'}</h2>
        <p className="text-gray-500 text-sm">This payment link may have expired or been cancelled.</p>
        <p className="text-xs text-gray-400">Please contact the person who shared this link.</p>
      </div>
    </div>
  )

  // Paid state
  if (link.status === 'paid') return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md text-center space-y-6 bg-white rounded-2xl p-8 shadow-sm border border-gray-100">
        <div className="w-16 h-16 rounded-full bg-emerald-100 flex items-center justify-center mx-auto">
          <span className="text-3xl">✅</span>
        </div>
        <h2 className="text-xl font-bold text-gray-800">Payment Complete!</h2>
        <p className="text-3xl font-bold text-gray-900">{(link.amount/100).toFixed(2)} {link.currency}</p>
        <p className="text-gray-500">{link.title}</p>
        <div className="bg-gray-50 rounded-xl p-4 text-sm text-gray-600">
          <p>This payment link has been paid and cannot be used again.</p>
        </div>
      </div>
    </div>
  )

  // Expired/Cancelled state
  if (link.status === 'expired' || link.status === 'cancelled') return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md text-center space-y-4 bg-white rounded-2xl p-8 shadow-sm">
        <div className="text-5xl">⏰</div>
        <h2 className="text-xl font-semibold text-gray-800">Link {link.status}</h2>
        <p className="text-gray-500">This payment link is no longer active.</p>
      </div>
    </div>
  )

  // Pending — show payment page
  const expiresIn = link.expire_at ? Math.max(0, Math.floor((new Date(link.expire_at).getTime() - Date.now()) / 1000)) : 0
  const hours = Math.floor(expiresIn / 3600)
  const mins = Math.floor((expiresIn % 3600) / 60)

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md space-y-4">
        {/* Brand */}
        <div className="text-center py-4">
          <div className="inline-flex items-center gap-2">
            <img src="/logo.png" alt="Aspira Pay" className="w-8 h-8 object-contain opacity-90" />
            <span className="text-sm font-semibold text-gray-600">Aspira Pay</span>
          </div>
        </div>

        {/* Amount Card */}
        <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100 text-center space-y-3">
          <p className="text-sm text-gray-500">{link.title || 'Payment Request'}</p>
          <p className="text-4xl font-bold text-gray-900 tracking-tight">
            {(link.amount/100).toFixed(2)} <span className="text-2xl text-gray-500">{link.currency}</span>
          </p>
          {link.description && <p className="text-sm text-gray-400">{link.description}</p>}
          {expiresIn > 0 && (
            <div className="inline-flex items-center gap-1.5 px-3 py-1 bg-amber-50 text-amber-700 rounded-full text-xs font-medium">
              <span>⏳</span> Expires in {hours}h {mins}m
            </div>
          )}
        </div>

        {/* Action */}
        {!token ? (
          <div className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
            <p className="text-sm font-medium text-gray-700 mb-4">Sign in to pay</p>
            <form onSubmit={handleLogin} className="space-y-3">
              <input type="text" value={loginForm.username} onChange={e => setLoginForm({...loginForm, username: e.target.value})}
                placeholder="Username" required className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm" />
              <input type="password" value={loginForm.password} onChange={e => setLoginForm({...loginForm, password: e.target.value})}
                placeholder="Password" required className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm" />
              <button type="submit" className="w-full py-3 bg-[#C84B4B] text-white rounded-xl text-sm font-medium hover:bg-[#8B2E2E] transition-colors">
                Sign In & Pay
              </button>
            </form>
          </div>
        ) : (
          <button onClick={handlePay} disabled={paying}
            className="w-full py-4 bg-[#C84B4B] text-white rounded-2xl text-base font-semibold hover:bg-[#8B2E2E] transition-colors disabled:opacity-50 shadow-lg shadow-[#E07373]">
            {paying ? 'Processing...' : `Pay ${(link.amount/100).toFixed(2)} ${link.currency}`}
          </button>
        )}

        {/* Security notice (§5.2) */}
        <p className="text-center text-xs text-gray-400 flex items-center justify-center gap-1">
          <span>🔒</span> Encrypted · Secure payment · Aspira Pay
        </p>
      </div>
    </div>
  )
}
