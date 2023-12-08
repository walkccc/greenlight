ALTER TABLE "Movies" DROP CONSTRAINT IF EXISTS movies_runtime_check;

ALTER TABLE "Movies" DROP CONSTRAINT IF EXISTS movies_year_check;

ALTER TABLE "Movies" DROP CONSTRAINT IF EXISTS movies_genres_length_check;
