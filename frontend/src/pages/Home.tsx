import { Link } from 'react-router-dom'
import { MainLayout } from '@/layouts/MainLayout'
import { Button } from '@/components/ui/Button'

export function Home() {
  return (
    <MainLayout>
      <section className="container-page flex flex-col items-center py-24 text-center sm:py-32">
        <h1 className="max-w-3xl font-sans text-4xl font-bold leading-tight sm:text-6xl">
          Rent GPU Power.
          <br />
          Run AI. Pay As You Go.
        </h1>
        <p className="mt-6 max-w-xl text-base text-muted sm:text-lg">
          Access powerful GPUs from independent providers and run AI workloads
          without buying expensive hardware.
        </p>
        <div className="mt-9 flex flex-col gap-3 sm:flex-row">
          <Link to="/marketplace">
            <Button className="w-full sm:w-auto">Explore GPUs</Button>
          </Link>
          <Link to="/host">
            <Button variant="secondary" className="w-full sm:w-auto">Host Your GPU</Button>
          </Link>
        </div>
      </section>

      <section className="container-page grid gap-4 border-t border-border py-16 sm:grid-cols-3">
        <div className="card p-6">
          <p className="label-mono mb-3">Rent</p>
          <h3 className="font-sans text-lg font-bold">By the hour</h3>
          <p className="mt-2 text-sm text-muted">
            Choose a GPU, add TOJI balance, and start a rental in minutes.
          </p>
        </div>
        <div className="card p-6">
          <p className="label-mono mb-3">Run</p>
          <h3 className="font-sans text-lg font-bold">Real workloads</h3>
          <p className="mt-2 text-sm text-muted">
            AI inference, image generation, ComfyUI, Jupyter, small fine-tuning.
          </p>
        </div>
        <div className="card p-6">
          <p className="label-mono mb-3">Pay</p>
          <h3 className="font-sans text-lg font-bold">Only for usage</h3>
          <p className="mt-2 text-sm text-muted">
            Billed to the second. Stop anytime — billing stops with it.
          </p>
        </div>
      </section>

      <section className="border-t border-border">
        <div className="container-page flex flex-col items-center gap-4 py-16 text-center sm:flex-row sm:justify-between sm:text-left">
          <div>
            <h2 className="font-sans text-2xl font-bold">Have a GPU sitting idle?</h2>
            <p className="mt-2 text-sm text-muted">Turn it into income — you don't need to be technical.</p>
          </div>
          <Link to="/host">
            <Button>Host Your GPU</Button>
          </Link>
        </div>
      </section>
    </MainLayout>
  )
}
