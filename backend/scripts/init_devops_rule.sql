-- KK Alert 初始化脚本
-- 创建 devops 渠道和 up == 1 告警规则

-- 1. 创建 devops 通知渠道 (如果不存在)
INSERT INTO channels (name, type, config, enabled, created_at, updated_at)
SELECT 'devops', 'telegram', '{"token":"YOUR_BOT_TOKEN","chat_id":"YOUR_CHAT_ID"}', true, datetime('now'), datetime('now')
WHERE NOT EXISTS (SELECT 1 FROM channels WHERE name = 'devops');

-- 查看创建的渠道ID
SELECT id, name, type, enabled FROM channels WHERE name = 'devops';

-- 2. 创建告警规则：监控 up == 1 (服务在线)
-- 注意：请根据实际的 channel_id 修改下方的 [CHANNEL_ID]
INSERT INTO rules (
    name,
    enabled,
    priority,
    datasource_ids,
    query_language,
    query_expression,
    match_labels,
    match_severity,
    channel_ids,
    template_id,
    check_interval,
    duration,
    send_interval,
    recovery_notify,
    aggregate_by,
    aggregate_window,
    exclude_windows,
    suppression,
    jira_enabled,
    jira_after_n,
    jira_config,
    created_at,
    updated_at
) VALUES (
    '服务在线监控',
    true,
    10,
    '[]',                          -- 匹配所有数据源，或改为特定数据源ID，如 '[1]'
    'promql',                      -- 查询语言：PromQL
    'up == 1',                     -- 查询表达式：监控服务在线状态
    '{}',                          -- 匹配标签（空表示匹配所有）
    '',                            -- 匹配严重程度（空表示匹配所有）
    '[1]',                         -- 渠道ID数组，这里假设devops渠道ID为1，请根据实际情况修改
    null,                          -- 模板ID（可选）
    '1m',                          -- 检测频率：1分钟
    '0',                           -- 持续时间：0表示立即触发
    '5m',                          -- 发送间隔：5分钟内同一告警只发一次
    true,                          -- 恢复通知：服务恢复时发送通知
    'instance',                    -- 按实例聚合
    '5m',                          -- 聚合窗口：5分钟
    '[]',                          -- 排除时段（空表示不排除）
    '{}',                          -- 抑制配置（空表示不抑制）
    false,                         -- 不启用Jira
    3,
    '',
    datetime('now'),
    datetime('now')
);

-- 查看创建的规则
SELECT r.id, r.name, r.enabled, r.query_expression, r.channel_ids 
FROM rules r 
WHERE r.name = '服务在线监控';

-- 3. 验证渠道和规则的关联
SELECT 
    r.id as rule_id,
    r.name as rule_name,
    r.query_expression,
    r.channel_ids,
    c.id as channel_id,
    c.name as channel_name,
    c.type as channel_type
FROM rules r
JOIN channels c ON c.id = json_extract(r.channel_ids, '$[0]')
WHERE r.name = '服务在线监控';
