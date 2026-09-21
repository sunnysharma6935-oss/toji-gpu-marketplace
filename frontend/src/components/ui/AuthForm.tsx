import type { FormEvent, ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Logo } from '@/components/ui/Logo'
import { Button } from '@/components/ui/Button'

interface AuthFormProps {
  title: string
  subtitle: string
  email: string
  password: string
  onEmailChange: (v: string) => void
  onPasswordChange: (v: string) => void
  onSubmit: (e: FormEvent) => void
  submitLabel: string
  loading: boolean
  error: string | null
  footer: ReactNode
}

export function AuthForm({
  title,
  subtitle,
  email,
  password,
  onEmailChange,
  onPasswordChange,
  onSubmit,
  submitLabel,
  loading,
  error,
  footer,
}: AuthFormProps) {
  return (
    <div className="flex min-h-screen items-center justify-center px-5">
      <div className="w-full max-w-sm">
        <Link to="/" className="mb-10 flex justify-center">
          <Logo />
        </Link>
        <h1 className="text-center font-sans text-2xl font-bold">{title}</h1>
        <p className="mt-2 text-center text-sm text-muted-dim">{subtitle}</p>

        <form onSubmit={onSubmit} className="card mt-8 space-y-4 p-6">
          <div>
            <label className="label-mono mb-2 block">Email</label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => onEmailChange(e.target.value)}
              className="w-full rounded border border-border bg-ink px-3 py-2.5 text-sm text-paper focus:border-border-strong focus:outline-none"
            />
          </div>
          <div>
            <label className="label-mono mb-2 block">Password</label>
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => onPasswordChange(e.target.value)}
              className="w-full rounded border border-border bg-ink px-3 py-2.5 text-sm text-paper focus:border-border-strong focus:outline-none"
            />
          </div>

          {error && <p className="text-sm text-warn">{error}</p>}

          <Button type="submit" disabled={loading} className="w-full">
            {loading ? 'Please wait…' : submitLabel}
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-muted-dim">{footer}</p>
      </div>
    </div>
  )
}
