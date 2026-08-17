#!/bin/sh

set -eu

api_base_url="${API_BASE_URL:-http://localhost:8080}"
poll_limit="${LIFECYCLE_POLL_LIMIT:-20}"

printf 'Checking React interface...\n'
ui_response="$(curl -fsS "${api_base_url}/")"
if ! printf '%s' "$ui_response" | grep -q 'id="root"'; then
	printf 'React interface was not served by %s\n' "$api_base_url" >&2
	exit 1
fi

submit_job() {
	payload="$1"
	response="$(curl -fsS \
		-X POST \
		-H 'Content-Type: application/json' \
		-H 'X-Request-ID: lifecycle-check' \
		-d "$payload" \
		"${api_base_url}/api/v1/jobs")"

	job_id="$(printf '%s' "$response" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')"
	if [ -z "$job_id" ]; then
		printf 'Could not read job ID from response: %s\n' "$response" >&2
		exit 1
	fi

	printf '%s\n' "$job_id"
}

wait_for_status() {
	job_id="$1"
	expected_status="$2"
	attempt=1

	while [ "$attempt" -le "$poll_limit" ]; do
		response="$(curl -fsS \
			-H "X-Request-ID: lifecycle-check-${job_id}" \
			"${api_base_url}/api/v1/jobs/${job_id}")"
		status="$(printf '%s' "$response" | sed -n 's/.*"status":"\([^"]*\)".*/\1/p')"

		if [ "$status" = "$expected_status" ]; then
			printf '%s\n' "$response"
			return 0
		fi

		if [ "$status" = "failed" ] || [ "$status" = "success" ]; then
			printf 'Job %s reached unexpected status %s\n' "$job_id" "$status" >&2
			exit 1
		fi

		sleep 1
		attempt=$((attempt + 1))
	done

	printf 'Job %s did not reach %s after %s checks\n' \
		"$job_id" "$expected_status" "$poll_limit" >&2
	exit 1
}

printf 'Checking successful lifecycle...\n'
successful_job_id="$(submit_job \
	'{"type":"report_generation","payload":{"report":"day-8-check","format":"pdf"}}')"
wait_for_status "$successful_job_id" "success" >/dev/null
printf 'Job %s reached success.\n' "$successful_job_id"

printf 'Checking retry and terminal-failure lifecycle...\n'
failed_job_id="$(submit_job \
	'{"type":"report_generation","payload":{"report":"day-8-check","format":"xml"}}')"
failed_response="$(wait_for_status "$failed_job_id" "failed")"

if ! printf '%s' "$failed_response" | grep -q '"retry_count":3'; then
	printf 'Failed job did not report retry_count=3: %s\n' "$failed_response" >&2
	exit 1
fi

if ! printf '%s' "$failed_response" | grep -q '"last_error":"'; then
	printf 'Failed job did not report last_error: %s\n' "$failed_response" >&2
	exit 1
fi

printf 'Job %s reached failed after 3 retries.\n' "$failed_job_id"
printf 'Lifecycle verification passed.\n'
