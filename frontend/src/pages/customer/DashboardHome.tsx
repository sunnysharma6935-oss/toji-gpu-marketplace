import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { DashboardLayout } from '@/layouts/DashboardLayout'
import { Button } from '@/components/ui/Button'
import { getBalance } from '@/services/billingService'
import { formatPaise } from '@/utils/currency'
import type { BalanceResponse } from '@/types/api'

export function DashboardHome() {
  const [balance, setBalance] = useState<BalanceResponse | null>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    getBalance().then(setBalance).catch(() => setError(true))
  }, [])

  return (
    <DashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Dashboard</h1>

      <div className="mt-8 grid gap-4 sm:grid-cols-3">
        <div className="card p-5">
          <p className="label-mono mb-2">TOJI Balance</p>
          {error && <p className="text-sm text-muted-dim">Couldn't load balance.</p>}
          {!error && !balance && <p className="text-sm text-muted-dim">Loading…</p>}
          {balance && <p className="font-sans text-2xl font-bold">{formatPaise(balance.balance_paise)}</p>}
          <Link to="/dashboard/billing" className="mt-3 inline-block text-xs text-accent hover:underline">
            Add balance →
          </Link>
        </div>

        {/* No GET /rentals endpoint exists yet on the backend — this card is honest
            about that rather than fabricating an "active rental" figure. */}
        <div className="card p-5">
          <p className="label-mono mb-2">Active Rental</p>
          <p className="text-sm text-muted-dim">
            Rental status will appear here once you start one.
          </p>
        </div>

        <div className="card p-5">
          <p className="label-mono mb-2">Quick Access</p>
          <Link to="/marketplace">
            <Button variant="secondary" className="mt-1 w-full !py-2 text-sm">
              Browse Marketplace
            </Button>
          </Link>
        </div>
      </div>
    </DashboardLayout>
  )
}
