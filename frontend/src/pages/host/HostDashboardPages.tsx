import { Link } from 'react-router-dom'
import { HostDashboardLayout } from '@/layouts/HostDashboardLayout'
import { Button } from '@/components/ui/Button'

// None of GET /hosts/mine, GET /hosts/{id}/gpus, or GET /host/earnings exist on the
// backend yet (confirmed by reading the repository directly — see services/hostService.ts
// for the exact list). Every page below is structured per the brief and ready to bind
// to real data the moment those endpoints exist, but shows an explicit waiting state
// rather than inventing machines, listings, or earnings figures.

function PendingBackendCard({ what }: { what: string }) {
  return (
    <div className="card p-8 text-center">
      <p className="text-sm text-muted-dim">{what} will appear here once connected.</p>
      <p className="mt-1 text-xs text-muted-dim">(Waiting on a backend endpoint — see hostService.ts)</p>
    </div>
  )
}

export function HostDashboardOverview() {
  return (
    <HostDashboardLayout>
      <div className="flex items-center justify-between">
        <h1 className="font-sans text-2xl font-bold">Host Dashboard</h1>
        <Link to="/host/onboarding">
          <Button className="!py-2 text-sm">+ Connect a Machine</Button>
        </Link>
      </div>
      <div className="mt-8 grid gap-4 sm:grid-cols-4">
        {['Connected Machines', 'GPUs', 'Active Rentals', 'Earnings'].map((label) => (
          <div key={label} className="card p-5">
            <p className="label-mono mb-2">{label}</p>
            <p className="font-sans text-2xl font-bold text-muted-dim">—</p>
          </div>
        ))}
      </div>
    </HostDashboardLayout>
  )
}

export function HostMachines() {
  return (
    <HostDashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Machines</h1>
      <div className="mt-6"><PendingBackendCard what="Your connected machines" /></div>
    </HostDashboardLayout>
  )
}

export function HostListings() {
  return (
    <HostDashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Listings</h1>
      <div className="mt-6"><PendingBackendCard what="Your GPU listings" /></div>
    </HostDashboardLayout>
  )
}

export function HostEarnings() {
  return (
    <HostDashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Earnings</h1>
      <div className="mt-6"><PendingBackendCard what="Your earnings and payout history" /></div>
    </HostDashboardLayout>
  )
}

export function HostAccount() {
  return (
    <HostDashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Account</h1>
    </HostDashboardLayout>
  )
}
