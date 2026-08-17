import { useCallback, useEffect, useRef, useState } from 'react'
import { getJob } from '../api/jobs.js'

const TERMINAL_STATUSES = new Set(['success', 'failed'])
const POLL_INTERVAL_MS = 1000

export function useJobTracker() {
  const [job, setJob] = useState(null)
  const [error, setError] = useState('')
  const [isPolling, setIsPolling] = useState(false)
  const trackingGeneration = useRef(0)

  const stopTracking = useCallback(() => {
    trackingGeneration.current += 1
    setIsPolling(false)
  }, [])

  const trackJob = useCallback(async (jobID, initialJob = null) => {
    const generation = trackingGeneration.current + 1
    trackingGeneration.current = generation
    setJob(initialJob)
    setError('')
    setIsPolling(true)

    while (trackingGeneration.current === generation) {
      try {
        const currentJob = await getJob(jobID)
        if (trackingGeneration.current !== generation) return

        setJob(currentJob)
        if (TERMINAL_STATUSES.has(currentJob.status)) {
          setIsPolling(false)
          return
        }
      } catch (trackingError) {
        if (trackingGeneration.current !== generation) return
        setError(trackingError.message)
        setIsPolling(false)
        return
      }

      await delay(POLL_INTERVAL_MS)
    }
  }, [])

  useEffect(() => stopTracking, [stopTracking])

  return {
    job,
    error,
    isPolling,
    trackJob,
    stopTracking,
  }
}

function delay(milliseconds) {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds))
}
