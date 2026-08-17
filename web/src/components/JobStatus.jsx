const STATUS_LABELS = {
  pending: 'Waiting in queue',
  processing: 'Worker processing',
  success: 'Completed successfully',
  failed: 'Processing failed',
}

export function JobStatus({ job, error, isPolling }) {
  return (
    <section className="panel status-panel" aria-live="polite">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Monitor</p>
          <h2>Job lifecycle</h2>
        </div>
        <span className="step-number">02</span>
      </div>

      {!job && !error && (
        <div className="empty-state">
          <div className="queue-illustration"><span /><span /><span /></div>
          <h3>No job selected</h3>
          <p>Submit a job or enter an existing ID to watch its persisted state.</p>
        </div>
      )}

      {error && <p className="inline-error" role="alert">{error}</p>}

      {job && (
        <div className="job-details">
          <div className="status-line">
            <span className={`status-dot status-${job.status}`} />
            <div>
              <p className="status-name">{STATUS_LABELS[job.status] || job.status}</p>
              <p className="status-code">{job.status}</p>
            </div>
            {isPolling && <span className="polling-badge">Live</span>}
          </div>

          <dl>
            <div><dt>Job ID</dt><dd className="monospace">{job.id}</dd></div>
            <div><dt>Type</dt><dd>{job.type}</dd></div>
            <div><dt>Retries</dt><dd>{job.retry_count} / {job.max_retries}</dd></div>
            <div><dt>Created</dt><dd>{formatTime(job.created_at)}</dd></div>
            {job.completed_at && <div><dt>Completed</dt><dd>{formatTime(job.completed_at)}</dd></div>}
          </dl>

          {job.last_error && (
            <div className="failure-box">
              <strong>Last processing error</strong>
              <p>{job.last_error}</p>
            </div>
          )}
        </div>
      )}
    </section>
  )
}

function formatTime(value) {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(new Date(value))
}
