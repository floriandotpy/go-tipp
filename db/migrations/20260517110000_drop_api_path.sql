-- migrate:up
ALTER TABLE `event_phases` DROP COLUMN `api_path`;

-- migrate:down
ALTER TABLE `event_phases` ADD COLUMN `api_path` varchar(512) NOT NULL DEFAULT '' AFTER `title`;
