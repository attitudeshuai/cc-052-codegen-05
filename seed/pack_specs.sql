-- Seed data for pack specs dictionary
INSERT INTO pack_spec (name, capacity_kg, material) VALUES
('5kg 纸箱', 5, '瓦楞纸箱'),
('10kg 纸箱', 10, '瓦楞纸箱'),
('15kg 泡沫箱', 15, '泡沫箱'),
('20kg 塑料筐', 20, '塑料筐')
ON CONFLICT DO NOTHING;
