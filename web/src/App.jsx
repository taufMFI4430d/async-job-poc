import { JobForm } from './components/JobForm.jsx'
import { JobLookup } from './components/JobLookup.jsx'
import { JobStatus } from './components/JobStatus.jsx'
import { useJobTracker } from './hooks/useJobTracker.js'
import './styles.css'

export default function App() {
  const tracker = useJobTracker()

  function handleCreated(job) {
    tracker.trackJob(job.id, job)
  }

  return (
    <main>
      <header className="hero">
        <div className="brand"><span className="brand-mark">AQ</span><span>Async Queue</span></div>
        <div className="hero-copy">
          <p className="eyebrow">Go · Redis · MySQL</p>
          <h1>Send work now.<br /><em>Process it later.</em></h1>
          <p className="hero-summary">A small control room for creating background jobs and watching five workers move them through a durable lifecycle.</p>
        </div>
        <div className="system-state"><span className="online-dot" />System interface ready</div>
      </header>

      <section className="workspace">
        <JobForm onCreated={handleCreated} />
        <JobStatus job={tracker.job} error={tracker.error} isPolling={tracker.isPolling} />
      </section>

      <JobLookup onLookup={tracker.trackJob} isPolling={tracker.isPolling} />

      <footer>
        <span>pending</span><span>→</span><span>processing</span><span>→</span><span>success / failed</span>
      </footer>
    </main>
  )
}
