import { Link } from 'react-router-dom'
import { Badge } from '@/components/ui/Badge'
import { formatPaisePerHour } from '@/utils/currency'
import type { Listing } from '@/types/api'

// Renders ONLY fields that actually exist on the Listing type — no fabricated
// "availability" beyond what the backend's active-listings query already filters for
// (a listing only appears here if is_active=true and its host is online, per
// ListActiveListings' SQL), no invented location/host details beyond machine_label.
export function GPUCard({ listing }: { listing: Listing }) {
  return (
    <Link
      to={`/marketplace/${listing.listing_id}`}
      className="card group block p-5 transition-colors hover:border-border-strong"
    >
      <div className="flex items-start justify-between">
        <h3 className="font-sans text-lg font-bold">{listing.model}</h3>
        <Badge tone="success">Available</Badge>
      </div>

      <div className="mt-2 flex flex-wrap gap-1.5">
        <span className="rounded bg-ink px-2 py-0.5 text-xs text-muted">
          {(listing.vram_mb / 1024).toFixed(0)} GB VRAM
        </span>
        {listing.driver_version && (
          <span className="rounded bg-ink px-2 py-0.5 text-xs text-muted">
            Driver {listing.driver_version}
          </span>
        )}
      </div>

      <div className="mt-6 flex items-end justify-between">
        <div>
          <p className="text-xs text-muted-dim">from</p>
          <p className="font-sans text-xl font-bold">{formatPaisePerHour(listing.price_paise_per_hour)}</p>
        </div>
        <span className="text-sm font-semibold text-accent group-hover:underline">
          View GPU →
        </span>
      </div>
    </Link>
  )
}
