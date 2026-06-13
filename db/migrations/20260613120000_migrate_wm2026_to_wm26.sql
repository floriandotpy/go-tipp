-- migrate:up
-- Migration: Switch WM 2026 event from wm2026 to wm26 API dataset.
-- The wm26 dataset is better maintained (proper goal minutes, per-goal records).
-- All 72 matches map 1:1 between datasets; only match IDs and minor spellings differ.

-- Step 1: Remap api_match_id values (old wm2026 IDs -> new wm26 IDs).
-- Strategy: capture internal match IDs, null out old values (unique index), set new values.

CREATE TEMPORARY TABLE `_migration_map` (
  `internal_id` int NOT NULL,
  `new_api_id` int NOT NULL,
  PRIMARY KEY (`internal_id`)
);

INSERT INTO `_migration_map` (`internal_id`, `new_api_id`)
SELECT m.`id`, v.`new_id`
FROM `matches` m
INNER JOIN (
  SELECT 81464 AS old_id, 80099 AS new_id UNION ALL
  SELECT 81465, 80100 UNION ALL
  SELECT 81466, 80101 UNION ALL
  SELECT 81484, 80102 UNION ALL
  SELECT 81485, 80104 UNION ALL
  SELECT 81486, 80105 UNION ALL
  SELECT 81487, 80106 UNION ALL
  SELECT 81488, 80103 UNION ALL
  SELECT 81489, 80107 UNION ALL
  SELECT 81490, 80108 UNION ALL
  SELECT 81491, 80109 UNION ALL
  SELECT 81492, 80110 UNION ALL
  SELECT 81493, 80111 UNION ALL
  SELECT 81494, 80112 UNION ALL
  SELECT 81495, 80113 UNION ALL
  SELECT 81496, 80114 UNION ALL
  SELECT 81497, 80116 UNION ALL
  SELECT 81498, 80117 UNION ALL
  SELECT 81499, 80118 UNION ALL
  SELECT 81500, 80115 UNION ALL
  SELECT 81501, 80119 UNION ALL
  SELECT 81502, 80120 UNION ALL
  SELECT 81503, 80121 UNION ALL
  SELECT 81504, 80122 UNION ALL
  SELECT 81505, 80124 UNION ALL
  SELECT 81506, 80125 UNION ALL
  SELECT 81507, 80126 UNION ALL
  SELECT 81508, 80127 UNION ALL
  SELECT 81509, 80129 UNION ALL
  SELECT 81510, 80130 UNION ALL
  SELECT 81511, 80131 UNION ALL
  SELECT 81512, 80128 UNION ALL
  SELECT 81513, 80133 UNION ALL
  SELECT 81514, 80134 UNION ALL
  SELECT 81515, 80135 UNION ALL
  SELECT 81516, 80132 UNION ALL
  SELECT 81517, 80136 UNION ALL
  SELECT 81518, 80137 UNION ALL
  SELECT 81519, 80138 UNION ALL
  SELECT 81520, 80139 UNION ALL
  SELECT 81521, 80140 UNION ALL
  SELECT 81522, 80141 UNION ALL
  SELECT 81523, 80142 UNION ALL
  SELECT 81524, 80143 UNION ALL
  SELECT 81525, 80144 UNION ALL
  SELECT 81526, 80145 UNION ALL
  SELECT 81527, 80146 UNION ALL
  SELECT 81528, 80147 UNION ALL
  SELECT 81529, 80148 UNION ALL
  SELECT 81530, 80149 UNION ALL
  SELECT 81531, 80151 UNION ALL
  SELECT 81532, 80150 UNION ALL
  SELECT 81533, 80153 UNION ALL
  SELECT 81534, 80152 UNION ALL
  SELECT 81535, 80155 UNION ALL
  SELECT 81536, 80154 UNION ALL
  SELECT 81537, 80156 UNION ALL
  SELECT 81538, 80157 UNION ALL
  SELECT 81539, 80158 UNION ALL
  SELECT 81540, 80159 UNION ALL
  SELECT 81541, 80160 UNION ALL
  SELECT 81542, 80161 UNION ALL
  SELECT 81543, 80163 UNION ALL
  SELECT 81544, 80162 UNION ALL
  SELECT 81545, 80164 UNION ALL
  SELECT 81546, 80165 UNION ALL
  SELECT 81547, 80166 UNION ALL
  SELECT 81548, 80167 UNION ALL
  SELECT 81549, 80168 UNION ALL
  SELECT 81550, 80169 UNION ALL
  SELECT 81551, 80171 UNION ALL
  SELECT 81552, 80170
) v ON m.`api_match_id` = v.`old_id`;

-- Clear old api_match_ids (required due to unique index)
UPDATE `matches` m
  INNER JOIN `_migration_map` map ON m.`id` = map.`internal_id`
SET m.`api_match_id` = NULL;

-- Set new api_match_ids
UPDATE `matches` m
  INNER JOIN `_migration_map` map ON m.`id` = map.`internal_id`
SET m.`api_match_id` = map.`new_api_id`;

DROP TEMPORARY TABLE `_migration_map`;

-- Step 2: Fix team name spellings to match new API
UPDATE `matches` SET `team_a` = 'Bosnien und Herzegowina' WHERE `team_a` = 'Bosnien-Herzegowina';
UPDATE `matches` SET `team_b` = 'Bosnien und Herzegowina' WHERE `team_b` = 'Bosnien-Herzegowina';
UPDATE `matches` SET `team_a` = 'Saudi Arabien' WHERE `team_a` = 'Saudi-Arabien';
UPDATE `matches` SET `team_b` = 'Saudi Arabien' WHERE `team_b` = 'Saudi-Arabien';

-- Step 3: Update event_phases titles to match new API group names
UPDATE `event_phases` ep
  INNER JOIN `events` e ON ep.`event_id` = e.`id`
SET ep.`title` = 'Gruppenphase 1'
WHERE e.`api_base_url` LIKE '%wm2026%' AND ep.`title` = '1. Runde';

UPDATE `event_phases` ep
  INNER JOIN `events` e ON ep.`event_id` = e.`id`
SET ep.`title` = 'Gruppenphase 2'
WHERE e.`api_base_url` LIKE '%wm2026%' AND ep.`title` = '2. Runde';

UPDATE `event_phases` ep
  INNER JOIN `events` e ON ep.`event_id` = e.`id`
SET ep.`title` = 'Gruppenphase 3'
WHERE e.`api_base_url` LIKE '%wm2026%' AND ep.`title` = '3. Runde';

-- Step 4: Update the event's API base URL
UPDATE `events`
SET `api_base_url` = REPLACE(`api_base_url`, 'wm2026', 'wm26')
WHERE `api_base_url` LIKE '%wm2026%';

-- Step 5: Delete goals for already-finished matches so fetch-results re-syncs them
-- with proper per-goal data (minutes, scorer IDs) from the new API.
DELETE g FROM `goals` g
  INNER JOIN `matches` m ON g.`match_id` = m.`id`
WHERE m.`api_match_id` IN (80099, 80100, 80101, 80102);


-- migrate:down

-- Reverse Step 4: Restore old API URL (do this first so phase title reversal can match)
UPDATE `events`
SET `api_base_url` = REPLACE(`api_base_url`, '/wm26/', '/wm2026/')
WHERE `api_base_url` LIKE '%/wm26/%';

-- Reverse Step 3: Restore old phase titles
UPDATE `event_phases` ep
  INNER JOIN `events` e ON ep.`event_id` = e.`id`
SET ep.`title` = '1. Runde'
WHERE e.`api_base_url` LIKE '%wm2026%' AND ep.`title` = 'Gruppenphase 1';

UPDATE `event_phases` ep
  INNER JOIN `events` e ON ep.`event_id` = e.`id`
SET ep.`title` = '2. Runde'
WHERE e.`api_base_url` LIKE '%wm2026%' AND ep.`title` = 'Gruppenphase 2';

UPDATE `event_phases` ep
  INNER JOIN `events` e ON ep.`event_id` = e.`id`
SET ep.`title` = '3. Runde'
WHERE e.`api_base_url` LIKE '%wm2026%' AND ep.`title` = 'Gruppenphase 3';

-- Reverse Step 2: Restore old team name spellings
UPDATE `matches` SET `team_a` = 'Bosnien-Herzegowina' WHERE `team_a` = 'Bosnien und Herzegowina';
UPDATE `matches` SET `team_b` = 'Bosnien-Herzegowina' WHERE `team_b` = 'Bosnien und Herzegowina';
UPDATE `matches` SET `team_a` = 'Saudi-Arabien' WHERE `team_a` = 'Saudi Arabien';
UPDATE `matches` SET `team_b` = 'Saudi-Arabien' WHERE `team_b` = 'Saudi Arabien';

-- Reverse Step 1: Remap api_match_ids back to old values
CREATE TEMPORARY TABLE `_migration_map_down` (
  `internal_id` int NOT NULL,
  `old_api_id` int NOT NULL,
  PRIMARY KEY (`internal_id`)
);

INSERT INTO `_migration_map_down` (`internal_id`, `old_api_id`)
SELECT m.`id`, v.`old_id`
FROM `matches` m
INNER JOIN (
  SELECT 80099 AS new_id, 81464 AS old_id UNION ALL
  SELECT 80100, 81465 UNION ALL
  SELECT 80101, 81466 UNION ALL
  SELECT 80102, 81484 UNION ALL
  SELECT 80104, 81485 UNION ALL
  SELECT 80105, 81486 UNION ALL
  SELECT 80106, 81487 UNION ALL
  SELECT 80103, 81488 UNION ALL
  SELECT 80107, 81489 UNION ALL
  SELECT 80108, 81490 UNION ALL
  SELECT 80109, 81491 UNION ALL
  SELECT 80110, 81492 UNION ALL
  SELECT 80111, 81493 UNION ALL
  SELECT 80112, 81494 UNION ALL
  SELECT 80113, 81495 UNION ALL
  SELECT 80114, 81496 UNION ALL
  SELECT 80116, 81497 UNION ALL
  SELECT 80117, 81498 UNION ALL
  SELECT 80118, 81499 UNION ALL
  SELECT 80115, 81500 UNION ALL
  SELECT 80119, 81501 UNION ALL
  SELECT 80120, 81502 UNION ALL
  SELECT 80121, 81503 UNION ALL
  SELECT 80122, 81504 UNION ALL
  SELECT 80124, 81505 UNION ALL
  SELECT 80125, 81506 UNION ALL
  SELECT 80126, 81507 UNION ALL
  SELECT 80127, 81508 UNION ALL
  SELECT 80129, 81509 UNION ALL
  SELECT 80130, 81510 UNION ALL
  SELECT 80131, 81511 UNION ALL
  SELECT 80128, 81512 UNION ALL
  SELECT 80133, 81513 UNION ALL
  SELECT 80134, 81514 UNION ALL
  SELECT 80135, 81515 UNION ALL
  SELECT 80132, 81516 UNION ALL
  SELECT 80136, 81517 UNION ALL
  SELECT 80137, 81518 UNION ALL
  SELECT 80138, 81519 UNION ALL
  SELECT 80139, 81520 UNION ALL
  SELECT 80140, 81521 UNION ALL
  SELECT 80141, 81522 UNION ALL
  SELECT 80142, 81523 UNION ALL
  SELECT 80143, 81524 UNION ALL
  SELECT 80144, 81525 UNION ALL
  SELECT 80145, 81526 UNION ALL
  SELECT 80146, 81527 UNION ALL
  SELECT 80147, 81528 UNION ALL
  SELECT 80148, 81529 UNION ALL
  SELECT 80149, 81530 UNION ALL
  SELECT 80151, 81531 UNION ALL
  SELECT 80150, 81532 UNION ALL
  SELECT 80153, 81533 UNION ALL
  SELECT 80152, 81534 UNION ALL
  SELECT 80155, 81535 UNION ALL
  SELECT 80154, 81536 UNION ALL
  SELECT 80156, 81537 UNION ALL
  SELECT 80157, 81538 UNION ALL
  SELECT 80158, 81539 UNION ALL
  SELECT 80159, 81540 UNION ALL
  SELECT 80160, 81541 UNION ALL
  SELECT 80161, 81542 UNION ALL
  SELECT 80163, 81543 UNION ALL
  SELECT 80162, 81544 UNION ALL
  SELECT 80164, 81545 UNION ALL
  SELECT 80165, 81546 UNION ALL
  SELECT 80166, 81547 UNION ALL
  SELECT 80167, 81548 UNION ALL
  SELECT 80168, 81549 UNION ALL
  SELECT 80169, 81550 UNION ALL
  SELECT 80171, 81551 UNION ALL
  SELECT 80170, 81552
) v ON m.`api_match_id` = v.`new_id`;

UPDATE `matches` m
  INNER JOIN `_migration_map_down` map ON m.`id` = map.`internal_id`
SET m.`api_match_id` = NULL;

UPDATE `matches` m
  INNER JOIN `_migration_map_down` map ON m.`id` = map.`internal_id`
SET m.`api_match_id` = map.`old_api_id`;

DROP TEMPORARY TABLE `_migration_map_down`;

-- Note: Goals deleted in Step 5 cannot be restored to their old (incomplete) state.
-- Run fetch-results after rolling back to re-sync from whichever API is active.
