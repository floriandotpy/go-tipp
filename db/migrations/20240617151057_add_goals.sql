-- migrate:up
CREATE TABLE IF NOT EXISTS goals (
    id INT AUTO_INCREMENT PRIMARY KEY,
    match_id INT NOT NULL,
    score_team_a INT,
    score_team_b INT,
    match_minute INT,
    goal_getter_id INT,
    goal_getter_name VARCHAR(255),
    is_penalty BOOLEAN,
    is_own_goal BOOLEAN,
    is_overtime BOOLEAN,
    comment TEXT,
    FOREIGN KEY (match_id) REFERENCES matches(id),
    UNIQUE KEY unique_match_score (match_id, score_team_a, score_team_b)
);

-- migrate:down
DROP TABLE IF EXISTS goals;
