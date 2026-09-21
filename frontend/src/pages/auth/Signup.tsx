import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AuthForm } from '@/components/ui/AuthForm'
import { signup as signupRequest } from '@/services/authService'
import { useAuth } from '@/hooks/useAuth'

export function Signup() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const isHostFlow = params.get('role') === 'host'

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const res = await signupRequest(email, password)
      login(res)
      // There is no separate host account type on the backend — the same signup
      // is used, then a machine gets registered as a second step.
      navigate(isHostFlow ? '/host/onboarding' : '/dashboard')
    } catch {
      setError('Could not create an account. That email may already be in use.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthForm
      title={isHostFlow ? 'Become a Host' : 'Get Started'}
      subtitle={isHostFlow ? 'Create your TOJI account to connect a GPU.' : 'Create your TOJI account.'}
      email={email}
      password={password}
      onEmailChange={setEmail}
      onPasswordChange={setPassword}
      onSubmit={handleSubmit}
      submitLabel="Create Account"
      loading={loading}
      error={error}
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="text-accent hover:underline">Log In</Link>
        </>
      }
    />
  )
}
