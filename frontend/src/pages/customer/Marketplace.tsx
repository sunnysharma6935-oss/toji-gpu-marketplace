import { useEffect, useMemo, useState } from 'react'
import { MainLayout } from '@/layouts/MainLayout'
import { GPUCard } from '@/components/marketplace/GPUCard'
import { MarketplaceControls, type SortOption } from '@/components/marketplace/MarketplaceControls'
import { getActiveListings } from '@/services/marketplaceService'
import type { Listing } from '@/types/api'

export function Marketplace() {
  const [listings, setListings] = useState<Listing[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<SortOption>('price-asc')
  const [modelFilter, setModelFilter] = useState('')

  useEffect(() => {
    let cancelled = false
    getActiveListings()
      .then((data) => {
        if (!cancelled) setListings(data)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Could not load the marketplace.')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const availableModels = useMemo(() => {
    if (!listings) return []
    return Array.from(new Set(listings.map((l) => l.model))).sort()
  }, [listings])

  const visible = useMemo(() => {
    if (!listings) return []
    let result = listings.filter((l) =>
      l.model.toLowerCase().includes(search.toLowerCase())
    )
    if (modelFilter) {
      result = result.filter((l) => l.model === modelFilter)
    }
    result = [...result].sort((a, b) => {
      if (sort === 'price-asc') return a.price_paise_per_hour - b.price_paise_per_hour
      if (sort === 'price-desc') return b.price_paise_per_hour - a.price_paise_per_hour
      return b.vram_mb - a.vram_mb // vram-desc
    })
    return result
  }, [listings, search, modelFilter, sort])

  return (
    <MainLayout>
      <section className="container-page py-12">
        <h1 className="font-sans text-3xl font-bold">GPU Marketplace</h1>
        <p className="mt-2 text-sm text-muted">
          Browse available GPUs. Priced by the hour, billed by the second.
        </p>

        <div className="mt-8">
          <MarketplaceControls
            search={search}
            onSearchChange={setSearch}
            sort={sort}
            onSortChange={setSort}
            modelFilter={modelFilter}
            onModelFilterChange={setModelFilter}
            availableModels={availableModels}
          />
        </div>

        <div className="mt-8">
          {error && (
            <div className="card p-8 text-center">
              <p className="text-sm text-muted">
                Couldn't reach the marketplace right now.{' '}
                <span className="text-muted-dim">({error})</span>
              </p>
            </div>
          )}

          {!error && listings === null && (
            <div className="card p-8 text-center text-sm text-muted-dim">Loading GPUs…</div>
          )}

          {!error && listings !== null && visible.length === 0 && (
            <div className="card p-8 text-center text-sm text-muted-dim">
              No GPUs match your filters right now.
            </div>
          )}

          {visible.length > 0 && (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {visible.map((listing) => (
                <GPUCard key={listing.listing_id} listing={listing} />
              ))}
            </div>
          )}
        </div>
      </section>
    </MainLayout>
  )
}
