-- chair_locationsの中身をlatest_chair_locationsにコピーする
INSERT INTO latest_chair_locations (chair_id, latitude, longitude, created_at)
SELECT chair_id, latitude, longitude, created_at
FROM chair_locations
WHERE (chair_id, created_at) IN (
  SELECT chair_id, MAX(created_at)
  FROM chair_locations
  GROUP BY chair_id
);