-- migrate:up
-- Add tv_channel and stream_link columns to matches table.
-- tv_channel: short identifier like "ARD", "ZDF", "MagentaTV"
-- stream_link: optional URL to a livestream page

ALTER TABLE `matches` ADD COLUMN `tv_channel` VARCHAR(20) DEFAULT NULL;
ALTER TABLE `matches` ADD COLUMN `stream_link` VARCHAR(512) DEFAULT NULL;

-- Seed group stage TV data from sportschau.de schedule (WM 2026).
-- Matches are identified by start time and team names.
-- Source: https://www.sportschau.de/fussball/fifa-wm-2026/der-spielplan-der-fussball-wm-2026,fifawm-spielplan-100.html

-- Spieltag 1 (11.06. - 17.06.)
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Mexiko' AND team_b = 'Südafrika' AND start LIKE '2026-06-11%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Südkorea' AND team_b = 'Tschechien' AND start LIKE '2026-06-12%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Kanada' AND team_b = 'Bosnien und Herzegowina' AND start LIKE '2026-06-12%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'USA' AND team_b = 'Paraguay' AND start LIKE '2026-06-13%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Katar' AND team_b = 'Schweiz' AND start LIKE '2026-06-13%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Brasilien' AND team_b = 'Marokko' AND start LIKE '2026-06-14%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Haiti' AND team_b = 'Schottland' AND start LIKE '2026-06-14%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Australien' AND team_b = 'Türkei' AND start LIKE '2026-06-14%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Deutschland' AND team_b = 'Curaçao' AND start LIKE '2026-06-14%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Niederlande' AND team_b = 'Japan' AND start LIKE '2026-06-14%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Elfenbeinküste' AND team_b = 'Ecuador' AND start LIKE '2026-06-15%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Schweden' AND team_b = 'Tunesien' AND start LIKE '2026-06-15%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Spanien' AND team_b = 'Kap Verde' AND start LIKE '2026-06-15%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Belgien' AND team_b = 'Ägypten' AND start LIKE '2026-06-15%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Saudi Arabien' AND team_b = 'Uruguay' AND start LIKE '2026-06-16%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Iran' AND team_b = 'Neuseeland' AND start LIKE '2026-06-16%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Frankreich' AND team_b = 'Senegal' AND start LIKE '2026-06-16%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Irak' AND team_b = 'Norwegen' AND start LIKE '2026-06-17%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Argentinien' AND team_b = 'Algerien' AND start LIKE '2026-06-17%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Österreich' AND team_b = 'Jordanien' AND start LIKE '2026-06-17%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Portugal' AND team_b = 'DR Kongo' AND start LIKE '2026-06-17%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'England' AND team_b = 'Kroatien' AND start LIKE '2026-06-17%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Ghana' AND team_b = 'Panama' AND start LIKE '2026-06-18%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Usbekistan' AND team_b = 'Kolumbien' AND start LIKE '2026-06-18%';

-- Spieltag 2 (18.06. - 24.06.)
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Tschechien' AND team_b = 'Südafrika' AND start LIKE '2026-06-18%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Schweiz' AND team_b = 'Bosnien und Herzegowina' AND start LIKE '2026-06-18%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Kanada' AND team_b = 'Katar' AND start LIKE '2026-06-19%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Mexiko' AND team_b = 'Südkorea' AND start LIKE '2026-06-19%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'USA' AND team_b = 'Australien' AND start LIKE '2026-06-19%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Schottland' AND team_b = 'Marokko' AND start LIKE '2026-06-20%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Brasilien' AND team_b = 'Haiti' AND start LIKE '2026-06-20%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Türkei' AND team_b = 'Paraguay' AND start LIKE '2026-06-20%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Niederlande' AND team_b = 'Schweden' AND start LIKE '2026-06-20%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Deutschland' AND team_b = 'Elfenbeinküste' AND start LIKE '2026-06-20%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Ecuador' AND team_b = 'Curaçao' AND start LIKE '2026-06-21%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Tunesien' AND team_b = 'Japan' AND start LIKE '2026-06-21%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Spanien' AND team_b = 'Saudi Arabien' AND start LIKE '2026-06-21%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Belgien' AND team_b = 'Iran' AND start LIKE '2026-06-21%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Uruguay' AND team_b = 'Kap Verde' AND start LIKE '2026-06-22%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Neuseeland' AND team_b = 'Ägypten' AND start LIKE '2026-06-22%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Argentinien' AND team_b = 'Österreich' AND start LIKE '2026-06-22%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Frankreich' AND team_b = 'Irak' AND start LIKE '2026-06-22%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Norwegen' AND team_b = 'Senegal' AND start LIKE '2026-06-23%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Jordanien' AND team_b = 'Algerien' AND start LIKE '2026-06-23%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Portugal' AND team_b = 'Usbekistan' AND start LIKE '2026-06-23%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'England' AND team_b = 'Ghana' AND start LIKE '2026-06-23%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Panama' AND team_b = 'Kroatien' AND start LIKE '2026-06-24%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Kolumbien' AND team_b = 'DR Kongo' AND start LIKE '2026-06-24%';

-- Spieltag 3 (24.06. - 28.06.)
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Schweiz' AND team_b = 'Kanada' AND start LIKE '2026-06-24%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Bosnien und Herzegowina' AND team_b = 'Katar' AND start LIKE '2026-06-24%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Schottland' AND team_b = 'Brasilien' AND start LIKE '2026-06-25%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Marokko' AND team_b = 'Haiti' AND start LIKE '2026-06-25%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Tschechien' AND team_b = 'Mexiko' AND start LIKE '2026-06-25%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Südafrika' AND team_b = 'Südkorea' AND start LIKE '2026-06-25%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Curaçao' AND team_b = 'Elfenbeinküste' AND start LIKE '2026-06-25%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Ecuador' AND team_b = 'Deutschland' AND start LIKE '2026-06-25%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Japan' AND team_b = 'Schweden' AND start LIKE '2026-06-26%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Tunesien' AND team_b = 'Niederlande' AND start LIKE '2026-06-26%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Türkei' AND team_b = 'USA' AND start LIKE '2026-06-26%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Paraguay' AND team_b = 'Australien' AND start LIKE '2026-06-26%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Norwegen' AND team_b = 'Frankreich' AND start LIKE '2026-06-26%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Senegal' AND team_b = 'Irak' AND start LIKE '2026-06-26%';
UPDATE matches SET tv_channel = 'ARD' WHERE team_a = 'Kap Verde' AND team_b = 'Saudi Arabien' AND start LIKE '2026-06-27%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Uruguay' AND team_b = 'Spanien' AND start LIKE '2026-06-27%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Ägypten' AND team_b = 'Iran' AND start LIKE '2026-06-27%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Neuseeland' AND team_b = 'Belgien' AND start LIKE '2026-06-27%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Panama' AND team_b = 'England' AND start LIKE '2026-06-27%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Kroatien' AND team_b = 'Ghana' AND start LIKE '2026-06-27%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Kolumbien' AND team_b = 'Portugal' AND start LIKE '2026-06-28%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'DR Kongo' AND team_b = 'Usbekistan' AND start LIKE '2026-06-28%';
UPDATE matches SET tv_channel = 'ZDF' WHERE team_a = 'Algerien' AND team_b = 'Österreich' AND start LIKE '2026-06-28%';
UPDATE matches SET tv_channel = 'MagentaTV' WHERE team_a = 'Jordanien' AND team_b = 'Argentinien' AND start LIKE '2026-06-28%';

-- Finale (ZDF confirmed)
UPDATE matches SET tv_channel = 'ZDF' WHERE match_type LIKE '%Finale%' AND start LIKE '2026-07-19%';


-- migrate:down
ALTER TABLE `matches` DROP COLUMN `tv_channel`;
ALTER TABLE `matches` DROP COLUMN `stream_link`;
