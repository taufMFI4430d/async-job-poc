const JOBS_ENDPOINT = '/api/v1/jobs'

export async function createJob(type, payload) {
  return request(JOBS_ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type, payload }),
  })
}

export async function getJob(jobID) {
  return request(`${JOBS_ENDPOINT}/${encodeURIComponent(jobID.trim())}`)
}

async function request(url, options = {}) {
  const response = await fetch(url, options)
  const requestID = response.headers.get('X-Request-ID')
  const body = await parseJSON(response)

  if (!response.ok) {
    const message = body?.error?.message || `Request failed with status ${response.status}`
    throw new APIError(message, response.status, body?.error?.code, requestID)
  }

  return body
}

async function parseJSON(response) {
  try {
    return await response.json()
  } catch {
    return null
  }
}

export class APIError extends Error {
  constructor(message, status, code, requestID) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
    this.requestID = requestID
  }
}
