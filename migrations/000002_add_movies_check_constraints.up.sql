ALTER TABLE "Movies"
ADD CONSTRAINT movies_runtime_check CHECK (runtime > 0);

ALTER TABLE "Movies"
ADD CONSTRAINT movies_year_check CHECK (
    year BETWEEN 1895 AND DATE_PART('year', NOW())
  );

ALTER TABLE "Movies"
ADD CONSTRAINT movies_genres_length_check CHECK (
    ARRAY_LENGTH(genres, 1) BETWEEN 1 AND 5
  );
