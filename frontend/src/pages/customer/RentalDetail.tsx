import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { http } from '@/services/httpClient'

interface RentalDetailData {
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

interface StopRentalResponse {
  status: string
  charged_paise?: number
  shortfall_paise?: number
  already_final?: boolean
}

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

function getStatusLabel(status: string) {
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

function getStatusClass(status: string) {
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

function calculateRuntime(startedAt?: string, stoppedAt?: string) {
  if (!startedAt) return '—'

  const start = new Date(startedAt).getTime()
  const end = stoppedAt
    ? new Date(stoppedAt).getTime()
    : Date.now()

  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) {
    return '—'
  }

  const totalSeconds = Math.floor((end - start) / 1000)

  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`
  }

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`
  }

  return `${seconds}s`
}

export function RentalDetail() {
  const { rentalId } = useParams<{ rentalId: string }>()

  const [rental, setRental] = useState<RentalDetailData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [stopping, setStopping] = useState(false)
  const [stopResult, setStopResult] = useState<StopRentalResponse | null>(null)
  const [now, setNow] = useState(Date.now())

  async function loadRental() {
    if (!rentalId) {
      setError('Rental ID is missing.')
      setLoading(false)
      return
    }

    try {
      setError('')

      const data = await http.get<RentalDetailData>(
        `/rentals/${rentalId}`,
        true
      )

      setRental(data)
    } catch (err) {
      console.error('Failed to load rental:', err)
      setError('Could not load this rental.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadRental()
  }, [rentalId])

  useEffect(() => {
    if (!rental || rental.status !== 'running') {
      return
    }

    const timer = window.setInterval(() => {
      setNow(Date.now())
    }, 1000)

    return () => window.clearInterval(timer)
  }, [rental?.status])

  async function handleStopRental() {
    if (!rentalId || stopping) return

    const confirmed = window.confirm(
      'Are you sure you want to stop this rental?'
    )

    if (!confirmed) return

    try {
      setStopping(true)
      setStopResult(null)

      const result = await http.post<StopRentalResponse>(
        `/rentals/${rentalId}/stop`,
        undefined,
        true
      )

      setStopResult(result)

      await loadRental()
    } catch (err) {
      console.error('Failed to stop rental:', err)
      setError('Could not stop this rental.')
    } finally {
      setStopping(false)
    }
  }

  const workspaceUrl = useMemo(() => {
    if (!rental?.ssh_port || !rental?.jupyter_token) {
      return null
    }

    return `http://localhost:${rental.ssh_port}/lab?token=${encodeURIComponent(
      rental.jupyter_token
    )}`
  }, [rental])

  const runtime = useMemo(() => {
    if (!rental) return '—'

    if (rental.status === 'running') {
      const start = rental.started_at
        ? new Date(rental.started_at).getTime()
        : 0

      if (!start) return '—'

      const totalSeconds = Math.floor((now - start) / 1000)

      if (totalSeconds < 0) return '—'

      const hours = Math.floor(totalSeconds / 3600)
      const minutes = Math.floor((totalSeconds % 3600) / 60)
      const seconds = totalSeconds % 60

      if (hours > 0) {
        return `${hours}h ${minutes}m ${seconds}s`
      }

      if (minutes > 0) {
        return `${minutes}m ${seconds}s`
      }

      return `${seconds}s`
    }

    return calculateRuntime(rental.started_at, rental.stopped_at)
  }, [rental, now])

  if (loading) {
    return (
      <div className="min-h-screen bg-black px-10 py-8 text-paper">
        <div className="text-sm text-muted-dim">
          Loading rental...
        </div>
      </div>
    )
  }

  if (error && !rental) {
    return (
      <div className="min-h-screen bg-black px-10 py-8 text-paper">
        <Link
          to="/dashboard/rentals"
          className="text-sm text-muted-dim hover:text-paper"
        >
          ← My Rentals
        </Link>

        <div className="card mt-8 max-w-xl p-8">
          <p className="text-sm text-red-400">{error}</p>
        </div>
      </div>
    )
  }

  if (!rental) {
    return null
  }

  const isActive =
    rental.status === 'running' ||
    rental.status === 'provisioning'

  return (
    <div className="min-h-screen bg-black px-10 py-8 text-paper">
      <div className="max-w-6xl">
        <Link
          to="/dashboard/rentals"
          className="text-sm text-muted-dim hover:text-paper"
        >
          ← My Rentals
        </Link>

        <div className="mt-6">
          <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
            <div>
              <h1 className="font-sans text-3xl font-bold">
                {rental.model}
              </h1>

              <p className="mt-1 font-mono text-xs text-muted-dim">
                {rental.rental_id}
              </p>
            </div>

            <div
              className={`text-sm font-medium ${getStatusClass(
                rental.status
              )}`}
            >
              {getStatusLabel(rental.status)}
            </div>
          </div>
        </div>

        {error && (
          <div className="mt-6 rounded-lg border border-red-900 bg-red-950/20 p-4">
            <p className="text-sm text-red-400">{error}</p>
          </div>
        )}

        {stopResult && (
          <div className="mt-6 rounded-lg border border-border bg-panel p-4">
            <p className="text-sm text-paper">
              Rental stopped successfully.
            </p>

            {stopResult.charged_paise !== undefined && (
              <p className="mt-1 text-xs text-muted-dim">
                Final charge: {formatINR(stopResult.charged_paise)}
              </p>
            )}
          </div>
        )}

        <div className="mt-8 grid gap-6 lg:grid-cols-3">
          {/* Status */}
          <div className="card p-6">
            <p className="text-xs uppercase tracking-wide text-muted-dim">
              Status
            </p>

            <p
              className={`mt-3 text-xl font-semibold ${getStatusClass(
                rental.status
              )}`}
            >
              {getStatusLabel(rental.status)}
            </p>

            <p className="mt-2 text-sm text-muted-dim">
              {rental.status === 'running'
                ? 'Your GPU is currently running.'
                : rental.status === 'provisioning'
                  ? 'Your GPU is being prepared.'
                  : rental.status === 'stopped'
                    ? 'This rental has been stopped.'
                    : 'This rental is no longer active.'}
            </p>
          </div>

          {/* GPU */}
          <div className="card p-6">
            <p className="text-xs uppercase tracking-wide text-muted-dim">
              GPU
            </p>

            <p className="mt-3 text-xl font-semibold">
              {rental.model}
            </p>

            <div className="mt-4 space-y-2 text-sm">
              <div className="flex justify-between gap-4">
                <span className="text-muted-dim">VRAM</span>
                <span>{rental.vram_mb} MB</span>
              </div>

              <div className="flex justify-between gap-4">
                <span className="text-muted-dim">Driver</span>
                <span>{rental.driver_version || '—'}</span>
              </div>

              <div className="flex justify-between gap-4">
                <span className="text-muted-dim">Machine</span>
                <span>{rental.machine_label}</span>
              </div>
            </div>
          </div>

          {/* Billing */}
          <div className="card p-6">
            <p className="text-xs uppercase tracking-wide text-muted-dim">
              Billing
            </p>

            <p className="mt-3 text-xl font-semibold">
              {formatINR(rental.price_paise_per_hour)} / hr
            </p>

            <div className="mt-4 space-y-2 text-sm">
              <div className="flex justify-between gap-4">
                <span className="text-muted-dim">Billed</span>
                <span>{formatINR(rental.total_paise_billed)}</span>
              </div>

              <div className="flex justify-between gap-4">
                <span className="text-muted-dim">Runtime</span>
                <span>{runtime}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Workspace */}
        <div className="card mt-6 p-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="text-xs uppercase tracking-wide text-muted-dim">
                Workspace
              </p>

              <h2 className="mt-2 text-lg font-semibold">
                JupyterLab
              </h2>

              <p className="mt-1 text-sm text-muted-dim">
                {workspaceUrl
                  ? 'Your GPU workspace is ready.'
                  : rental.status === 'provisioning'
                    ? 'Workspace will appear when provisioning is complete.'
                    : 'Workspace is not available for this rental.'}
              </p>
            </div>

            {workspaceUrl && rental.status === 'running' && (
              <button
                onClick={() => window.open(workspaceUrl, '_blank')}
                className="rounded-md bg-accent px-5 py-2.5 text-sm font-semibold text-black transition hover:opacity-90"
              >
                Open JupyterLab
              </button>
            )}
          </div>
        </div>

        {/* Timeline */}
        <div className="card mt-6 p-6">
          <p className="text-xs uppercase tracking-wide text-muted-dim">
            Rental Timeline
          </p>

          <div className="mt-5 grid gap-5 md:grid-cols-3">
            <div>
              <p className="text-xs text-muted-dim">
                Created
              </p>

              <p className="mt-1 text-sm">
                {formatDate(rental.created_at)}
              </p>
            </div>

            <div>
              <p className="text-xs text-muted-dim">
                Started
              </p>

              <p className="mt-1 text-sm">
                {formatDate(rental.started_at)}
              </p>
            </div>

            <div>
              <p className="text-xs text-muted-dim">
                Stopped
              </p>

              <p className="mt-1 text-sm">
                {formatDate(rental.stopped_at)}
              </p>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="mt-6 flex flex-col gap-3 sm:flex-row">
          <button
            onClick={loadRental}
            className="rounded-md border border-border px-5 py-2.5 text-sm font-medium text-paper transition hover:border-accent hover:text-accent"
          >
            Refresh Status
          </button>

          {isActive && (
            <button
              onClick={handleStopRental}
              disabled={stopping}
              className="rounded-md border border-red-900 px-5 py-2.5 text-sm font-medium text-red-400 transition hover:bg-red-950/30 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {stopping ? 'Stopping...' : 'Stop Rental'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}