import { DashboardLayout } from '@/layouts/DashboardLayout'
import { getSession } from '@/services/authService'

export function Account() {
  const session = getSession()
  return (
    <DashboardLayout>
      <h1 className="font-sans text-2xl font-bold">Account</h1>
      <div className="card mt-6 max-w-md p-6">
        <p className="label-mono mb-1">User ID</p>
        <p className="font-mono text-xs text-muted">{session?.userId}</p>
      </div>
    </DashboardLayout>
  )
}
