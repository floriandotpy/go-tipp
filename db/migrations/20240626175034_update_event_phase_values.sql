-- migrate:up
LOCK TABLES `matches` WRITE;

-- Update event_phase to 2 for matches with id 13 to 24 (inclusive)
UPDATE `matches`
SET `event_phase` = 2
WHERE `id` BETWEEN 13 AND 24;

-- Update event_phase to 3 for matches with id 25 to 36
UPDATE `matches`
SET `event_phase` = 3
WHERE `id` BETWEEN 25 AND 36;

UNLOCK TABLES;

-- migrate:down
LOCK TABLES `matches` WRITE;

-- Revert event_phase to 1 for matches with id 13 to 24 (inclusive)
UPDATE `matches`
SET `event_phase` = 1
WHERE `id` BETWEEN 13 AND 24;

-- Revert event_phase to 1 for matches with id 25 to 36
UPDATE `matches`
SET `event_phase` = 1
WHERE `id` BETWEEN 25 AND 36;

UNLOCK TABLES;
