CREATE TABLE realms (
  id INTEGER PRIMARY KEY,
  slug VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  region CHAR(2) NOT NULL,
  UNIQUE (slug, region)
);

CREATE TABLE races (
  id INTEGER PRIMARY KEY,
  name VARCHAR(32) NOT NULL
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
  icon VARCHAR(128)
);

CREATE TABLE talents (
  id INTEGER PRIMARY KEY,
  spell_id INTEGER NOT NULL,
  class_id INTEGER NOT NULL REFERENCES classes (id),
  spec_id INTEGER NOT NULL DEFAULT 0,
  name VARCHAR(128) NOT NULL,
  icon VARCHAR(128)
);
CREATE INDEX ON talents (class_id, spec_id);

CREATE TABLE pvp_talents (
  id INTEGER PRIMARY KEY,
  spell_id INTEGER NOT NULL,
  spec_id INTEGER NOT NULL REFERENCES specs (id),
  name VARCHAR(128) NOT NULL,
  icon VARCHAR(128)
);
CREATE INDEX ON pvp_talents (spec_id);

CREATE TABLE achievements (
  id INTEGER PRIMARY KEY,
  name VARCHAR(128),
  description VARCHAR(1024)
);

CREATE TABLE players (
  id SERIAL PRIMARY KEY,
  name VARCHAR(32) NOT NULL,
  realm_id INTEGER NOT NULL REFERENCES realms (id),
  blizzard_id INTEGER NOT NULL,
  class_id INTEGER REFERENCES classes (id),
  spec_id INTEGER REFERENCES specs (id),
  faction_id INTEGER REFERENCES factions (id),
  race_id INTEGER REFERENCES races (id),
  gender SMALLINT,
  guild VARCHAR(64),
  last_update TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE (realm_id, blizzard_id)
);
CREATE INDEX ON players (class_id, spec_id);
CREATE INDEX ON players (faction_id, race_id);
CREATE INDEX ON players (guild);

CREATE TABLE leaderboards (
  region CHAR(2) NOT NULL,
  bracket CHAR(3) NOT NULL,
  player_id INTEGER NOT NULL REFERENCES players (id),
  ranking SMALLINT NOT NULL,
  rating SMALLINT NOT NULL,
  season_wins SMALLINT,
  season_losses SMALLINT,
  last_update TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (region, bracket, player_id)
);
CREATE INDEX ON leaderboards (ranking);
CREATE INDEX ON leaderboards (rating);

CREATE TABLE players_achievements (
  player_id INTEGER NOT NULL REFERENCES players (id),
  achievement_id INTEGER NOT NULL REFERENCES achievements (id),
  PRIMARY KEY (player_id, achievement_id)
);

CREATE TABLE players_talents (
  player_id INTEGER NOT NULL REFERENCES players (id),
  talent_id INTEGER NOT NULL REFERENCES talents (id),
  PRIMARY KEY (player_id, talent_id)
);

CREATE TABLE players_pvp_talents (
  player_id INTEGER NOT NULL REFERENCES players (id),
  pvp_talent_id INTEGER NOT NULL REFERENCES pvp_talents (id),
  PRIMARY KEY (player_id, pvp_talent_id)
);

CREATE TABLE players_stats (
  player_id INTEGER PRIMARY KEY REFERENCES players (id),
  strength INTEGER,
  agility INTEGER,
  intellect INTEGER,
  stamina INTEGER,
  critical_strike INTEGER,
  haste INTEGER,
  versatility INTEGER,
  mastery INTEGER,
  leech INTEGER,
  dodge INTEGER,
  parry INTEGER
);

CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  name VARCHAR(128)
);

 CREATE TABLE players_items (
  player_id INTEGER PRIMARY KEY REFERENCES players (id),
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

-- Shadowlands schema changes
ALTER TABLE items ADD COLUMN quality VARCHAR(64);
ALTER TABLE items ADD COLUMN last_update TIMESTAMP DEFAULT NOW();

ALTER TABLE players ADD COLUMN last_login TIMESTAMP NOT NULL DEFAULT '2004-11-23';
ALTER TABLE players ADD COLUMN profile_id TEXT;

ALTER TABLE achievements ADD COLUMN icon VARCHAR(128);

-- Dragonflight schema changes
ALTER TABLE talents ADD COLUMN node_id INTEGER;
ALTER TABLE talents ADD COLUMN display_row INTEGER;
ALTER TABLE talents ADD COLUMN display_col INTEGER;
CREATE INDEX ON talents (node_id);
CREATE INDEX ON talents (display_col);

ALTER TABLE players_talents ADD COLUMN stale BOOLEAN DEFAULT TRUE;
ALTER TABLE players_pvp_talents ADD COLUMN stale BOOLEAN DEFAULT TRUE;
ALTER TABLE players_talents DROP CONSTRAINT players_talents_talent_id_fkey,
  ADD CONSTRAINT players_talents_talent_id_fkey
  FOREIGN KEY ("talent_id") REFERENCES talents(id) ON DELETE CASCADE;
ALTER TABLE players_pvp_talents DROP CONSTRAINT players_pvp_talents_pvp_talent_id_fkey,
  ADD CONSTRAINT players_pvp_talents_pvp_talent_id_fkey
  FOREIGN KEY ("pvp_talent_id") REFERENCES pvp_talents(id) ON DELETE CASCADE;
ALTER TABLE talents ADD COLUMN stale BOOLEAN DEFAULT TRUE;
ALTER TABLE pvp_talents ADD COLUMN stale BOOLEAN DEFAULT TRUE;

ALTER TABLE leaderboards ALTER COLUMN bracket TYPE VARCHAR(16);
CREATE INDEX ON leaderboards (bracket);

ALTER TABLE players ALTER COLUMN blizzard_id TYPE BIGINT;

CREATE INDEX ON players_talents (stale);
CREATE INDEX ON players_pvp_talents (stale);

CREATE INDEX ON items (last_update);
CREATE INDEX ON players (last_update);
CREATE INDEX ON leaderboards (player_id);
ALTER TABLE players_achievements DROP CONSTRAINT players_achievements_player_id_fkey,
  ADD CONSTRAINT players_achievements_player_id_fkey
  FOREIGN KEY ("player_id") REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE players_stats DROP CONSTRAINT players_stats_player_id_fkey,
  ADD CONSTRAINT players_stats_player_id_fkey
  FOREIGN KEY ("player_id") REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE players_items DROP CONSTRAINT players_items_player_id_fkey,
  ADD CONSTRAINT players_items_player_id_fkey
  FOREIGN KEY ("player_id") REFERENCES players(id) ON DELETE CASCADE;

-- create a stored proc to remove players (and associated data) for those
-- that are not currently on a leaderboard
CREATE OR REPLACE FUNCTION purge_old_players()
RETURNS VOID LANGUAGE plpgsql AS $proc$
BEGIN
  DELETE FROM players_pvp_talents WHERE stale=TRUE;
  DELETE FROM players_talents WHERE stale=TRUE;
  DELETE FROM players WHERE DATE_PART('day', NOW() - players.last_update) > 30 AND id NOT IN (SELECT player_id FROM leaderboards);
  DELETE FROM items WHERE DATE_PART('day', NOW() - items.last_update) > 30;
END; $proc$;
