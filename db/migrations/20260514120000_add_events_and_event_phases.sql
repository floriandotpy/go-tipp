-- migrate:up
CREATE TABLE `events` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `slug` varchar(255) NOT NULL,
  `api_base_url` varchar(255) NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 0,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `events_uc_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `event_phases` (
  `id` int NOT NULL AUTO_INCREMENT,
  `event_id` int NOT NULL,
  `number` int NOT NULL,
  `title` varchar(255) NOT NULL,
  `api_path` varchar(512) NOT NULL,
  `phase_type` varchar(50) NOT NULL,
  `start` datetime NOT NULL,
  `end` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `event_phases_uc_event_number` (`event_id`, `number`),
  CONSTRAINT `fk_event_phases_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_phase_type` CHECK (`phase_type` IN ('phase_group', 'phase_ko')),
  CONSTRAINT `chk_start_before_end` CHECK (`start` < `end`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed Euro 2024 event
INSERT INTO `events` (`name`, `slug`, `api_base_url`, `is_active`)
VALUES ('Euro 2024', 'euro-2024', 'https://api.openligadb.de', TRUE);

-- Seed 7 phases for Euro 2024
INSERT INTO `event_phases` (`event_id`, `number`, `title`, `api_path`, `phase_type`, `start`, `end`) VALUES
(LAST_INSERT_ID(), 1, 'Gruppenphase 1', '/getmatchdata/em/2024/1', 'phase_group', '2024-06-14 00:00:00', '2024-06-18 23:59:59'),
(LAST_INSERT_ID(), 2, 'Gruppenphase 2', '/getmatchdata/em/2024/2', 'phase_group', '2024-06-19 00:00:00', '2024-06-22 23:59:59'),
(LAST_INSERT_ID(), 3, 'Gruppenphase 3', '/getmatchdata/em/2024/3', 'phase_group', '2024-06-23 00:00:00', '2024-06-28 23:59:59'),
(LAST_INSERT_ID(), 4, 'Achtelfinale',   '/getmatchdata/em/2024/4', 'phase_ko',    '2024-06-29 00:00:00', '2024-07-04 23:59:59'),
(LAST_INSERT_ID(), 5, 'Viertelfinale',  '/getmatchdata/em/2024/5', 'phase_ko',    '2024-07-05 00:00:00', '2024-07-08 23:59:59'),
(LAST_INSERT_ID(), 6, 'Halbfinale',     '/getmatchdata/em/2024/6', 'phase_ko',    '2024-07-09 00:00:00', '2024-07-13 23:59:59'),
(LAST_INSERT_ID(), 7, 'Finale',         '/getmatchdata/em/2024/7', 'phase_ko',    '2024-07-14 00:00:00', '2024-07-14 23:59:59');

-- Add event_id column to matches and backfill with Euro 2024 ID
ALTER TABLE `matches` ADD COLUMN `event_id` int NOT NULL DEFAULT 1;
UPDATE `matches` SET `event_id` = (SELECT `id` FROM `events` WHERE `slug` = 'euro-2024');
ALTER TABLE `matches` ADD CONSTRAINT `fk_matches_event` FOREIGN KEY (`event_id`) REFERENCES `events` (`id`);

-- migrate:down
ALTER TABLE `matches` DROP FOREIGN KEY `fk_matches_event`;
ALTER TABLE `matches` DROP COLUMN `event_id`;
DROP TABLE IF EXISTS `event_phases`;
DROP TABLE IF EXISTS `events`;
