CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS segments (
    slug TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users_segments (
    user_id TEXT REFERENCES users (id) NOT NULL,
    segment_slug TEXT REFERENCES segments (slug) NOT NULL,

    CONSTRAINT users_segments_pk PRIMARY KEY(user_id, segment_slug)
);

CREATE INDEX IF NOT EXISTS users_segments_user_id ON users_segments (user_id);
-- CREATE INDEX IF NOT EXISTS users_segments_segment_slug ON users_segments (segment_slug); -- May be useful for future /get-all-users-by-segment request

CREATE TABLE IF NOT EXISTS users_segments_history (
    user_id TEXT NOT NULL,
    segment_slug TEXT NOT NULL,
    action TEXT NOT NULL,
    log_time timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS users_segments_history_user_id ON users_segments_history (user_id);
CREATE INDEX IF NOT EXISTS users_segments_history_segment_slug ON users_segments_history (segment_slug);
CREATE INDEX IF NOT EXISTS users_segments_history_log_time ON users_segments_history(log_time);

CREATE TABLE IF NOT EXISTS users_segments_to_remove (
    user_id TEXT REFERENCES users (id) ON DELETE CASCADE NOT NULL,
    segment_slug TEXT REFERENCES segments (slug) ON DELETE CASCADE NOT NULL,
    remove_time timestamptz NOT NULL,

    CONSTRAINT users_segments_to_remove_pk  PRIMARY KEY(user_id, segment_slug)
)