CREATE TABLE realms (
  id INTEGER PRIMARY KEY,
  slug VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  region CHAR(2) NOT NULL,
  UNIQUE (slug, region)
);

CREATE TABLE races (
  id INTEGER PRIMARY KEY,
  name VARCHAR(32) NOT NULL,
  side VARCHAR(32) NOT NULL,
  UNIQUE (name, side)
);

CREATE TABLE factions (
  id INTEGER PRIMARY KEY,
  name VARCHAR(32) NOT NULL,
  UNIQUE (name)
);
-- there are only two playable factions and no endpoint
-- to either retrieve only those or filter so just insert them
INSERT INTO factions(id, name) VALUES (67, 'Horde'), (469, 'Alliance');

CREATE TABLE classes (
  id INTEGER PRIMARY KEY,
  name VARCHAR(32) NOT NULL,
  UNIQUE (name)
);

CREATE TABLE specs (
  id INTEGER PRIMARY KEY,
  class_id INTEGER NOT NULL REFERENCES classes (id),
  name VARCHAR(32) NOT NULL,
  role VARCHAR(32),
  description VARCHAR(1024),
  background_image VARCHAR(128),
  icon VARCHAR(128),
  UNIQUE (class_id, name)
);

CREATE TABLE talents (
  id INTEGER PRIMARY KEY,
  spell_id INTEGER NOT NULL,
  class_id INTEGER NOT NULL REFERENCES classes (id),
  spec_id INTEGER DEFAULT 0,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(1024),
  icon VARCHAR(128),
  tier SMALLINT,
  col SMALLINT
);

CREATE INDEX ON talents (tier, col);
CREATE INDEX ON talents (class_id, spec_id);

CREATE TABLE players (
  id SERIAL PRIMARY KEY,
  api_id INTEGER NOT NULL,
  name VARCHAR(32) NOT NULL,
  realm_id INTEGER NOT NULL REFERENCES realms (id),
  class_id INTEGER REFERENCES classes (id),
  spec_id INTEGER REFERENCES specs (id),
  faction_id INTEGER REFERENCES factions (id),
  race_id INTEGER REFERENCES races (id),
  guild VARCHAR(64),
  gender SMALLINT,
  last_update TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (name, realm_id),
  UNIQUE (api_id, realm_id)
);

CREATE INDEX ON players (class_id, spec_id);
CREATE INDEX ON players (faction_id, race_id);
CREATE INDEX ON players (guild);

CREATE TABLE leaderboards (
  bracket CHAR(3) NOT NULL,
  region CHAR(2) NOT NULL,
  ranking INTEGER NOT NULL,
  player_id INTEGER NOT NULL REFERENCES players (id),
  rating SMALLINT NOT NULL,
  season_wins SMALLINT,
  season_losses SMALLINT,
  last_update TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (bracket, region, player_id)
);

CREATE INDEX ON leaderboards (ranking);
CREATE INDEX ON leaderboards (rating);

CREATE TABLE achievements (
  id INTEGER PRIMARY KEY,
  name VARCHAR(128),
  description VARCHAR(1024),
  points SMALLINT
);

CREATE TABLE players_achievements (
  player_id INTEGER NOT NULL REFERENCES players (id),
  achievement_id INTEGER NOT NULL REFERENCES achievements (id),
  achieved_at TIMESTAMP,
  PRIMARY KEY (player_id, achievement_id)
);

CREATE TABLE players_talents (
  player_id INTEGER NOT NULL REFERENCES players (id),
  talent_id INTEGER NOT NULL REFERENCES talents (id),
  PRIMARY KEY (player_id, talent_id)
);

CREATE TABLE players_stats (
  player_id INTEGER PRIMARY KEY REFERENCES players (id),
  strength INTEGER,
  agility INTEGER,
  intellect INTEGER,
  stamina INTEGER,
  critical_strike INTEGER,
  haste INTEGER,
  mastery INTEGER,
  versatility INTEGER,
  leech REAL,
  dodge REAL,
  parry REAL
);

CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(128)
);

 CREATE TABLE players_items (
  player_id INTEGER PRIMARY KEY REFERENCES players (id),
  average_item_level INTEGER,
  average_item_level_equipped INTEGER,
  head INTEGER,
  neck INTEGER,
  shoulder INTEGER,
  back INTEGER,
  chest INTEGER,
  shirt INTEGER,
  tabard INTEGER,
  wrist INTEGER,
  hands INTEGER,
  waist INTEGER,
  legs INTEGER,
  feet INTEGER,
  finger1 INTEGER,
  finger2 INTEGER,
  trinket1 INTEGER,
  trinket2 INTEGER,
  mainhand INTEGER,
  offhand INTEGER
);

CREATE TABLE metadata (
  key VARCHAR(32) PRIMARY KEY,
  value VARCHAR(512) NOT NULL DEFAULT '',
  last_update TIMESTAMP DEFAULT NOW()
);

-- create a stored proc to remove players (and associated data) for those
-- that are not currently on a leaderboard
CREATE OR REPLACE FUNCTION purge_old_players()
RETURNS VOID LANGUAGE plpgsql AS $proc$
BEGIN
  DELETE FROM players_achievements WHERE player_id NOT IN (SELECT player_id FROM leaderboards);
  DELETE FROM players_talents WHERE player_id NOT IN (SELECT player_id FROM leaderboards);
  DELETE FROM players_stats WHERE player_id NOT IN (SELECT player_id FROM leaderboards);
  DELETE FROM players_items WHERE player_id NOT IN (SELECT player_id FROM leaderboards);
  DELETE FROM players WHERE id NOT IN (SELECT player_id FROM leaderboards);
END; $proc$;
