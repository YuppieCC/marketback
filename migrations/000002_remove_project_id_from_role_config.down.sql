-- 添加 project_id 字段
ALTER TABLE role_config ADD COLUMN IF NOT EXISTS project_id bigint;

-- 从 role_config_relation 表恢复数据
UPDATE role_config rc
SET project_id = rcr.project_id
FROM role_config_relation rcr
WHERE rc.id = rcr.role_id; 