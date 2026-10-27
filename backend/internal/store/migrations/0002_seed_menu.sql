-- +goose Up
INSERT INTO vendor_settings (vendor_name, pickup_address, rider_phone, order_prefix, prep_minutes,
  lunch_open, lunch_close, lunch_orders_close, evening_open, evening_close, evening_orders_close)
VALUES ('Iya Bisi''s Kitchen', '14 Adelabu Street, Surulere, Lagos', '0801 000 0000', 'IBK', 40,
  '07:00','16:00','15:45','16:00','23:59','23:30');

INSERT INTO delivery_zones (id, name, fee_kobo, sort) VALUES
  ('surulere','Surulere', 70000,1),
  ('yaba',    'Yaba',    120000,2),
  ('ikeja',   'Ikeja',   180000,3);

INSERT INTO menu_items (id, name, description, price_kobo, portions, "window", sort) VALUES
  ('jollof',  'Jollof rice & chicken', 'Party-style smoky jollof with a piece of chicken', 350000, 20, 'all',     1),
  ('fried',   'Fried rice & chicken',  'Fried rice with mixed veg and a piece of chicken', 350000, 15, 'all',     2),
  ('ofada',   'Ofada rice & ayamase',  'Local ofada rice with green pepper sauce',         400000,  2, 'lunch',   3),
  ('beans',   'Beans & plantain',      'Ewa riro with fried plantain',                    250000, 12, 'all',     4),
  ('moimoi',  'Moi moi',               'Steamed bean pudding',                             80000, 30, 'all',     5),
  ('egusi',   'Egusi & pounded yam',   'Egusi soup with assorted meat, evening only',     450000, 10, 'evening', 6),
  ('chapman', 'Chapman',               'Chilled, 50cl',                                   100000, 25, 'all',     7),
  ('water',   'Bottled water',         '75cl',                                             30000, 40, 'all',     8);

-- +goose Down
DELETE FROM menu_items; DELETE FROM delivery_zones; DELETE FROM vendor_settings;
