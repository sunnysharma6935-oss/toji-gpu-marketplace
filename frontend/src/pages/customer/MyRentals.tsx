import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { DashboardLayout } from '@/layouts/DashboardLayout'
import { http } from '@/services/httpClient'

interface Rental {
  rental_id: string
  listing_id: string
  status: string
  ssh_port?: number
  jupyter_token?: string
  started_at?: string
  stopped_at?: string
  total_paise_billed: number
  created_at: string
  model: string
  vram_mb: number
  driver_version: string
  machine_label: string
  price_paise_per_hour: number
}

type Tab = 'Active' | 'Completed' | 'Stopped'

function formatINR(paise: number) {
  return `₹${(paise / 100).toFixed(2)}`
}

function formatDate(date?: string) {
  if (!date) return '—'

  return new Date(date).toLocaleString('en-IN', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function statusLabel(status: string) {
  switch (status) {
    case 'provisioning':
      return 'Provisioning'
    case 'running':
      return 'Running'
    case 'stopped':
      return 'Stopped'
    case 'failed':
      return 'Failed'
    default:
      return status
  }
}

function statusClass(status: string) {
  switch (status) {
    case 'running':
      return 'text-green-400'
    case 'provisioning':
      return 'text-yellow-400'
    case 'stopped':
      return 'text-muted-dim'
    case 'failed':
      return 'text-red-400'
    default:
      return 'text-muted-dim'
  }
}

export function MyRentals() {
  const [rentals, setRentals] = useState<Rental[]>([])
  const [activeTab, setActiveTab] = useState<Tab>('Active')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    async function loadRentals() {
      try {
        setLoading(true)
        setError('')

        const data = await http.get<Rental[]>('/rentals', true)

        setRentals(data)
      } catch (err) {
        console.error('Failed to load rentals:', err)
        setError('Could not load your rentals.')
      } finally {
        setLoading(false)
      }
    }

    loadRentals()
  }, [])

  const filteredRentals = useMemo(() => {
    if (activeTab === 'Active') {
      return rentals.filter(
        (rental) =>
          rental.status === 'provisioning' ||
          rental.status === 'running'
      )
    }

    if (activeTab === 'Stopped') {
      return rentals.filter((rental) => rental.status === 'stopped')
    }

    return rentals.filter((rental) => rental.status === 'completed')
  }, [rentals, activeTab])

  return (
    <DashboardLayout>
      <div>
        <h1 className="font-sans text-2xl font-bold">My Rentals</h1>

        <div className="mt-6 flex gap-1 border-b border-border">
          {(['Active', 'Completed', 'Stopped'] as Tab[]).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2.5 text-sm font-medium transition ${
                activeTab === tab
                  ? 'border-b-2 border-accent text-paper'
                  : 'text-muted-dim hover:text-paper'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {loading && (
          <div className="card mt-6 p-8 text-center">
            <p className="text-sm text-muted-dim">
              Loading your rentals...
            </p>
          </div>
        )}

        {!loading && error && (
          <div className="card mt-6 p-8 text-center">
            <p className="text-sm text-red-400">{error}</p>
          </div>
        )}

        {!loading && !error && filteredRentals.length === 0 && (
          <div className="card mt-6 p-8 text-center">
            <p className="text-sm text-muted-dim">
              No {activeTab.toLowerCase()} rentals.
            </p>
          </div>
        )}

        {!loading && !error && filteredRentals.length > 0 && (
          <div className="mt-6 space-y-4">
            {filteredRentals.map((rental) => (
              <div
                key={rental.rental_id}
                className="card p-5"
              >
                <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                  <div>
                    <div className="flex items-center gap-3">
                      <h2 className="text-base font-semibold text-paper">
                        {rental.model}
                      </h2>

                      <span
                        className={`text-xs font-medium ${statusClass(
                          rental.status
                        )}`}
                      >
                        {statusLabel(rental.status)}
                      </span>
                    </div>

                    <p className="mt-1 text-sm text-muted-dim">
                      {rental.machine_label}
                    </p>

                    <div className="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-muted-dim">
                      <span>{rental.vram_mb} MB VRAM</span>

                      <span>
                        {formatINR(rental.price_paise_per_hour)} / hr
                      </span>

                      <span>
                        Billed: {formatINR(rental.total_paise_billed)}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <Link
                      to={`/dashboard/rentals/${rental.rental_id}`}
                      className="rounded-md border border-border px-4 py-2 text-sm font-medium text-paper transition hover:border-accent hover:text-accent"
                    >
                      View Rental
                    </Link>
                  </div>
                </div>

                <div className="mt-4 grid gap-3 border-t border-border pt-4 text-xs text-muted-dim md:grid-cols-3">
                  <div>
                    <span className="block text-[11px] uppercase tracking-wide">
                      Created
                    </span>
                    <span className="mt-1 block text-paper">
                      {formatDate(rental.created_at)}
                    </span>
                  </div>

                  <div>
                    <span className="block text-[11px] uppercase tracking-wide">
                      Started
                    </span>
                    <span className="mt-1 block text-paper">
                      {formatDate(rental.started_at)}
                    </span>
                  </div>

                  <div>
                    <span className="block text-[11px] uppercase tracking-wide">
                      Stopped
                    </span>
                    <span className="mt-1 block text-paper">
                      {formatDate(rental.stopped_at)}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </DashboardLayout>
  )
}