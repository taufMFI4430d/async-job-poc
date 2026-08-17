import { useState } from 'react'

export function JobLookup({ onLookup, isPolling }) {
  const [jobID, setJobID] = useState('')

  function submit(event) {
    event.preventDefault()
    const normalizedID = jobID.trim()
    if (normalizedID) onLookup(normalizedID)
  }

  return (
    <form className="lookup" onSubmit={submit}>
      <label htmlFor="job-id">Already have a job ID?</label>
      <div className="lookup-row">
        <input
          id="job-id"
          placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
          value={jobID}
          onChange={(event) => setJobID(event.target.value)}
          required
        />
        <button type="submit" disabled={isPolling && !jobID.trim()}>Check status</button>
      </div>
    </form>
  )
}
