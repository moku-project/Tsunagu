-- Older AnimeHttpSource concatenated baseUrl with a relative "series/episode"
-- identifier, treating the series as part of the host and persisting only /episode.
-- Repair this verified 123Anime format in place so progress/download foreign keys survive.
UPDATE chapters AS c
SET external_id = (
    SELECT substr(m.external_id, 8) || c.external_id FROM media m WHERE m.id = c.media_id
)
WHERE c.external_id GLOB '/[0-9]*'
  AND substr(c.external_id, 2) NOT GLOB '*[^0-9]*'
  AND EXISTS (
    SELECT 1 FROM media m JOIN extensions e ON e.id = m.extension_id
    WHERE m.id = c.media_id AND m.content_type = 'anime'
      AND e.package_name = 'eu.kanade.tachiyomi.animeextension.en.onetwothreeanime'
      AND m.external_id LIKE '/anime/%' AND length(m.external_id) > 7
      AND NOT EXISTS (
        SELECT 1 FROM chapters valid WHERE valid.media_id = c.media_id
          AND valid.external_id = substr(m.external_id, 8) || c.external_id
      )
  );
