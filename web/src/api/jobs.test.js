import { afterEach, describe, expect, it, vi } from 'vitest'
import { APIError, createJob, getJob } from './jobs.js'

afterEach(() => vi.unstubAllGlobals())

describe('jobs API', () => {
  it('submits the selected type and payload', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: 'job-1', status: 'pending' }, 202))
    vi.stubGlobal('fetch', fetchMock)

    const result = await createJob('data_cleanup', { scope: 'expired', older_than_days: 30 })

    expect(result.status).toBe('pending')
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/jobs', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({
        type: 'data_cleanup',
        payload: { scope: 'expired', older_than_days: 30 },
      }),
    }))
  })

  it('encodes a job ID and exposes safe API errors', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({
      error: { code: 'job_not_found', message: 'job was not found' },
    }, 404, { 'X-Request-ID': 'request-123' }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(getJob(' job/id ')).rejects.toMatchObject({
      name: 'APIError',
      status: 404,
      code: 'job_not_found',
      requestID: 'request-123',
    })
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/jobs/job%2Fid', {})
    expect(APIError.prototype).toBeInstanceOf(Error)
  })
})

function jsonResponse(body, status, headers = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  })
}
