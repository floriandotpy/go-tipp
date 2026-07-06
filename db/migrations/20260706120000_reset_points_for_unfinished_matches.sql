-- migrate:up
-- One-time cleanup: zero out points for tipps belonging to matches that are not
-- finished. Earlier versions of UpdatePoints scored tipps against live/intermediate
-- results (matches with a result set but finished = 0), leaving stale non-zero points
-- in the tipps table. The scoring code now only writes points for finished matches,
-- but it no longer resets stale points, so we clear them here explicitly.
UPDATE `tipps` t
  INNER JOIN `matches` m ON t.`match_id` = m.`id`
SET t.`points` = 0,
    t.`result_correct` = 0,
    t.`tendency_correct` = 0,
    t.`goal_difference_correct` = 0
WHERE m.`finished` = 0;

-- migrate:down
-- No-op: stale intermediate points cannot be meaningfully restored. Correct values
-- for finished matches are recomputed by the fetch-results job (UpdatePoints).
