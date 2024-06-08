-- migrate:up
LOCK TABLES `groups` WRITE;
INSERT INTO `groups` (`id`, `name`, `invite`)
VALUES
	(1,'Tropic','runde-ins-eckige-2024'),
	(2,'Fanø Tippliga','dk-tippspiel-2024');
UNLOCK TABLES;

LOCK TABLES `matches` WRITE;
INSERT INTO `matches` (`id`, `start`, `team_a`, `team_b`, `result_a`, `result_b`, `match_type`)
VALUES
	(1,'2024-06-14 21:00:00','Deutschland','Schottland',NULL,NULL,'Gruppe A'),
	(2,'2024-06-15 15:00:00','Ungarn','Schweiz',NULL,NULL,'Gruppe A'),
	(3,'2024-06-15 18:00:00','Spanien','Kroatien',NULL,NULL,'Gruppe B'),
	(4,'2024-06-15 21:00:00','Italien','Albanien',NULL,NULL,'Gruppe B'),
	(5,'2024-06-16 15:00:00','Polen','Niederlande',NULL,NULL,'Gruppe D'),
	(6,'2024-06-16 18:00:00','Slowenien','Dänemark',NULL,NULL,'Gruppe C'),
	(7,'2024-06-16 21:00:00','Serbien','England',NULL,NULL,'Gruppe C'),
	(8,'2024-06-17 15:00:00','Rumänien','Ukraine',NULL,NULL,'Gruppe E'),
	(9,'2024-06-17 18:00:00','Belgien','Slowakei',NULL,NULL,'Gruppe E'),
	(10,'2024-06-17 21:00:00','Österreich','Frankreich',NULL,NULL,'Gruppe D'),
	(11,'2024-06-18 18:00:00','Türkei','Georgien',NULL,NULL,'Gruppe F'),
	(12,'2024-06-18 21:00:00','Portugal','Tschechien',NULL,NULL,'Gruppe F'),
	(13,'2024-06-19 15:00:00','Kroatien','Albanien',NULL,NULL,'Gruppe B'),
	(14,'2024-06-19 18:00:00','Deutschland','Ungarn',NULL,NULL,'Gruppe A'),
	(15,'2024-06-19 21:00:00','Schottland','Schweiz',NULL,NULL,'Gruppe A'),
	(16,'2024-06-20 15:00:00','Slowenien','Serbien',NULL,NULL,'Gruppe C'),
	(17,'2024-06-20 18:00:00','Dänemark','England',NULL,NULL,'Gruppe C'),
	(18,'2024-06-20 21:00:00','Spanien','Italien',NULL,NULL,'Gruppe B'),
	(19,'2024-06-21 15:00:00','Slowakei','Ukraine',NULL,NULL,'Gruppe E'),
	(20,'2024-06-21 18:00:00','Polen','Österreich',NULL,NULL,'Gruppe D'),
	(21,'2024-06-21 21:00:00','Niederlande','Frankreich',NULL,NULL,'Gruppe B'),
	(22,'2024-06-22 15:00:00','Georgien','Tschechien',NULL,NULL,'Gruppe F'),
	(23,'2024-06-22 18:00:00','Türkei','Portugal',NULL,NULL,'Gruppe F'),
	(24,'2024-06-22 21:00:00','Belgien','Rumänien',NULL,NULL,'Gruppe E'),
	(25,'2024-06-23 21:00:00','Schweiz','Deutschland',NULL,NULL,'Gruppe A'),
	(26,'2024-06-23 21:00:00','Schottland','Ungarn',NULL,NULL,'Gruppe A'),
	(27,'2024-06-24 21:00:00','Kroatien','Italien',NULL,NULL,'Gruppe B'),
	(28,'2024-06-24 21:00:00','Albanien','Spanien',NULL,NULL,'Gruppe B'),
	(29,'2024-06-25 18:00:00','Niederlande','Österreich',NULL,NULL,'Gruppe D'),
	(30,'2024-06-25 18:00:00','Frankreich','Polen',NULL,NULL,'Gruppe D'),
	(31,'2024-06-25 21:00:00','England','Slowenien',NULL,NULL,'Gruppe C'),
	(32,'2024-06-25 21:00:00','Dänemark','Serbien',NULL,NULL,'Gruppe C'),
	(33,'2024-06-26 18:00:00','Slowakei','Rumänien',NULL,NULL,'Gruppe E'),
	(34,'2024-06-26 18:00:00','Ukraine','Belgien',NULL,NULL,'Gruppe E'),
	(35,'2024-06-26 21:00:00','Tschechien','Türkei',NULL,NULL,'Gruppe F'),
	(36,'2024-06-26 21:00:00','Georgien','Portugal',NULL,NULL,'Gruppe F');
UNLOCK TABLES;


-- migrate:down
LOCK TABLES `groups` WRITE;
DELETE FROM `groups`
WHERE `id` IN (1, 2);
UNLOCK TABLES;

LOCK TABLES `matches` WRITE;
DELETE FROM `matches`
WHERE `id` IN (
    1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 
    11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 
    21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 
    31, 32, 33, 34, 35, 36
);
UNLOCK TABLES;
