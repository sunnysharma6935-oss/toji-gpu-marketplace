import { MainLayout } from '@/layouts/MainLayout'

const customerSteps = [
  'Choose a GPU',
  'Add TOJI balance',
  'Start your rental',
  'Run your workload',
  'Stop when finished',
  'Pay only for actual usage',
]

const hostSteps = [
  'Connect your machine',
  'Add your GPU',
  'Set your price',
  'Make it available',
  'Earn when customers use it',
]

function StepList({ title, steps }: { title: string; steps: string[] }) {
  return (
    <div>
      <p className="label-mono mb-6">{title}</p>
      <ol className="space-y-4">
        {steps.map((step, i) => (
          <li key={step} className="flex items-center gap-4">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent text-sm font-bold text-ink">
              {i + 1}
            </span>
            <span className="text-sm text-paper">{step}</span>
          </li>
        ))}
      </ol>
    </div>
  )
}

export function HowItWorks() {
  return (
    <MainLayout>
      <section className="container-page py-16">
        <h1 className="font-sans text-3xl font-bold sm:text-4xl">How It Works</h1>
        <div className="mt-12 grid gap-12 sm:grid-cols-2">
          <StepList title="For Customers" steps={customerSteps} />
          <StepList title="For Hosts" steps={hostSteps} />
        </div>
      </section>
    </MainLayout>
  )
}
