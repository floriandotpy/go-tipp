-- migrate:up
ALTER TABLE `matches`
ADD COLUMN `finished` BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE `matches`
SET `finished` = TRUE
WHERE `result_a` IS NOT NULL AND `result_b` IS NOT NULL;

-- migrate:down
ALTER TABLE `matches`
DROP COLUMN `finished`;
