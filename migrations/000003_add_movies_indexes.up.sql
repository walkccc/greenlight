CREATE INDEX IF NOT EXISTS movies_title_index ON "Movies" USING GIN (TO_TSVECTOR('simple', title));

CREATE INDEX IF NOT EXISTS movies_genres_index ON "Movies" USING GIN (genres);
