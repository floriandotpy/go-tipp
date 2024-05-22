USE gotipp;

-- Create a `snippets` table.
CREATE TABLE snippets (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created DATETIME NOT NULL,
    expires DATETIME NOT NULL
);


CREATE TABLE matches (
    id INT AUTO_INCREMENT PRIMARY KEY,
    start DATETIME NOT NULL,
    team_a VARCHAR(100) NOT NULL,
    team_b VARCHAR(100) NOT NULL,
    result_a INT NULL,
    result_b INT NULL,
    match_type VARCHAR(255)
);

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255),
    email VARCHAR(255) NOT NULL,
    hashed_password CHAR(60) NOT NULL,
    created DATETIME NOT NULL
);

CREATE TABLE tipps (
    id INT AUTO_INCREMENT PRIMARY KEY,
    match_id INT NOT NULL,
    user_id INT NOT NULL,
    tipp_a INT NOT NULL,
    tipp_b INT NOT NULL,
    created DATETIME NOT NULL,
    FOREIGN KEY (match_id) REFERENCES matches(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);


-- Add an index on the created column.
CREATE INDEX idx_snippets_created ON snippets(created);