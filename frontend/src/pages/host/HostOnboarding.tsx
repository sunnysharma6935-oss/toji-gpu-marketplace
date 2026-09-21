import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { DashboardLayout } from '@/layouts/DashboardLayout'
import { Button } from '@/components/ui/Button'
import { registerHost } from '@/services/hostService'

// The only fully real step in host onboarding today: POST /hosts/register exists and
// works. Hardware detection, GPU reporting, and going "online" all happen via the
// separately-running host agent binary (backend/internal/api/handlers.go: HostHeartbeat,
// GPUReport) — not via this web UI — so this page's job is specifically to get the
// human a machine_label and an agent_token to paste into that agent, and nothing more.
export function HostOnboarding() {
  const [machineLabel, setMachineLabel] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<{ host_id: string; agent_token: string } | null>(null)
  const navigate = useNavigate()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const res = await registerHost(machineLabel)
      setResult(res)
    } catch {
      setError('Could not register this machine. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <DashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Connect Your Machine</h1>
      <p className="mt-2 max-w-md text-sm text-muted">
        Give this machine a name. You'll use the token below to connect the TOJI host
        agent — it detects your GPU automatically from there.
      </p>

      {!result ? (
        <form onSubmit={handleSubmit} className="card mt-6 max-w-md space-y-4 p-6">
          <div>
            <label className="label-mono mb-2 block">Machine Name</label>
            <input
              value={machineLabel}
              onChange={(e) => setMachineLabel(e.target.value)}
              placeholder="e.g. Gaming PC 01"
              required
              className="w-full rounded border border-border bg-ink px-3 py-2.5 text-sm text-paper focus:border-border-strong focus:outline-none"
            />
          </div>
          {error && <p className="text-sm text-warn">{error}</p>}
          <Button type="submit" disabled={loading} className="w-full">
            {loading ? 'Registering…' : 'Register Machine'}
          </Button>
        </form>
      ) : (
        <div className="card mt-6 max-w-md p-6">
          <p className="text-sm font-semibold text-success">Machine registered.</p>
          <p className="mt-4 label-mono">Agent Token</p>
          <p className="mt-1 break-all rounded bg-ink p-3 font-mono text-xs text-paper">
            {result.agent_token}
          </p>
          <p className="mt-3 text-xs text-muted-dim">
            Shown once. Set this as <span className="font-mono">AGENT_TOKEN</span> when
            starting the TOJI host agent on this machine.
          </p>
          <Button onClick={() => navigate('/host/dashboard')} variant="secondary" className="mt-5 w-full">
            Go to Host Dashboard
          </Button>
        </div>
      )}
    </DashboardLayout>
  )
}
