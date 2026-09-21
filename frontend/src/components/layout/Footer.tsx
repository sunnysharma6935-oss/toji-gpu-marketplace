import { Link } from 'react-router-dom'
import { Logo } from '@/components/ui/Logo'

export function Footer() {
  return (
    <footer className="border-t border-border">
      <div className="container-page grid grid-cols-2 gap-10 py-14 sm:grid-cols-4">
        <div className="col-span-2 sm:col-span-1">
          <Logo size={22} />
          <p className="mt-3 text-sm text-muted-dim">
            Rent GPU power. Run AI. Pay as you go.
          </p>
        </div>
        <div>
          <p className="label-mono mb-4">Marketplace</p>
          <ul className="space-y-2.5 text-sm text-muted">
            <li><Link to="/marketplace" className="hover:text-paper">Explore GPUs</Link></li>
            <li><Link to="/how-it-works" className="hover:text-paper">How It Works</Link></li>
          </ul>
        </div>
        <div>
          <p className="label-mono mb-4">Hosts</p>
          <ul className="space-y-2.5 text-sm text-muted">
            <li><Link to="/host" className="hover:text-paper">Host Your GPU</Link></li>
          </ul>
        </div>
        <div>
          <p className="label-mono mb-4">Account</p>
          <ul className="space-y-2.5 text-sm text-muted">
            <li><Link to="/login" className="hover:text-paper">Log In</Link></li>
            <li><Link to="/signup" className="hover:text-paper">Get Started</Link></li>
          </ul>
        </div>
      </div>
    </footer>
  )
}
