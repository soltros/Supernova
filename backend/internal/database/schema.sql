CREATE TABLE IF NOT EXISTS artists (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    musicbrainz_id TEXT,
    image_url TEXT,
    bio TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS albums (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    release_year INTEGER,
    musicbrainz_id TEXT,
    cover_art_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tracks (
    id TEXT PRIMARY KEY,
    album_id TEXT NOT NULL,
    title TEXT NOT NULL,
    track_number INTEGER,
    disc_number INTEGER,
    duration_ms INTEGER,
    format TEXT,
    bitrate INTEGER,
    file_path TEXT UNIQUE NOT NULL,
    FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS hearts (
    id TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id)
);

CREATE TABLE IF NOT EXISTS track_artists (
    track_id TEXT NOT NULL,
    artist_id TEXT NOT NULL,
    PRIMARY KEY (track_id, artist_id),
    FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE,
    FOREIGN KEY(artist_id) REFERENCES artists(id) ON DELETE CASCADE
);

-- Triggers to automatically purge orphaned hearts if a track or album is deleted from the filesystem
CREATE TRIGGER IF NOT EXISTS delete_track_heart
AFTER DELETE ON tracks
BEGIN
    DELETE FROM hearts WHERE entity_type = 'track' AND entity_id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS delete_album_heart
AFTER DELETE ON albums
BEGIN
    DELETE FROM hearts WHERE entity_type = 'album' AND entity_id = OLD.id;
END;

CREATE TABLE IF NOT EXISTS scrobbles (
    id TEXT PRIMARY KEY,
    track_id TEXT NOT NULL,
    listened_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS album_artists (
    album_id TEXT,
    artist_id TEXT,
    role TEXT DEFAULT 'primary',
    PRIMARY KEY (album_id, artist_id),
    FOREIGN KEY(album_id) REFERENCES albums(id) ON DELETE CASCADE,
    FOREIGN KEY(artist_id) REFERENCES artists(id) ON DELETE CASCADE
);

-- Indexes to massively speed up library scanning and API queries
CREATE INDEX IF NOT EXISTS idx_tracks_album_id ON tracks(album_id);
CREATE INDEX IF NOT EXISTS idx_tracks_file_path ON tracks(file_path);
CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name);
CREATE INDEX IF NOT EXISTS idx_albums_title ON albums(title);
