import { useState } from 'react'
import { createJob } from '../api/jobs.js'
import { initialPayloadFor, JOB_TYPES } from '../domain/jobTypes.js'

export function JobForm({ onCreated }) {
  const [type, setType] = useState('send_email')
  const [payload, setPayload] = useState(initialPayloadFor('send_email'))
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')

  function selectType(event) {
    const selectedType = event.target.value
    setType(selectedType)
    setPayload(initialPayloadFor(selectedType))
    setError('')
  }

  function updateField(event) {
    const { name, type: inputType, value } = event.target
    setPayload((current) => ({
      ...current,
      [name]: inputType === 'number' ? Number(value) : value,
    }))
  }

  async function submit(event) {
    event.preventDefault()
    setIsSubmitting(true)
    setError('')

    try {
      const created = await createJob(type, payload)
      onCreated(created)
    } catch (submissionError) {
      setError(submissionError.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="panel job-form" onSubmit={submit}>
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Create</p>
          <h2>Submit a background job</h2>
        </div>
        <span className="step-number">01</span>
      </div>

      <label>
        Job type
        <select name="job_type" value={type} onChange={selectType}>
          {Object.entries(JOB_TYPES).map(([value, definition]) => (
            <option key={value} value={value}>{definition.label}</option>
          ))}
        </select>
      </label>
      <p className="field-hint">{JOB_TYPES[type].summary}</p>

      <div className="dynamic-fields">
        {type === 'send_email' && (
          <>
            <label>Recipient<input name="to" type="email" required value={payload.to} onChange={updateField} /></label>
            <label>Subject<input name="subject" required maxLength="200" value={payload.subject} onChange={updateField} /></label>
            <label>Message<textarea name="body" required rows="4" value={payload.body} onChange={updateField} /></label>
          </>
        )}

        {type === 'report_generation' && (
          <>
            <label>Report name<input name="report" required maxLength="100" value={payload.report} onChange={updateField} /></label>
            <label>Format<select name="format" value={payload.format} onChange={updateField}><option value="pdf">PDF</option><option value="csv">CSV</option><option value="json">JSON</option></select></label>
          </>
        )}

        {type === 'data_cleanup' && (
          <>
            <label>Cleanup scope<input name="scope" required maxLength="100" value={payload.scope} onChange={updateField} /></label>
            <label>Older than (days)<input name="older_than_days" type="number" required min="1" max="3650" value={payload.older_than_days} onChange={updateField} /></label>
          </>
        )}
      </div>

      {error && <p className="inline-error" role="alert">{error}</p>}
      <button className="primary-button" type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Submitting…' : 'Submit job'}
      </button>
    </form>
  )
}
