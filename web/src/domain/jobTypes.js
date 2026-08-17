export const JOB_TYPES = {
  send_email: {
    label: 'Send email',
    summary: 'Simulate delivering an email message.',
    initialPayload: {
      to: 'learner@example.com',
      subject: 'Async processing demo',
      body: 'This message was processed by a background worker.',
    },
  },
  report_generation: {
    label: 'Generate report',
    summary: 'Simulate producing a report in a selected format.',
    initialPayload: {
      report: 'monthly-summary',
      format: 'pdf',
    },
  },
  data_cleanup: {
    label: 'Clean data',
    summary: 'Simulate cleaning records older than a threshold.',
    initialPayload: {
      scope: 'expired-sessions',
      older_than_days: 30,
    },
  },
}

export function initialPayloadFor(type) {
  return { ...JOB_TYPES[type].initialPayload }
}
