-- Seed data: 4 default teams with synthetic prior ratings.
-- The attack/defense/form ratings provide a useful default roster for local demos.

INSERT INTO teams (name, strength, attack_rating, defense_rating, form_rating) VALUES
    ('Bosphorus United', 74, 78, 70, 76),
    ('Anka FC',          69, 72, 67, 71),
    ('Galata Rovers',    81, 84, 79, 80),
    ('Moda Athletic',    66, 68, 65, 69)
ON CONFLICT (name) DO UPDATE SET
    strength = EXCLUDED.strength,
    attack_rating = EXCLUDED.attack_rating,
    defense_rating = EXCLUDED.defense_rating,
    form_rating = EXCLUDED.form_rating,
    updated_at = NOW();

-- Synthetic historical sample data: two compact seasons used as prediction priors.
INSERT INTO historical_matches (season, week, home_team_id, away_team_id, home_score, away_score)
SELECT
    seed.season,
    seed.week,
    home_team.id,
    away_team.id,
    seed.home_score,
    seed.away_score
FROM (
    VALUES
        ('2022-2023', 1,  'Bosphorus United', 'Moda Athletic',    3, 1),
        ('2022-2023', 1,  'Anka FC',          'Galata Rovers',    2, 2),
        ('2022-2023', 2,  'Bosphorus United', 'Galata Rovers',    2, 1),
        ('2022-2023', 2,  'Moda Athletic',    'Anka FC',          1, 2),
        ('2022-2023', 3,  'Bosphorus United', 'Anka FC',          2, 2),
        ('2022-2023', 3,  'Galata Rovers',    'Moda Athletic',    3, 1),
        ('2022-2023', 4,  'Moda Athletic',    'Bosphorus United', 1, 3),
        ('2022-2023', 4,  'Galata Rovers',    'Anka FC',          1, 2),
        ('2022-2023', 5,  'Galata Rovers',    'Bosphorus United', 2, 2),
        ('2022-2023', 5,  'Anka FC',          'Moda Athletic',    3, 1),
        ('2022-2023', 6,  'Anka FC',          'Bosphorus United', 1, 2),
        ('2022-2023', 6,  'Moda Athletic',    'Galata Rovers',    1, 1),
        ('2023-2024', 1,  'Bosphorus United', 'Anka FC',          3, 2),
        ('2023-2024', 1,  'Galata Rovers',    'Moda Athletic',    2, 0),
        ('2023-2024', 2,  'Bosphorus United', 'Galata Rovers',    1, 1),
        ('2023-2024', 2,  'Anka FC',          'Moda Athletic',    2, 0),
        ('2023-2024', 3,  'Moda Athletic',    'Bosphorus United', 0, 2),
        ('2023-2024', 3,  'Galata Rovers',    'Anka FC',          2, 2),
        ('2023-2024', 4,  'Anka FC',          'Bosphorus United', 1, 1),
        ('2023-2024', 4,  'Moda Athletic',    'Galata Rovers',    1, 3),
        ('2023-2024', 5,  'Galata Rovers',    'Bosphorus United', 2, 3),
        ('2023-2024', 5,  'Moda Athletic',    'Anka FC',          1, 2),
        ('2023-2024', 6,  'Bosphorus United', 'Moda Athletic',    4, 1),
        ('2023-2024', 6,  'Anka FC',          'Galata Rovers',    2, 1)
) AS seed(season, week, home_name, away_name, home_score, away_score)
JOIN teams home_team ON home_team.name = seed.home_name
JOIN teams away_team ON away_team.name = seed.away_name
ON CONFLICT DO NOTHING;
