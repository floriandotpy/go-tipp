-- migrate:up
-- Enforce "one guess per user per match" at the database level. Until now this
-- invariant was only maintained by application-level check-then-write logic,
-- which is not safe under concurrent submissions (e.g. double-clicks or
-- simultaneous web + API writes) and could produce duplicate tipp rows that get
-- double-counted in the leaderboards.
ALTER TABLE `tipps`
  ADD UNIQUE KEY `tipps_uc_user_match` (`user_id`, `match_id`);

-- migrate:down
ALTER TABLE `tipps`
  DROP INDEX `tipps_uc_user_match`;
