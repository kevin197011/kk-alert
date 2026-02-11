#!/usr/bin/env ruby
# frozen_string_literal: true

# KK Alert 初始化脚本：创建 devops 渠道和 up==1 监控规则

require 'json'
require 'net/http'
require 'uri'

class InitDevopsRule
  def initialize(base_url = 'http://localhost:8080')
    @base_url = base_url
    @token = ENV['KK_ALERT_TOKEN'] || get_token
  end

  def get_token
    uri = URI("#{@base_url}/api/v1/auth/login")
    req = Net::HTTP::Post.new(uri)
    req['Content-Type'] = 'application/json'
    req.body = { username: 'admin', password: 'admin123' }.to_json

    res = Net::HTTP.start(uri.hostname, uri.port) { |http| http.request(req) }
    data = JSON.parse(res.body)
    data['token']
  end

  def headers
    {
      'Content-Type' => 'application/json',
      'Authorization' => "Bearer #{@token}"
    }
  end

  def create_channel
    uri = URI("#{@base_url}/api/v1/channels")
    req = Net::HTTP::Post.new(uri, headers)
    req.body = {
      name: 'devops',
      type: 'telegram',
      config: {
        token: ENV['TELEGRAM_BOT_TOKEN'] || 'your-bot-token',
        chat_id: ENV['TELEGRAM_CHAT_ID'] || 'your-chat-id'
      }.to_json,
      enabled: true
    }.to_json

    res = Net::HTTP.start(uri.hostname, uri.port) { |http| http.request(req) }
    data = JSON.parse(res.body)

    if res.is_a?(Net::HTTPSuccess)
      puts "✅ 渠道 'devops' 创建成功，ID: #{data['id']}"
      data['id']
    else
      puts "⚠️  创建渠道失败: #{data['error']}"
      nil
    end
  end

  def find_channel
    uri = URI("#{@base_url}/api/v1/channels")
    req = Net::HTTP::Get.new(uri, headers)

    res = Net::HTTP.start(uri.hostname, uri.port) { |http| http.request(req) }
    channels = JSON.parse(res.body)

    devops = channels.find { |c| c['name'] == 'devops' }
    return unless devops

    puts "✅ 找到已存在的 'devops' 渠道，ID: #{devops['id']}"
    devops['id']
  end

  def create_rule(channel_id)
    uri = URI("#{@base_url}/api/v1/rules")
    req = Net::HTTP::Post.new(uri, headers)
    req.body = {
      name: '服务在线监控',
      enabled: true,
      priority: 10,
      datasource_ids: '[]',
      query_language: 'promql',
      query_expression: 'up == 1',
      match_labels: '{}',
      match_severity: '',
      channel_ids: "[#{channel_id}]",
      template_id: nil,
      check_interval: '1m',
      duration: '0',
      send_interval: '5m',
      recovery_notify: true,
      aggregate_by: 'instance',
      aggregate_window: '5m',
      exclude_windows: '[]',
      suppression: '{}',
      jira_enabled: false
    }.to_json

    res = Net::HTTP.start(uri.hostname, uri.port) { |http| http.request(req) }
    data = JSON.parse(res.body)

    if res.is_a?(Net::HTTPSuccess)
      puts "✅ 规则 '服务在线监控' 创建成功，ID: #{data['id']}"
      puts '   查询条件: up == 1'
      puts "   发送渠道: devops (ID: #{channel_id})"
      puts '   检测频率: 1分钟'
      true
    else
      puts "❌ 创建规则失败: #{data['error']}"
      false
    end
  end

  def run
    puts '🚀 开始初始化 devops 渠道和监控规则...'
    puts "   API地址: #{@base_url}"
    puts

    channel_id = find_channel || create_channel

    unless channel_id
      puts '❌ 无法获取或创建渠道，退出'
      exit 1
    end

    puts

    if create_rule(channel_id)
      puts
      puts '✨ 初始化完成！'
      puts "   - 渠道: devops (ID: #{channel_id})"
      puts '   - 规则: 服务在线监控 (up == 1)'
      puts
      puts '⚠️  提示:'
      puts '   1. 请确保已配置 Telegram Bot Token 和 Chat ID'
      puts '   2. 可以通过环境变量设置: TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID'
      puts '   3. 登录 KK Alert 管理界面查看和修改配置'
    else
      puts '❌ 初始化失败'
      exit 1
    end
  end
end

if __FILE__ == $PROGRAM_NAME
  base_url = ARGV[0] || 'http://localhost:8080'
  InitDevopsRule.new(base_url).run
end
