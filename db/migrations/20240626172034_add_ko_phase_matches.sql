-- migrate:up
LOCK TABLES `matches` WRITE;
INSERT INTO `matches` (`id`, `start`, `team_a`, `team_b`, `result_a`, `result_b`, `match_type`, `event_phase`)
VALUES
    (37, '2024-06-29 18:00:00', 'Schweiz', 'Italien', NULL, NULL, 'Achtelfinale', 4),
    (38, '2024-06-29 21:00:00', 'Deutschland', 'Dänemark', NULL, NULL, 'Achtelfinale', 4),
    (39, '2024-06-30 18:00:00', 'England', '', NULL, NULL, 'Achtelfinale', 4),
    (40, '2024-06-30 21:00:00', 'Spanien', '', NULL, NULL, 'Achtelfinale', 4),
    (41, '2024-07-01 18:00:00', 'Frankreich', 'Belgien', NULL, NULL, 'Achtelfinale', 4),
    (42, '2024-07-01 21:00:00', 'Portugal', '', NULL, NULL, 'Achtelfinale', 4),
    (43, '2024-07-02 18:00:00', 'Rumänien', '', NULL, NULL, 'Achtelfinale', 4),
    (44, '2024-07-02 21:00:00', 'Österreich', '', NULL, NULL, 'Achtelfinale', 4),
    (45, '2024-07-05 18:00:00', '', '', NULL, NULL, 'Viertelfinale', 5),
    (46, '2024-07-05 21:00:00', '', '', NULL, NULL, 'Viertelfinale', 5),
    (47, '2024-07-06 18:00:00', '', '', NULL, NULL, 'Viertelfinale', 5),
    (48, '2024-07-06 21:00:00', '', '', NULL, NULL, 'Viertelfinale', 5),
    (49, '2024-07-09 21:00:00', '', '', NULL, NULL, 'Halbfinale', 6),
    (50, '2024-07-10 21:00:00', '', '', NULL, NULL, 'Halbfinale', 6),
    (51, '2024-07-14 21:00:00', '', '', NULL, NULL, 'Finale', 7);
UNLOCK TABLES;

-- migrate:down
LOCK TABLES `matches` WRITE;
    DELETE FROM `matches` 
    WHERE id IN 
    (37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51);
UNLOCK TABLES;
