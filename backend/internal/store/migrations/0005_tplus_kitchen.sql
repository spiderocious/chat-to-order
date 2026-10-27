-- +goose Up
-- The demo vendor is TPlus Kitchen, a real food and small-chops vendor in
-- Osogbo that takes its orders on WhatsApp. Existing orders keep their rows:
-- new ids are inserted first, the references are moved, then the old ids go.
UPDATE vendor_settings SET vendor_name = 'TPlus Kitchen', pickup_address = 'TPlus Kitchen, Oke-Fia, Osogbo, Osun',
  order_prefix = 'TPK', version = version + 1, updated_at = now();

INSERT INTO delivery_zones (id, name, fee_kobo, sort) VALUES
  ('okefia',    'Oke-Fia',    50000, 1),
  ('ogooluwa',  'Ogo-Oluwa',  80000, 2),
  ('otaefun',   'Ota-Efun',  120000, 3);
UPDATE orders SET zone_id = 'okefia'   WHERE zone_id = 'surulere';
UPDATE orders SET zone_id = 'ogooluwa' WHERE zone_id = 'yaba';
UPDATE orders SET zone_id = 'otaefun'  WHERE zone_id = 'ikeja';
DELETE FROM delivery_zones WHERE id IN ('surulere','yaba','ikeja');

UPDATE menu_items SET name = 'Jollof rice & chicken', description = 'Party-style smoky jollof with a piece of chicken', price_kobo = 350000, "window" = 'all', sort = 1, version = version + 1, updated_at = now() WHERE id = 'jollof';
UPDATE menu_items SET name = 'Fried rice & chicken',  description = 'Fried rice with mixed veg and a piece of chicken', price_kobo = 350000, "window" = 'all', sort = 2, version = version + 1, updated_at = now() WHERE id = 'fried';
UPDATE menu_items SET name = 'Chapman',               description = 'Chilled, 50cl',                                   price_kobo = 100000, "window" = 'all', sort = 7, version = version + 1, updated_at = now() WHERE id = 'chapman';

INSERT INTO menu_items (id, name, description, price_kobo, portions, "window", sort) VALUES
  ('biryani',    'Chicken biryani',        'Fragrant spiced rice with chicken; only a few pans a day',       400000,  2, 'all',     3),
  ('smallchops', 'Small chops pack',       'Puff puff, samosa, spring rolls and peppered gizzard',           250000, 12, 'all',     4),
  ('puffpuff',   'Puff puff (10 pieces)',  'Soft and sugar-dusted',                                           80000, 30, 'all',     5),
  ('asun',       'Asun & plantain',        'Peppered goat meat with fried plantain, evening only',           450000, 10, 'evening', 6),
  ('cocktail',   'Fruit cocktail',         'Chilled mixed-fruit cocktail, 50cl',                             150000, 20, 'all',     8);
UPDATE order_lines SET item_id = 'biryani'    WHERE item_id = 'ofada';
UPDATE order_lines SET item_id = 'smallchops' WHERE item_id = 'beans';
UPDATE order_lines SET item_id = 'puffpuff'   WHERE item_id = 'moimoi';
UPDATE order_lines SET item_id = 'asun'       WHERE item_id = 'egusi';
UPDATE order_lines SET item_id = 'cocktail'   WHERE item_id = 'water';
DELETE FROM menu_items WHERE id IN ('ofada','beans','moimoi','egusi','water');

-- +goose Down
INSERT INTO delivery_zones (id, name, fee_kobo, sort) VALUES
  ('surulere','Surulere', 70000,1), ('yaba','Yaba',120000,2), ('ikeja','Ikeja',180000,3);
UPDATE orders SET zone_id = 'surulere' WHERE zone_id = 'okefia';
UPDATE orders SET zone_id = 'yaba'     WHERE zone_id = 'ogooluwa';
UPDATE orders SET zone_id = 'ikeja'    WHERE zone_id = 'otaefun';
DELETE FROM delivery_zones WHERE id IN ('okefia','ogooluwa','otaefun');
INSERT INTO menu_items (id, name, description, price_kobo, portions, "window", sort) VALUES
  ('ofada',  'Ofada rice & ayamase', 'Local ofada rice with green pepper sauce',     400000,  2, 'lunch',   3),
  ('beans',  'Beans & plantain',     'Ewa riro with fried plantain',                250000, 12, 'all',     4),
  ('moimoi', 'Moi moi',              'Steamed bean pudding',                         80000, 30, 'all',     5),
  ('egusi',  'Egusi & pounded yam',  'Egusi soup with assorted meat, evening only', 450000, 10, 'evening', 6),
  ('water',  'Bottled water',        '75cl',                                         30000, 40, 'all',     8);
UPDATE order_lines SET item_id = 'ofada'  WHERE item_id = 'biryani';
UPDATE order_lines SET item_id = 'beans'  WHERE item_id = 'smallchops';
UPDATE order_lines SET item_id = 'moimoi' WHERE item_id = 'puffpuff';
UPDATE order_lines SET item_id = 'egusi'  WHERE item_id = 'asun';
UPDATE order_lines SET item_id = 'water'  WHERE item_id = 'cocktail';
DELETE FROM menu_items WHERE id IN ('biryani','smallchops','puffpuff','asun','cocktail');
UPDATE vendor_settings SET vendor_name = 'Iya Bisi''s Kitchen', pickup_address = '14 Adelabu Street, Surulere, Lagos',
  order_prefix = 'IBK', version = version + 1, updated_at = now();
