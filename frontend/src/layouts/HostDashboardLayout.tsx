import type { ReactNode } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Logo } from '@/components/ui/Logo'
import { useAuth } from '@/hooks/useAuth'

const links = [
  { label: 'Dashboard', to: '/host/dashboard' },
  { label: 'Machines', to: '/host/machines' },
  { label: 'Listings', to: '/host/listings' },
  { label: 'Earnings', to: '/host/earnings' },
  { label: 'Account', to: '/host/account' },
]

export function HostDashboardLayout({ children }: { children: ReactNode }) {
  const location = useLocation()
  const { logout } = useAuth()
  const navigate = useNavigate()

  return (
    <div className="flex min-h-screen">
      <aside className="hidden w-56 shrink-0 border-r border-border p-5 md:block">
        <Link to="/"><Logo size={24} /></Link>
        <p className="label-mono mt-2">Host</p>
        <nav className="mt-8 space-y-1">
          {links.map((link) => {
            const active = location.pathname === link.to
            return (
              <Link
                key={link.to}
                to={link.to}
                className={`block rounded px-3 py-2 text-sm font-medium ${
                  active ? 'bg-surface text-paper' : 'text-muted hover:text-paper'
                }`}
              >
                {link.label}
              </Link>
            )
          })}
        </nav>
        <button
          onClick={() => { logout(); navigate('/') }}
          className="mt-10 block px-3 text-sm text-muted-dim hover:text-paper"
        >
          Log out
        </button>
      </aside>
      <main className="flex-1 p-6 sm:p-10">{children}</main>
    </div>
  )
}
