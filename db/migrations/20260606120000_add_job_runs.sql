-- migrate:up
CREATE TABLE `job_runs` (
  `id` int NOT NULL AUTO_INCREMENT,
  `job_name` varchar(50) NOT NULL,
  `status` varchar(20) NOT NULL,
  `summary` varchar(500) NOT NULL,
  `details` json DEFAULT NULL,
  `started_at` datetime NOT NULL,
  `finished_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_job_runs_job_started` (`job_name`, `started_at`),
  CONSTRAINT `chk_job_status` CHECK (`status` IN ('success_noop', 'success_changed', 'error'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- migrate:down
DROP TABLE IF EXISTS `job_runs`;
