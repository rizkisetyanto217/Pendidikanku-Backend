INSERT INTO class_rooms (
  class_room_masjid_id,
  class_room_name,
  class_room_code,
  class_room_slug,
  class_room_location,
  class_room_capacity,
  class_room_description,
  class_room_is_virtual,
  class_room_is_active,
  class_room_features,
  class_room_virtual_links
)
VALUES (
  '2a1c3e4f-5b6a-7c8d-9e0f-1a2b3c4d5e60',
  'Virtual Room — Kelas 7',
  'V-7',
  'virtual-room-kelas-7',
  'Online',
  90,
  'Ruang virtual untuk Kelas 7 (A/B/C) bergantian.',
  TRUE,
  TRUE,
  '["virtual","recording","waiting-room"]'::jsonb,
  '[
     {"label":"Zoom Pagi","platform":"zoom","join_url":"https://zoom.us/j/111?pwd=pg","meeting_id":"111-111-111","passcode":"pg07","is_active":true,"time_window":{"from":"07:00","to":"09:00"},"tags":["7A","pagi"]},
     {"label":"Zoom Siang","platform":"zoom","join_url":"https://zoom.us/j/222?pwd=sg","meeting_id":"222-222-222","passcode":"sg13","is_active":true,"time_window":{"from":"13:00","to":"15:00"},"tags":["7B","siang"]},
     {"label":"Zoom Sore","platform":"zoom","join_url":"https://zoom.us/j/333?pwd=sr","meeting_id":"333-333-333","passcode":"sr16","is_active":true,"time_window":{"from":"16:00","to":"18:00"},"tags":["7C","sore"]}
   ]'::jsonb
);
