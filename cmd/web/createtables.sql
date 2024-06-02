USE gotipp;

CREATE TABLE snippets (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created DATETIME NOT NULL,
    expires DATETIME NOT NULL
);
CREATE INDEX idx_snippets_created ON snippets(created);

CREATE TABLE matches (
    id INT AUTO_INCREMENT PRIMARY KEY,
    start DATETIME NOT NULL,
    team_a VARCHAR(100) NOT NULL,
    team_b VARCHAR(100) NOT NULL,
    result_a INT NULL,
    result_b INT NULL,
    match_type VARCHAR(255)
);

CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `hashed_password` char(60) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `users_uc_email` (`email`),
  UNIQUE KEY `users_uc_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE tipps (
    id INT AUTO_INCREMENT PRIMARY KEY,
    match_id INT NOT NULL,
    user_id INT NOT NULL,
    tipp_a INT NOT NULL,
    tipp_b INT NOT NULL,
    created DATETIME NOT NULL,
    changed DATETIME NOT NULL,
    FOREIGN KEY (match_id) REFERENCES matches(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE sessions (
    token CHAR(43) PRIMARY KEY, data BLOB NOT NULL,
    expiry TIMESTAMP(6) NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

USE gotipp;

CREATE TABLE invites (
	id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT, 
	code VARCHAR(255) NOT NULL,
	note VARCHAR(255) NOT NULL,
	group_id INTEGER NOT NULL,
	created DATETIME NOT NULL
);
