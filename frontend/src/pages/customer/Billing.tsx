import { useEffect, useState } from 'react'
import { DashboardLayout } from '@/layouts/DashboardLayout'
import { Button } from '@/components/ui/Button'
import { getBalance, topUp, getTransactions } from '@/services/billingService'
import { formatPaise } from '@/utils/currency'
import type { BalanceResponse, Transaction, TopUpMethod } from '@/types/api'

const quickAmounts = [1000, 2500, 5000, 10000] // paise: ₹10, ₹25, ₹50, ₹100

export function Billing() {
  const [balance, setBalance] = useState<BalanceResponse | null>(null)
  const [transactions, setTransactions] = useState<Transaction[] | null>(null)
  const [amount, setAmount] = useState(2500)
  const [method, setMethod] = useState<TopUpMethod>('card')
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  function refresh() {
    getBalance().then(setBalance).catch(() => {})
    getTransactions().then(setTransactions).catch(() => {})
  }

  useEffect(refresh, [])

  async function handleTopUp() {
    setSubmitting(true)
    setMessage(null)
    try {
      await topUp(amount, method)
      setMessage('Balance added successfully.')
      refresh()
    } catch {
      setMessage('Top-up failed. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <DashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Billing</h1>

      <div className="mt-8 grid gap-6 lg:grid-cols-2">
        <div className="card p-6">
          <p className="label-mono mb-2">TOJI Balance</p>
          <p className="font-sans text-3xl font-bold">
            {balance ? formatPaise(balance.balance_paise) : '—'}
          </p>
          <p className="mt-1 text-xs text-muted-dim">
            This is prepaid TOJI credit, used to pay for rentals as you use them.
          </p>

          <div className="mt-6 border-t border-border pt-6">
            <p className="label-mono mb-3">Add Balance</p>
            <div className="flex flex-wrap gap-2">
              {quickAmounts.map((a) => (
                <button
                  key={a}
                  onClick={() => setAmount(a)}
                  className={`rounded border px-3 py-2 text-sm ${
                    amount === a ? 'border-accent text-accent' : 'border-border text-muted'
                  }`}
                >
                  {formatPaise(a)}
                </button>
              ))}
            </div>

            <div className="mt-4 flex gap-2">
              {(['card', 'upi', 'netbanking'] as TopUpMethod[]).map((m) => (
                <button
                  key={m}
                  onClick={() => setMethod(m)}
                  className={`rounded border px-3 py-2 text-xs capitalize ${
                    method === m ? 'border-accent text-accent' : 'border-border text-muted-dim'
                  }`}
                >
                  {m}
                </button>
              ))}
            </div>

            <Button onClick={handleTopUp} disabled={submitting} className="mt-5 w-full">
              {submitting ? 'Processing…' : `Add ${formatPaise(amount)}`}
            </Button>
            {message && <p className="mt-3 text-sm text-muted">{message}</p>}
          </div>
        </div>

        <div className="card p-6">
          <p className="label-mono mb-4">Transaction History</p>
          {transactions === null && <p className="text-sm text-muted-dim">Loading…</p>}
          {transactions?.length === 0 && (
            <p className="text-sm text-muted-dim">No transactions yet.</p>
          )}
          <ul className="space-y-3">
            {transactions?.map((tx) => (
              <li key={tx.id} className="flex items-center justify-between border-b border-border pb-3 text-sm">
                <div>
                  <p className="capitalize">{tx.type.replace('_', ' ')}</p>
                  <p className="text-xs text-muted-dim">{new Date(tx.created_at).toLocaleString('en-IN')}</p>
                </div>
                <span className={tx.amount_paise < 0 ? 'text-paper' : 'text-success'}>
                  {tx.amount_paise < 0 ? '' : '+'}
                  {formatPaise(tx.amount_paise)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </DashboardLayout>
  )
}
