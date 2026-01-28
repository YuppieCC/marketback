-- 修复 role_address 表的序列值
SELECT setval(
    pg_get_serial_sequence('role_address', 'id'), 
    COALESCE((SELECT MAX(id) FROM role_address), 0) + 1, 
    false
); 