-- 1. 先为 role_config_relation 添加唯一约束，避免 ON CONFLICT 报错
ALTER TABLE role_config_relation ADD CONSTRAINT IF NOT EXISTS role_config_relation_unique UNIQUE (role_id, project_id);

-- 2. 备份现有的关联关系到 role_config_relation 表
INSERT INTO role_config_relation (role_id, project_id, created_at)
SELECT id, project_id, NOW()
FROM role_config
WHERE project_id IS NOT NULL
ON CONFLICT (role_id, project_id) DO NOTHING;

-- 3. 删除 project_id 字段
ALTER TABLE role_config DROP COLUMN IF EXISTS project_id; 