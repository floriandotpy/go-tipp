-- migrate:up
ALTER TABLE `matches`
ADD COLUMN `event_phase` int NOT NULL DEFAULT 1;

-- migrate:down
ALTER TABLE `matches`
DROP COLUMN `event_phase`;
