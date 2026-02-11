#!/bin/bash

# KK Alert - 创建 devops 渠道和 up==1 告警规则
# 使用方法: ./init_devops_rule.sh [API_BASE_URL] [ADMIN_PASSWORD]

API_URL="${1:-http://localhost:8080}"
ADMIN_PASS="${2:-admin123}"

echo "🚀 初始化 devops 渠道和监控规则..."
echo "   API: $API_URL"
echo ""

# 1. 登录获取 Token
echo "1️⃣  登录获取 Token..."
TOKEN=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
	-H "Content-Type: application/json" \
	-d "{\"username\":\"admin\",\"password\":\"$ADMIN_PASS\"}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
	echo "❌ 登录失败，请检查用户名密码"
	exit 1
fi

echo "✅ 登录成功"
echo ""

# 2. 创建 devops 渠道
echo "2️⃣  创建 devops 通知渠道..."
CHANNEL_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/channels" \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $TOKEN" \
	-d '{
    "name": "devops",
    "type": "telegram",
    "config": "{\"token\":\"YOUR_BOT_TOKEN\",\"chat_id\":\"YOUR_CHAT_ID\"}",
    "enabled": true
  }')

CHANNEL_ID=$(echo $CHANNEL_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -z "$CHANNEL_ID" ]; then
	# 检查是否已存在
	EXISTING=$(curl -s "$API_URL/api/v1/channels" \
		-H "Authorization: Bearer $TOKEN" |
		grep -o '"id":[0-9]*.*"name":"devops"' | grep -o '"id":[0-9]*' | cut -d':' -f2)

	if [ -n "$EXISTING" ]; then
		CHANNEL_ID=$EXISTING
		echo "✅ 使用已存在的渠道 (ID: $CHANNEL_ID)"
	else
		echo "⚠️  创建渠道失败: $CHANNEL_RESPONSE"
		exit 1
	fi
else
	echo "✅ 渠道创建成功 (ID: $CHANNEL_ID)"
fi

echo ""

# 3. 创建告警规则
echo "3️⃣  创建 up == 1 告警规则..."
RULE_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/rules" \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $TOKEN" \
	-d "{
    \"name\": \"服务在线监控\",
    \"enabled\": true,
    \"priority\": 10,
    \"datasource_ids\": \"[]\",
    \"query_language\": \"promql\",
    \"query_expression\": \"up == 1\",
    \"match_labels\": \"{}\",
    \"match_severity\": \"\",
    \"channel_ids\": \"[$CHANNEL_ID]\",
    \"template_id\": null,
    \"check_interval\": \"1m\",
    \"duration\": \"0\",
    \"send_interval\": \"5m\",
    \"recovery_notify\": true,
    \"aggregate_by\": \"instance\",
    \"aggregate_window\": \"5m\",
    \"exclude_windows\": \"[]\",
    \"suppression\": \"{}\",
    \"jira_enabled\": false
  }")

RULE_ID=$(echo $RULE_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$RULE_ID" ]; then
	echo "✅ 规则创建成功 (ID: $RULE_ID)"
	echo ""
	echo "✨ 初始化完成！"
	echo "   - 渠道: devops (ID: $CHANNEL_ID)"
	echo "   - 规则: 服务在线监控 (ID: $RULE_ID)"
	echo "   - 条件: up == 1"
	echo ""
	echo "⚠️  注意:"
	echo "   1. 请在 KK Alert 界面中配置 Telegram Bot Token 和 Chat ID"
	echo "   2. 确保已添加 Prometheus 数据源"
	echo "   3. 规则默认每分钟检测一次"
else
	echo "❌ 创建规则失败: $RULE_RESPONSE"
	exit 1
fi
