import { useEffect, useState } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, Select, Switch, Card, Tag, Typography, Alert } from 'antd'
import { motion } from 'framer-motion'
import { 
  FileTextOutlined, 
  CopyOutlined, 
  EyeOutlined, 
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  InfoCircleOutlined,
  StarOutlined,
  StarFilled
} from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, EmptyState } from '../components/ui'

const { Text, Paragraph } = Typography

type Template = { id: number; name: string; channel_type: string; body: string; is_default?: boolean }

const BUILTIN_TEMPLATE: Template = {
  id: -1,
  name: '默认告警模板（示例）',
  channel_type: 'generic',
  body: `{{if .IsRecovery}}
━━━━━━━━━━━━━━━━━━━━━
✅ {{.Title}}
━━━━━━━━━━━━━━━━━━━━━
📊 数据源: {{.SourceType}}
📈 当前值/阈值: {{.Value}}
📍 标签:
{{range $key, $value := .Labels -}}
• {{$key}}: {{$value}}
{{end -}}
━━━━━━━━━━━━━━━━━━━━━
📌 告警ID: {{.AlertID}} 
⚠️ 严重程度: {{.Severity}} 
⏰ 发生时间: {{.StartAt}}{{if .ResolvedAt}} 
🕐 恢复时间: {{.ResolvedAt}}{{end}}
━━━━━━━━━━━━━━━━━━━━━
此告警由 KK Alert 系统自动发送
{{else}}
━━━━━━━━━━━━━━━━━━━━━
🔔 {{.Title}}
━━━━━━━━━━━━━━━━━━━━━
📊 数据源: {{.SourceType}} 
📈 当前值/阈值: {{.Value}}
📍 标签:
{{range $key, $value := .Labels -}}
• {{$key}}: {{$value}}
{{end -}}
━━━━━━━━━━━━━━━━━━━━━
📌 告警ID: {{.AlertID}} 
⚠️ 严重程度: {{.Severity}} 
⏰ 发生时间: {{.StartAt}}
━━━━━━━━━━━━━━━━━━━━━
此告警由 KK Alert 系统自动发送
{{end}}`
}

export default function Templates() {
  const { message, modal } = App.useApp()
  const [list, setList] = useState<Template[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState<boolean | { id: number }>(false)
  const [preview, setPreview] = useState<{ title: string; content: string } | null>(null)
  const [form] = Form.useForm()

  const load = () => {
    setLoading(true)
    fetch('/api/v1/templates', { headers: authHeaders() })
      .then((r) => r.json())
      .then((data) => {
        const templates = Array.isArray(data) ? data : []
        setList(templates)
      })
      .finally(() => setLoading(false))
  }
  
  useEffect(() => { load() }, [])

  const onFinish = async (v: any) => {
    const id = modalOpen && typeof modalOpen === 'object' && 'id' in modalOpen ? (modalOpen as any).id : null
    const url = id ? `/api/v1/templates/${id}` : '/api/v1/templates'
    const payload = { ...v, is_default: !!v.is_default }
    const res = await fetch(url, { method: id ? 'PUT' : 'POST', headers: authHeaders(), body: JSON.stringify(payload) })
    if (!res.ok) {
      message.error((await res.json()).error || '保存失败')
      return
    }
    message.success('保存成功')
    setModalOpen(false)
    form.resetFields()
    load()
  }

  const doPreview = (template: Template) => {
    const mockData = {
      title: 'CPU 使用率超过阈值',
      severity: 'warning',
      alert_id: 'ALERT-20240210-001',
      source_type: 'prometheus',
      start_at: '2026/2/11 09:46:54',
      description: '主机 cpu-usage 在过去5分钟内平均值超过 80%',
      rule_description: '规则说明示例（规则描述，可在模板中用 {{.RuleDescription}} 引用）',
      value: '80.5',
      labels: {
        instance: '192.168.1.100:9100',
        job: 'node-exporter',
        severity: 'warning',
        team: 'sre'
      }
    }
    const labelsBlock = Object.entries(mockData.labels)
      .map(([k, v]) => `• ${k}: ${v}`)
      .join('\n')
    const rangeBlockRe = /\{\{range \$\w+, \$\w+ := \.Labels -?\}\}[\s\S]*?\{\{end -?\}\}/g

    // Show firing branch (else) in preview: extract content between {{else}} and the closing {{end}}
    let content = template.body
    const elseIdx = content.indexOf('{{else}}')
    if (elseIdx >= 0) {
      const afterElse = content.slice(elseIdx + '{{else}}'.length)
      const lastEnd = afterElse.lastIndexOf('{{end}}')
      content = lastEnd >= 0 ? afterElse.slice(0, lastEnd) : afterElse
    }
    content = content
      .replace(rangeBlockRe, labelsBlock)
      .replace(/\{\{if \.RuleDescription\}\}\s*/g, '')
      .replace(/\{\{end\}\}/g, '')
      .replace(/{{\.Title}}/g, mockData.title)
      .replace(/{{\.Severity}}/g, mockData.severity)
      .replace(/{{\.AlertID}}/g, mockData.alert_id)
      .replace(/{{\.SourceType}}/g, mockData.source_type)
      .replace(/{{\.StartAt}}/g, mockData.start_at)
      .replace(/{{\.Description}}/g, mockData.description)
      .replace(/\{\{\.RuleDescription\}\}/g, mockData.rule_description)
      .replace(/\{\{\.Value\}\}/g, mockData.value)
      .replace(/{{\.Labels\.instance}}/g, mockData.labels.instance)
      .replace(/{{\.Labels\.job}}/g, mockData.labels.job)
      .replace(/{{\.Labels\.severity}}/g, mockData.labels.severity)
      .replace(/{{\.Labels\.team}}/g, mockData.labels.team)
    content = content.replace(/\{\{if \.ResolvedAt\}\}[\s\S]*?\{\{\.ResolvedAt\}\}[\s\S]*?/g, '').replace(/\{\{\.ResolvedAt\}\}/g, '')

    setPreview({ title: template.name, content })
  }

  const useBuiltinTemplate = () => {
    form.setFieldsValue({
      name: '默认告警模板',
      channel_type: 'generic',
      body: BUILTIN_TEMPLATE.body,
      is_default: false
    })
    setModalOpen(true)
  }

  const copyTemplate = (template: Template) => {
    form.setFieldsValue({
      name: `${template.name} - 副本`,
      channel_type: template.channel_type,
      body: template.body,
      is_default: false
    })
    setModalOpen(true)
    message.success('模板已复制到编辑器')
  }

  const setAsDefault = (id: number) => {
    fetch(`/api/v1/templates/${id}/set-default`, { method: 'PUT', headers: authHeaders() })
      .then((r) => {
        if (r.ok) {
          message.success('已设为默认模板')
          load()
        } else {
          r.json().then((d) => message.error(d?.error || '设置失败'))
        }
      })
      .catch(() => message.error('设置失败'))
  }

  const deleteTemplate = (id: number) => {
    modal.confirm({
      title: '确认删除',
      content: '删除后无法恢复，是否继续？',
      onOk: () => {
        fetch(`/api/v1/templates/${id}`, { method: 'DELETE', headers: authHeaders() })
          .then((x) => {
            if (x.ok) {
              message.success('删除成功')
              load()
            } else {
              message.error('删除失败')
            }
          })
      }
    })
  }

  const showBuiltinPreview = () => {
    doPreview(BUILTIN_TEMPLATE)
  }

  return (
    <div className="templates-page">
      <PageHeader
        title="通知模板"
        subtitle="管理告警通知的消息模板"
        actions={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setModalOpen(true); form.resetFields() }}
            size="large"
          >
            新建模板
          </Button>
        }
      />

      {list.length === 0 && !loading && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
        >
          <Alert
            message="开始使用通知模板"
            description="模板用于定义告警通知的消息格式。您可以使用内置示例快速开始，或创建自定义模板。"
            type="info"
            showIcon
            style={{ marginBottom: 24 }}
            action={
              <Button size="small" type="primary" onClick={useBuiltinTemplate}>
                使用示例模板
              </Button>
            }
          />
          
          <Card 
            title={
              <Space>
                <FileTextOutlined />
                <span>内置示例模板</span>
                <Tag color="blue">推荐</Tag>
              </Space>
            }
            extra={
              <Space>
                <Button icon={<EyeOutlined />} onClick={showBuiltinPreview}>
                  预览效果
                </Button>
                <Button type="primary" icon={<CopyOutlined />} onClick={useBuiltinTemplate}>
                  使用此模板
                </Button>
              </Space>
            }
            style={{ marginBottom: 24 }}
          >
            <Paragraph type="secondary">
              <InfoCircleOutlined style={{ marginRight: 8 }} />
              这是一个通用的告警通知模板，支持以下变量：
            </Paragraph>
            
            <div style={{ 
              background: '#f6f8fa', 
              padding: 16, 
              borderRadius: 8,
              fontFamily: 'monospace',
              fontSize: 13
            }}>
              <Text code>{'{{.Title}}'}</Text> - 告警标题{' '}
              <Text code>{'{{.AlertID}}'}</Text> - 告警ID{' '}
              <Text code>{'{{.Severity}}'}</Text> - 严重程度{' '}
              <Text code>{'{{.SourceType}}'}</Text> - 数据源类型{' '}
              <Text code>{'{{.Labels.xxx}}'}</Text> - 标签值
            </div>
          </Card>
        </motion.div>
      )}

      <Card variant="borderless" className="templates-table-card">
        <Table
          loading={loading}
          dataSource={list}
          rowKey="id"
          locale={{
            emptyText: (
              <EmptyState 
                type="create" 
                title="暂无模板" 
                description="点击下方按钮创建您的第一个通知模板"
                action={{ text: '创建模板', onClick: () => setModalOpen(true) }}
              />
            )
          }}
          columns={[
            { 
              title: 'ID', 
              dataIndex: 'id', 
              width: 70,
              render: (id) => <Tag>#{id}</Tag>
            },
            { 
              title: '模板名称', 
              dataIndex: 'name',
              render: (name, r) => (
                <Space>
                  <Text strong>{name}</Text>
                  {r.is_default && <Tag color="green">默认</Tag>}
                </Space>
              )
            },
            { 
              title: '渠道类型', 
              dataIndex: 'channel_type',
              width: 120,
              render: (type) => (
                <Tag color={type === 'generic' ? 'blue' : type === 'telegram' ? 'cyan' : 'green'}>
                  {type}
                </Tag>
              )
            },
            {
              title: '操作',
              width: 260,
              render: (_, r) => (
                <Space wrap>
                  <Button
                    type="text"
                    size="small"
                    icon={r.is_default ? <StarFilled /> : <StarOutlined />}
                    onClick={() => setAsDefault(r.id)}
                    disabled={!!r.is_default}
                    title={r.is_default ? '当前已是默认模板' : '设为默认模板'}
                  >
                    默认
                  </Button>
                  <Button 
                    type="text" 
                    size="small" 
                    icon={<EyeOutlined />}
                    onClick={() => doPreview(r)}
                  >
                    预览
                  </Button>
                  <Button 
                    type="text" 
                    size="small" 
                    icon={<CopyOutlined />}
                    onClick={() => copyTemplate(r)}
                  >
                    复制
                  </Button>
                  <Button 
                    type="text" 
                    size="small" 
                    icon={<EditOutlined />}
                    onClick={() => { setModalOpen({ id: r.id }); form.setFieldsValue(r) }}
                  >
                    编辑
                  </Button>
                  <Button 
                    type="text" 
                    size="small" 
                    danger 
                    icon={<DeleteOutlined />}
                    onClick={() => deleteTemplate(r.id)}
                  >
                    删除
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Modal 
        title={modalOpen && typeof modalOpen === 'object' ? '编辑模板' : '新建模板'}
        open={!!modalOpen} 
        onCancel={() => setModalOpen(false)} 
        footer={null} 
        width={700}
      >
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item 
            name="name" 
            label="模板名称" 
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="例如：企业微信告警模板" />
          </Form.Item>
          
          <Form.Item 
            name="channel_type" 
            label="渠道类型"
            rules={[{ required: true }]}
            initialValue="generic"
          >
            <Select 
              options={[
                { value: 'generic', label: '通用 (Generic)' },
                { value: 'telegram', label: 'Telegram' },
                { value: 'lark', label: '飞书 (Lark)' },
                { value: 'wecom', label: '企业微信' },
                { value: 'dingtalk', label: '钉钉' },
              ]} 
            />
          </Form.Item>

          <Form.Item name="is_default" label="设为默认模板" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="是" unCheckedChildren="否" />
          </Form.Item>
          <Paragraph type="secondary" style={{ marginTop: -8, marginBottom: 16, fontSize: 12 }}>
            默认模板会被未指定模板的规则使用；若规则所选的模板被删除，也会回退到默认模板。仅能有一个默认。
          </Paragraph>
          
          <Form.Item 
            name="body" 
            label="模板内容"
            rules={[{ required: true, message: '请输入模板内容' }]}
            extra={
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">可用变量：</Text>
                <Space size={4} wrap>
                  <Tag>{'{{.Title}}'}</Tag>
                  <Tag>{'{{.AlertID}}'}</Tag>
                  <Tag>{'{{.Severity}}'}</Tag>
                  <Tag>{'{{.StartAt}}'}</Tag>
                  <Tag>{'{{.SourceType}}'}</Tag>
                  <Tag>{'{{.Description}}'}</Tag>
                  <Tag>{'{{.ResolvedAt}}'}</Tag>
                  <Tag>{'{{.RuleDescription}}'}</Tag>
                  <Tag>{'{{range .Labels}}'}</Tag>
                </Space>
                <Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0, fontSize: 12 }}>
                  用 {'{{if .IsRecovery}}'} ... {'{{else}}'} ... {'{{end}}'} 可区分告警与恢复的展示样式（恢复时显示 ✅，告警时显示 🔔）。
                </Paragraph>
              </div>
            }
          >
            <Input.TextArea 
              rows={12} 
              placeholder="输入模板内容，使用 {{.Variable}} 语法插入变量"
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
          
          <Form.Item>
            <Button type="primary" htmlType="submit" size="large" block>
              保存模板
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      {preview && (
        <Modal 
          title={
            <Space>
              <EyeOutlined />
              <span>模板预览：{preview.title}</span>
            </Space>
          }
          open 
          onCancel={() => setPreview(null)} 
          footer={null}
          width={600}
        >
          <pre style={{ 
            whiteSpace: 'pre-wrap',
            background: '#f6f8fa',
            padding: 16,
            borderRadius: 8,
            fontSize: 13,
            lineHeight: 1.6,
            maxHeight: 400,
            overflow: 'auto'
          }}>
            {preview.content}
          </pre>
        </Modal>
      )}
    </div>
  )
}
