import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { MainLayout } from '@/layouts/MainLayout'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { getListingById } from '@/services/marketplaceService'
import { createRental } from '@/services/rentalService'
import { ApiError } from '@/services/httpClient'
import { formatPaisePerHour } from '@/utils/currency'
import { useAuth } from '@/hooks/useAuth'
import type { Listing } from '@/types/api'

export function GPUDetail() {
  const { listingId } = useParams<{ listingId: string }>()
  const navigate = useNavigate()
  const { isAuthenticated } = useAuth()

  const [listing, setListing] = useState<Listing | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [renting, setRenting] = useState(false)
  const [rentError, setRentError] = useState<string | null>(null)

  useEffect(() => {
    if (!listingId) return
    getListingById(listingId)
      .then(setListing)
      .catch((err) => setError(err instanceof Error ? err.message : 'Could not load this GPU.'))
      .finally(() => setLoading(false))
  }, [listingId])

  async function handleRent() {
    if (!listingId) return
    if (!isAuthenticated) {
      navigate('/login', { state: { redirectTo: `/marketplace/${listingId}` } })
      return
    }
    setRenting(true)
    setRentError(null)
    try {
      const res = await createRental(listingId)
      navigate(`/dashboard/rentals/${res.rental_id}`)
    } catch (err) {
      if (err instanceof ApiError && err.status === 402) {
        setRentError('Insufficient TOJI balance. Add balance to start this rental.')
      } else if (err instanceof ApiError && err.status === 409) {
        setRentError('This GPU is no longer available.')
      } else {
        setRentError('Could not start the rental. Please try again.')
      }
    } finally {
      setRenting(false)
    }
  }

  return (
    <MainLayout>
      <section className="container-page py-12">
        {loading && <p className="text-sm text-muted-dim">Loading…</p>}
        {error && <p className="text-sm text-muted">{error}</p>}

        {listing && (
          <div className="grid gap-8 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <div className="flex items-center gap-3">
                <h1 className="font-sans text-3xl font-bold">{listing.model}</h1>
                <Badge tone="success">Available</Badge>
              </div>
              <p className="mt-1 text-sm text-muted-dim">{listing.machine_label}</p>

              <div className="card mt-8 grid grid-cols-2 gap-6 p-6 sm:grid-cols-3">
                <div>
                  <p className="label-mono mb-1">VRAM</p>
                  <p className="font-sans text-lg font-bold">{(listing.vram_mb / 1024).toFixed(0)} GB</p>
                </div>
                <div>
                  <p className="label-mono mb-1">Driver</p>
                  <p className="font-sans text-lg font-bold">{listing.driver_version || '—'}</p>
                </div>
                <div>
                  <p className="label-mono mb-1">Price</p>
                  <p className="font-sans text-lg font-bold">{formatPaisePerHour(listing.price_paise_per_hour)}</p>
                </div>
              </div>

              <div className="mt-8">
                <p className="label-mono mb-3">Supported workloads</p>
                <div className="flex flex-wrap gap-2">
                  {['AI Inference', 'Image Generation', 'ComfyUI', 'Jupyter', 'Fine-tuning'].map((w) => (
                    <span key={w} className="rounded border border-border px-3 py-1.5 text-xs text-muted">
                      {w}
                    </span>
                  ))}
                </div>
              </div>
            </div>

            <div className="card h-fit p-6">
              <p className="text-xs text-muted-dim">from</p>
              <p className="font-sans text-3xl font-bold">{formatPaisePerHour(listing.price_paise_per_hour)}</p>
              <Button onClick={handleRent} disabled={renting} className="mt-6 w-full">
                {renting ? 'Starting…' : 'Rent GPU'}
              </Button>
              {rentError && (
                <p className="mt-3 text-sm text-warn">
                  {rentError}
                  {rentError.includes('balance') && (
                    <>
                      {' '}
                      <button
                        onClick={() => navigate('/dashboard/billing')}
                        className="underline"
                      >
                        Add balance →
                      </button>
                    </>
                  )}
                </p>
              )}
              <p className="mt-4 text-xs text-muted-dim">
                Billed by the second. Stop anytime from your dashboard.
              </p>
            </div>
          </div>
        )}

        {!loading && !error && !listing && (
          <p className="text-sm text-muted-dim">This GPU isn't available anymore.</p>
        )}
      </section>
    </MainLayout>
  )
}
