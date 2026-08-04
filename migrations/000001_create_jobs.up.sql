CREATE TABLE jobs (
    id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payload JSON NOT NULL,
    retry_count TINYINT UNSIGNED NOT NULL DEFAULT 0,
    max_retries TINYINT UNSIGNED NOT NULL DEFAULT 3,
    last_error TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    started_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,

    PRIMARY KEY (id),

    CONSTRAINT chk_jobs_type
        CHECK (type IN ('send_email', 'report_generation', 'data_cleanup')),
    CONSTRAINT chk_jobs_status
        CHECK (status IN ('pending', 'processing', 'success', 'failed')),
    CONSTRAINT chk_jobs_payload
        CHECK (JSON_TYPE(payload) = 'OBJECT' AND JSON_LENGTH(payload) > 0),
    CONSTRAINT chk_jobs_max_retries
        CHECK (max_retries > 0),
    CONSTRAINT chk_jobs_retry_count
        CHECK (retry_count <= max_retries),

    INDEX idx_jobs_status_created_at (status, created_at),
    INDEX idx_jobs_created_at (created_at)
) ENGINE = InnoDB
  DEFAULT CHARACTER SET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci;
