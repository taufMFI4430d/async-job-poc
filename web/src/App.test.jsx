import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import App from './App.jsx'

afterEach(() => vi.unstubAllGlobals())

it('submits a report job and renders its terminal status', async () => {
  const completedJob = {
    id: '6d3d991b-8270-4ea3-8e5c-8e8d692d6c5e',
    type: 'report_generation',
    status: 'success',
    retry_count: 0,
    max_retries: 3,
    created_at: '2026-08-17T04:09:20Z',
    completed_at: '2026-08-17T04:09:22Z',
    last_error: null,
  }
  const fetchMock = vi.fn()
    .mockResolvedValueOnce(jsonResponse(completedJob, 202))
    .mockResolvedValueOnce(jsonResponse(completedJob, 200))
  vi.stubGlobal('fetch', fetchMock)

  render(<App />)

  fireEvent.change(screen.getByLabelText('Job type'), { target: { value: 'report_generation' } })
  fireEvent.change(screen.getByLabelText('Report name'), { target: { value: 'weekly-activity' } })
  fireEvent.click(screen.getByRole('button', { name: 'Submit job' }))

  await waitFor(() => expect(screen.getByText('Completed successfully')).toBeInTheDocument())
  expect(screen.getByText(completedJob.id)).toBeInTheDocument()
  expect(fetchMock).toHaveBeenCalledTimes(2)

  const submission = JSON.parse(fetchMock.mock.calls[0][1].body)
  expect(submission).toEqual({
    type: 'report_generation',
    payload: { report: 'weekly-activity', format: 'pdf' },
  })
})

function jsonResponse(body, status) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}
