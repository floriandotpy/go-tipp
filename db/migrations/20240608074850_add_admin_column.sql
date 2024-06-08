-- migrate:up
ALTER TABLE `users`
ADD COLUMN `admin` TINYINT(1) NOT NULL DEFAULT 0;

-- migrate:down
ALTER TABLE `users`
DROP COLUMN `admin`;
