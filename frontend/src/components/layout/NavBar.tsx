import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Logo } from '@/components/ui/Logo'
import { Button } from '@/components/ui/Button'
import { useAuth } from '@/hooks/useAuth'

const navLinks = [
  { label: 'Explore GPUs', to: '/marketplace' },
  { label: 'How It Works', to: '/how-it-works' },
  { label: 'Host Your GPU', to: '/host' },
]

export function NavBar() {
  const { isAuthenticated } = useAuth()
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <header className="sticky top-0 z-50 border-b border-border bg-ink/95 backdrop-blur">
      <div className="container-page flex h-16 items-center justify-between">
        <Link to="/" aria-label="TOJI home">
          <Logo />
        </Link>

        <nav className="hidden items-center gap-8 md:flex">
          {navLinks.map((link) => (
            <Link
              key={link.to}
              to={link.to}
              className="text-sm font-medium text-muted transition-colors hover:text-paper"
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          {isAuthenticated ? (
            <Link to="/dashboard">
              <Button variant="secondary" className="!py-2 !px-4 text-sm">
                Dashboard
              </Button>
            </Link>
          ) : (
            <>
              <Link to="/login" className="text-sm font-medium text-muted hover:text-paper">
                Log In
              </Link>
              <Link to="/signup">
                <Button className="!py-2 !px-4 text-sm">Get Started</Button>
              </Link>
            </>
          )}
        </div>

        <button
          className="flex h-9 w-9 items-center justify-center rounded border border-border md:hidden"
          onClick={() => setMobileOpen((o) => !o)}
          aria-label="Toggle menu"
          aria-expanded={mobileOpen}
        >
          <span className="sr-only">Menu</span>
          <div className="flex flex-col gap-1">
            <span className="h-px w-4 bg-paper" />
            <span className="h-px w-4 bg-paper" />
            <span className="h-px w-4 bg-paper" />
          </div>
        </button>
      </div>

      {mobileOpen && (
        <div className="border-t border-border px-5 py-4 md:hidden">
          <nav className="flex flex-col gap-4">
            {navLinks.map((link) => (
              <Link
                key={link.to}
                to={link.to}
                className="text-sm font-medium text-muted"
                onClick={() => setMobileOpen(false)}
              >
                {link.label}
              </Link>
            ))}
            <div className="mt-2 flex flex-col gap-3 border-t border-border pt-4">
              {isAuthenticated ? (
                <Link to="/dashboard" onClick={() => setMobileOpen(false)}>
                  <Button variant="secondary" className="w-full">
                    Dashboard
                  </Button>
                </Link>
              ) : (
                <>
                  <Link to="/login" onClick={() => setMobileOpen(false)}>
                    <Button variant="secondary" className="w-full">
                      Log In
                    </Button>
                  </Link>
                  <Link to="/signup" onClick={() => setMobileOpen(false)}>
                    <Button className="w-full">Get Started</Button>
                  </Link>
                </>
              )}
            </div>
          </nav>
        </div>
      )}
    </header>
  )
}
