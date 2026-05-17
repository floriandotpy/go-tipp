-- migrate:up
ALTER TABLE `matches` ADD COLUMN `api_match_id` int DEFAULT NULL;
CREATE INDEX `idx_matches_api_match_id` ON `matches` (`api_match_id`);

-- migrate:down
DROP INDEX `idx_matches_api_match_id` ON `matches`;
ALTER TABLE `matches` DROP COLUMN `api_match_id`;
