-- Demo orders for the board. Applied by the reset endpoint and at boot when
-- SEED_DEMO_ORDERS=1 and the orders table is empty. Not a goose migration so
-- it can be re-run.
INSERT INTO orders (id, order_no, status, customer_name, phone, fulfilment, zone_id, address, "window",
  subtotal_kobo, delivery_fee_kobo, total_kobo, reference, cart_hash, paid_at, ready_by, confirmed_by,
  attention_reason, cancel_reason, expires_at, created_at, updated_at)
VALUES
  ('01DEMO0000000000000000001','IBK-0225','paid','Kunle','08010000001','delivery','yaba','5 Herbert Macaulay Way, Yaba','lunch',
     750000,120000,870000,'IBK-0225-d1a1','demo1', now()-interval '95 min', now()-interval '55 min','mcp', NULL,NULL, now()-interval '70 min', now()-interval '100 min', now()-interval '55 min'),
  ('01DEMO0000000000000000002','IBK-0226','paid','Ngozi',NULL,'pickup',NULL,NULL,'lunch',
     430000,0,430000,'IBK-0226-e2b2','demo2', now()-interval '80 min', now()-interval '40 min','reconciler', NULL,NULL, now()-interval '55 min', now()-interval '85 min', now()-interval '40 min'),
  ('01DEMO0000000000000000003','IBK-0227','paid','Seun','08010000003','delivery','surulere','11 Ogunlana Drive, Surulere','lunch',
     700000,70000,770000,'IBK-0227-f3c3','demo3', now()-interval '60 min', now()-interval '20 min','mcp', NULL,NULL, now()-interval '35 min', now()-interval '65 min', now()-interval '20 min'),
  ('01DEMO0000000000000000004','IBK-0228','cancelled','Bola',NULL,'pickup',NULL,NULL,'lunch',
     350000,0,350000,'IBK-0228-a4d4','demo4', NULL,NULL,NULL, NULL,'customer changed mind', now()-interval '20 min', now()-interval '50 min', now()-interval '45 min'),
  ('01DEMO0000000000000000005','IBK-0229','paid','Amaka','08010000005','delivery','ikeja','3 Allen Avenue, Ikeja','lunch',
     1050000,180000,1230000,'IBK-0229-b5e5','demo5', now()-interval '30 min', now()+interval '10 min','mcp', NULL,NULL, now()-interval '5 min', now()-interval '35 min', now()-interval '30 min'),
  ('01DEMO0000000000000000006','IBK-0230','pending_payment','Tunde',NULL,'pickup',NULL,NULL,'lunch',
     400000,0,400000,'IBK-0230-c6f6','demo6', NULL,NULL,NULL, NULL,NULL, now()+interval '22 min', now()-interval '8 min', now()-interval '8 min');

INSERT INTO order_lines (order_id, item_id, name_snapshot, unit_price_kobo, qty, line_kobo) VALUES
  ('01DEMO0000000000000000001','jollof','Jollof rice & chicken',350000,2,700000),
  ('01DEMO0000000000000000001','moimoi','Moi moi',80000,1,80000),
  ('01DEMO0000000000000000002','fried','Fried rice & chicken',350000,1,350000),
  ('01DEMO0000000000000000002','moimoi','Moi moi',80000,1,80000),
  ('01DEMO0000000000000000003','jollof','Jollof rice & chicken',350000,2,700000),
  ('01DEMO0000000000000000004','jollof','Jollof rice & chicken',350000,1,350000),
  ('01DEMO0000000000000000005','fried','Fried rice & chicken',350000,3,1050000),
  ('01DEMO0000000000000000006','ofada','Ofada rice & ayamase',400000,1,400000);

INSERT INTO order_events (order_id, type, actor, created_at) VALUES
  ('01DEMO0000000000000000001','created','mcp', now()-interval '100 min'),
  ('01DEMO0000000000000000001','link_issued','mcp', now()-interval '100 min'),
  ('01DEMO0000000000000000001','verified','mcp', now()-interval '95 min'),
  ('01DEMO0000000000000000001','confirmed','mcp', now()-interval '95 min'),
  ('01DEMO0000000000000000002','created','mcp', now()-interval '85 min'),
  ('01DEMO0000000000000000002','link_issued','mcp', now()-interval '85 min'),
  ('01DEMO0000000000000000002','confirmed','reconciler', now()-interval '80 min'),
  ('01DEMO0000000000000000003','created','mcp', now()-interval '65 min'),
  ('01DEMO0000000000000000003','link_issued','mcp', now()-interval '65 min'),
  ('01DEMO0000000000000000003','verified','mcp', now()-interval '60 min'),
  ('01DEMO0000000000000000003','confirmed','mcp', now()-interval '60 min'),
  ('01DEMO0000000000000000004','created','mcp', now()-interval '50 min'),
  ('01DEMO0000000000000000004','link_issued','mcp', now()-interval '50 min'),
  ('01DEMO0000000000000000004','cancelled','admin:adm_01', now()-interval '45 min'),
  ('01DEMO0000000000000000005','created','mcp', now()-interval '35 min'),
  ('01DEMO0000000000000000005','link_issued','mcp', now()-interval '35 min'),
  ('01DEMO0000000000000000005','verified','mcp', now()-interval '30 min'),
  ('01DEMO0000000000000000005','confirmed','mcp', now()-interval '30 min'),
  ('01DEMO0000000000000000006','created','mcp', now()-interval '8 min'),
  ('01DEMO0000000000000000006','link_issued','mcp', now()-interval '8 min');

-- stock after the paid demo orders: jollof 20-4=16, fried 15-4=11, moimoi 30-2=28
UPDATE menu_items SET portions = 16 WHERE id = 'jollof';
UPDATE menu_items SET portions = 11 WHERE id = 'fried';
UPDATE menu_items SET portions = 28 WHERE id = 'moimoi';
