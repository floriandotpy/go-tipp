-- migrate:up
ALTER TABLE `matches`
ADD COLUMN `result_aet_a` INT DEFAULT NULL,
ADD COLUMN `result_aet_b` INT DEFAULT NULL,
ADD COLUMN `result_apen_a` INT DEFAULT NULL,
ADD COLUMN `result_apen_b` INT DEFAULT NULL;

-- migrate:down
ALTER TABLE `matches`
DROP COLUMN `result_apen_b`,
DROP COLUMN `result_apen_a`,
DROP COLUMN `result_aet_b`,
DROP COLUMN `result_aet_a`;