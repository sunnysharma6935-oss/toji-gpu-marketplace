import { Link } from 'react-router-dom'
import { MainLayout } from '@/layouts/MainLayout'
import { Button } from '@/components/ui/Button'

const steps = [
  { title: 'Install TOJI Host Agent', desc: 'One download, runs quietly in the background.' },
  { title: 'Connect your machine', desc: 'The agent detects your GPU automatically.' },
  { title: 'Choose your hourly price', desc: 'Set your own rate, or use a suggested one.' },
  { title: 'Go online', desc: 'Your GPU appears in the marketplace.' },
  { title: 'Earn from rentals', desc: 'Get paid for every hour your GPU is rented.' },
]

export function HostLanding() {
  return (
    <MainLayout>
      <section className="container-page py-20 text-center">
        <h1 className="mx-auto max-w-2xl font-sans text-4xl font-bold sm:text-5xl">
          Turn Your Idle GPU Into Income.
        </h1>
        <p className="mx-auto mt-5 max-w-lg text-base text-muted">
          Your GPU can earn while you're not using it.
        </p>
        <div className="mt-8">
          <Link to="/signup?role=host">
            <Button>Become a Host</Button>
          </Link>
        </div>
      </section>

      <section className="container-page border-t border-border py-16">
        <div className="grid gap-6 sm:grid-cols-5">
          {steps.map((step, i) => (
            <div key={step.title} className="card p-5">
              <span className="label-mono text-accent">Step {i + 1}</span>
              <h3 className="mt-3 text-sm font-bold">{step.title}</h3>
              <p className="mt-2 text-xs text-muted-dim">{step.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-t border-border">
        <div className="container-page py-14 text-center">
          <p className="mx-auto max-w-md text-lg font-semibold">
            You don't need to be technical.
          </p>
          <p className="mx-auto mt-2 max-w-md text-sm text-muted-dim">
            The host agent handles Docker, isolation, networking, and billing math —
            you'll never see any of it.
          </p>
        </div>
      </section>
    </MainLayout>
  )
}
