import { useState, useEffect } from 'react'

export default function UserTransfer({ userId, token }: { userId: string; token: string }) {
  const [tab, setTab] = useState<'send' | 'request' | 'history'>('send')
  const h = { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' }

  // Send
  const [recipient, setRecipient] = useState<any>(null)
  const [searchVal, setSearchVal] = useState('')
  const [amount, setAmount] = useState(0)
  const [currency, setCurrency] = useState('USD')
  const [remark, setRemark] = useState('')
  const [quote, setQuote] = useState<any>(null)
  const [result, setResult] = useState<any>(null)
  const [totalUSD, setTotalUSD] = useState<number | null>(null)
  const [hasAccounts, setHasAccounts] = useState(true)

  // Check balance on mount and when amount changes
  useEffect(() => {
    fetch('/api/v2/accounts/total-usd', { headers: h })
      .then(r => r.json())
      .then(d => { setTotalUSD(d.total_usd || 0); setHasAccounts(d.has_accounts) })
      .catch(() => {})
  }, [token])

  // Request (Payment Link)
  const [linkAmount, setLinkAmount] = useState(0)
  const [linkCurrency, setLinkCurrency] = useState('USD')
  const [linkTitle, setLinkTitle] = useState('')
  const [linkExpire, setLinkExpire] = useState(60)
  const [linkResult, setLinkResult] = useState<any>(null)

  const searchRecipient = async () => {
    const r = await fetch('/api/v2/v4/transfer/resolve-recipient', { method:'POST', headers:h, body:JSON.stringify({recipient_type:'aspira_id',recipient_value:searchVal,currency}) })
    const d = await r.json()
    if (r.ok) setRecipient(d)
    else alert(d.error)
  }

  const getQuote = async () => {
    if (!recipient) return
    // Check balance first
    if (totalUSD !== null && amount * 100 > totalUSD) {
      alert(`Insufficient balance. You have $${(totalUSD/100).toFixed(2)} USD equivalent across all accounts.`)
      return
    }
    const r = await fetch('/api/v2/v4/transfer/quote', { method:'POST', headers:h, body:JSON.stringify({
      source_account_id:'acc_'+userId.slice(0,8)+'_'+currency, target_account_id:recipient.recipient_account_id,
      source_currency:currency, target_currency:currency, amount:Math.round(amount*100), remark
    })})
    const d = await r.json()
    if (r.ok) setQuote(d)
    else alert(d.error)
  }

  const confirmTransfer = async () => {
    if (!quote) return
    const r = await fetch('/api/v2/v4/transfer/confirm', { method:'POST', headers:h, body:JSON.stringify({quote_id:quote.quote_id}) })
    const d = await r.json()
    setResult(d)
    setQuote(null); setRecipient(null); setAmount(0)
  }

  const createPaymentLink = async () => {
    const r = await fetch('/api/v2/v4/payment-links', { method:'POST', headers:h, body:JSON.stringify({
      receiver_account_id:'acc_'+userId.slice(0,8)+'_'+linkCurrency, amount:Math.round(linkAmount*100),
      currency:linkCurrency, title:linkTitle, expire_minutes:linkExpire
    })})
    const d = await r.json()
    if (r.ok) setLinkResult(d)
    else alert(d.error)
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Transfer</h2>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-gray-800 rounded-lg p-1 w-fit">
        {[{k:'send',l:'Send Money'},{k:'request',l:'Request Money'},{k:'history',l:'History'}].map(t => (
          <button key={t.k} onClick={() => setTab(t.k as any)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${tab===t.k?'bg-[#C84B4B] text-white':'text-gray-400 hover:text-white'}`}>{t.l}</button>
        ))}
      </div>

      {tab === 'send' && (
        <div className="space-y-6 max-w-lg">
          {!recipient ? (
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4">
              <h3 className="font-semibold">Find Recipient</h3>
              <div className="flex gap-3">
                <input value={searchVal} onChange={e => setSearchVal(e.target.value)} placeholder="Aspira ID or username"
                  className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" />
                <select value={currency} onChange={e => setCurrency(e.target.value)}
                  className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"><option>USD</option><option>EUR</option><option>GBP</option><option>JPY</option></select>
                <button onClick={searchRecipient} className="px-4 py-2 bg-[#C84B4B] hover:bg-[#B04040] rounded-lg text-sm">Search</button>
              </div>
            </div>
          ) : !quote ? (
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4">
              {!hasAccounts && (
                <div className="bg-amber-900/20 border border-amber-800/50 rounded-xl p-4 text-amber-400 text-sm">
                  ⚠ No bank accounts found. You need at least one currency account to make transfers.
                </div>
              )}
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-[#8B2E2E] flex items-center justify-center text-white font-bold">{recipient.display_name?.[0]||'?'}</div>
                <div><p className="font-medium">{recipient.display_name}</p><p className="text-xs text-gray-500">{recipient.account_no_masked} · {recipient.currency}</p></div>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Amount</label>
                <input type="number" value={amount||''} onChange={e => setAmount(parseFloat(e.target.value)||0)}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white text-lg" placeholder="0.00" />
                {totalUSD !== null && (
                  <p className={`text-xs mt-2 ${amount * 100 > totalUSD ? 'text-red-400' : 'text-gray-500'}`}>
                    Available: ${(totalUSD/100).toLocaleString(undefined, {minimumFractionDigits: 2})} USD equivalent
                    {amount > 0 && amount * 100 > totalUSD ? ' — insufficient!' : ''}
                  </p>
                )}
              </div>
              <input value={remark} onChange={e => setRemark(e.target.value)} placeholder="Add a note (optional)"
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white text-sm" />
              <button onClick={getQuote} disabled={!amount} className="w-full py-3 bg-[#C84B4B] hover:bg-[#B04040] rounded-lg font-medium disabled:opacity-50">Review Transfer</button>
            </div>
          ) : (
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4">
              <h3 className="font-semibold">Confirm Transfer</h3>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between"><span className="text-gray-400">To</span><span>{recipient.display_name}</span></div>
                <div className="flex justify-between"><span className="text-gray-400">Amount</span><span className="font-bold">{(quote.amount/100).toFixed(2)} {quote.source_currency}</span></div>
                <div className="flex justify-between"><span className="text-gray-400">Fee</span><span>${(quote.fee/100).toFixed(2)}</span></div>
                <div className="flex justify-between border-t border-gray-800 pt-2"><span className="text-gray-400">Total</span><span className="font-bold text-lg">${(quote.total_debit_amount/100).toFixed(2)}</span></div>
              </div>
              <div className="flex gap-3">
                <button onClick={() => {setQuote(null); setAmount(0)}} className="flex-1 py-2.5 bg-gray-700 rounded-lg text-sm">Cancel</button>
                <button onClick={confirmTransfer} className="flex-1 py-2.5 bg-emerald-600 hover:bg-emerald-500 rounded-lg text-sm font-medium">Confirm & Send</button>
              </div>
            </div>
          )}
          {result && (
            <div className="bg-emerald-900/30 border border-emerald-800 rounded-xl p-4 text-center">
              <p className="text-2xl mb-2">✅</p>
              <p className="font-medium text-emerald-300">Transfer Complete!</p>
              <p className="text-sm text-emerald-400/60 mt-1">{result.transfer_id}</p>
            </div>
          )}
        </div>
      )}

      {tab === 'request' && (
        <div className="max-w-lg">
          {!linkResult ? (
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4">
              <h3 className="font-semibold">Create Payment Link</h3>
              <input value={linkTitle} onChange={e => setLinkTitle(e.target.value)} placeholder="What's this for? (e.g. Lunch)"
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" />
              <div className="flex gap-3">
                <input type="number" value={linkAmount||''} onChange={e => setLinkAmount(parseFloat(e.target.value)||0)}
                  placeholder="Amount" className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white" />
                <select value={linkCurrency} onChange={e => setLinkCurrency(e.target.value)}
                  className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"><option>USD</option><option>EUR</option><option>GBP</option></select>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Expires in (minutes)</label>
                <select value={linkExpire} onChange={e => setLinkExpire(parseInt(e.target.value))}
                  className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"><option value="30">30 min</option><option value="60">1 hour</option><option value="1440">24 hours</option><option value="10080">7 days</option></select>
              </div>
              <button onClick={createPaymentLink} disabled={!linkAmount} className="w-full py-3 bg-[#C84B4B] hover:bg-[#B04040] rounded-lg font-medium disabled:opacity-50">Create Payment Link</button>
            </div>
          ) : (
            <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4 text-center">
              <p className="text-2xl">🔗</p>
              <h3 className="font-semibold">Payment Link Ready!</h3>
              <div className="bg-gray-800 rounded-lg p-3 text-left">
                <p className="text-xs text-gray-500 mb-1">Share this link:</p>
                <p className="text-sm font-mono text-[#C84B4B] break-all">https://pay.aspira.com/l/{linkResult.link_token}</p>
              </div>
              <p className="text-sm text-gray-500">Amount: ${(linkResult.amount/100).toFixed(2)} {linkResult.currency} · Status: {linkResult.status}</p>
              <button onClick={() => {setLinkResult(null); setLinkAmount(0); setLinkTitle('')}} className="px-4 py-2 bg-gray-700 rounded-lg text-sm">Create Another</button>
            </div>
          )}
        </div>
      )}

      {tab === 'history' && <TransferHistory userId={userId} token={token} />}
    </div>
  )
}

function TransferHistory({ userId, token }: { userId: string; token: string }) {
  const [transfers, setTransfers] = useState<any[]>([])
  const [contacts, setContacts] = useState<any[]>([])
  const h = { 'Authorization': `Bearer ${token}` }

  useState(() => {
    fetch('/api/v2/v4/transfer/history',{headers:h}).then(r=>r.json()).then(d=>setTransfers(d.transfers||[])).catch(()=>{})
    fetch('/api/v2/v4/transfer/contacts',{headers:h}).then(r=>r.json()).then(d=>setContacts(d.contacts||[])).catch(()=>{})
  })

  return (
    <div className="space-y-6">
      <div>
        <h3 className="font-semibold mb-3 text-gray-300">Recent Contacts</h3>
        <div className="space-y-2">
          {contacts.length===0 && <p className="text-gray-600 text-sm">No transfer history yet</p>}
          {contacts.map((c:any) => (
            <div key={c.contact_id} className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-full bg-emerald-700 flex items-center justify-center text-white text-xs font-bold">{(c.target_display_name||'?')[0]}</div>
                <div><p className="text-sm font-medium">{c.target_display_name||c.target_account_no_masked}</p><p className="text-xs text-gray-500">{c.transfer_count} transfers</p></div>
              </div>
              <span className="text-sm text-gray-400">{c.target_currency}</span>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="font-semibold mb-3 text-gray-300">Transfer History</h3>
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-left">
            <th className="p-3 text-gray-500 text-xs">ID</th><th className="p-3 text-gray-500 text-xs">Status</th><th className="p-3 text-gray-500 text-xs">Amount</th></tr></thead>
            <tbody>{transfers.length===0 && <tr><td colSpan={3} className="p-6 text-center text-gray-600">No transfers yet</td></tr>}
            {transfers.slice(0,15).map((t:any) => (
              <tr key={t.transfer_id} className="border-b border-gray-800/50">
                <td className="p-3 font-mono text-xs text-gray-400">{t.transfer_id?.substring(0,20)}</td>
                <td className={`p-3 text-xs font-medium ${t.status==='succeeded'?'text-green-400':'text-red-400'}`}>{t.status}</td>
                <td className="p-3">{(t.source_amount/100).toFixed(2)} {t.source_currency}</td>
              </tr>
            ))}</tbody></table>
        </div>
      </div>
    </div>
  )
}
