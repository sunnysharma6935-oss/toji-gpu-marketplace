import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { AuthForm } from '@/components/ui/AuthForm'
import { login as loginRequest } from '@/services/authService'
import { useAuth } from '@/hooks/useAuth'

export function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const res = await loginRequest(email, password)
      login(res)
      const redirectTo = (location.state as { redirectTo?: string })?.redirectTo || '/dashboard'
      navigate(redirectTo)
    } catch {
      setError('Invalid email or password.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthForm
      title="Log In"
      subtitle="Welcome back to TOJI."
      email={email}
      password={password}
      onEmailChange={setEmail}
      onPasswordChange={setPassword}
      onSubmit={handleSubmit}
      submitLabel="Log In"
      loading={loading}
      error={error}
      footer={
        <>
          Don't have an account?{' '}
          <Link to="/signup" className="text-accent hover:underline">Get Started</Link>
        </>
      }
    />
  )
}
