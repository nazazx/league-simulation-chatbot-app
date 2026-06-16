-- Schema for football league simulation

CREATE TABLE IF NOT EXISTS teams (
    id             SERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL UNIQUE,
    strength       INTEGER NOT NULL DEFAULT 50,
    attack_rating  INTEGER NOT NULL DEFAULT 50,
    defense_rating INTEGER NOT NULL DEFAULT 50,
    form_rating    INTEGER NOT NULL DEFAULT 50,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS matches (
    id            SERIAL PRIMARY KEY,
    week          INTEGER NOT NULL,
    home_team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    away_team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    home_score    INTEGER DEFAULT NULL,
    away_score    INTEGER DEFAULT NULL,
    played        BOOLEAN NOT NULL DEFAULT FALSE,
    played_at     TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT different_teams CHECK (home_team_id != away_team_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_week ON matches(week);
CREATE INDEX IF NOT EXISTS idx_matches_played ON matches(played);

CREATE TABLE IF NOT EXISTS historical_matches (
    id            SERIAL PRIMARY KEY,
    season        VARCHAR(20) NOT NULL,
    week          INTEGER NOT NULL,
    home_team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    away_team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    home_score    INTEGER NOT NULL,
    away_score    INTEGER NOT NULL,
    played_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT historical_different_teams CHECK (home_team_id != away_team_id),
    CONSTRAINT unique_historical_fixture UNIQUE (season, week, home_team_id, away_team_id)
);

CREATE INDEX IF NOT EXISTS idx_historical_matches_team_home ON historical_matches(home_team_id);
CREATE INDEX IF NOT EXISTS idx_historical_matches_team_away ON historical_matches(away_team_id);
CREATE INDEX IF NOT EXISTS idx_historical_matches_season ON historical_matches(season);
