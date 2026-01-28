-- 修复 project_fund_transfer_record 表的序列值
SELECT setval(
    pg_get_serial_sequence('project_fund_transfer_record', 'id'), 
    COALESCE((SELECT MAX(id) FROM project_fund_transfer_record), 0) + 1, 
    false
); 