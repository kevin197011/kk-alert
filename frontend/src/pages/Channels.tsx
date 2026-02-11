import { useEffect, useState } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, Select, Switch, Card, Tag, Row, Col } from 'antd'
import { motion } from 'framer-motion'
import { PlusOutlined, EditOutlined, DeleteOutlined, NotificationOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, StatusTag, EmptyState, StatCard } from '../components/ui'

type Channel = { id: number; name: string; type: string; enabled: boolean }

const TYPE_OPTIONS = [
  { value: 'telegram', label: 'Telegram' },
  { value: 'lark', label: '飞书 (Lark)' },
  { value: 'wecom', label: '企业微信' },
  { value: 'dingtalk', label: '钉钉' },
  { value: 'email', label: '邮件' },
  { value: 'webhook', label: 'Webhook' },
]

export default function Channels() {
  const { message, modal } = App.useApp()
  const [list, setList] = useState<Channel[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState<boolean | { id: number }>(false)
  const [form] = Form.useForm()

  const load = () => {
    setLoading(true)
    fetch('/api/v1/channels', { headers: authHeaders() })
      .then((r) => r.json())
      .then((data) => setList(Array.isArray(data) ? data : []))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const onFinish = async (v: any) => {
    const isEdit = modalOpen && typeof modalOpen === 'object' && 'id' in modalOpen
    const url = isEdit ? `/api/v1/channels/${(modalOpen as any).id}` : '/api/v1/channels'
    const method = isEdit ? 'PUT' : 'POST'
    const res = await fetch(url, { method, headers: authHeaders(), body: JSON.stringify(v) })
    if (!res.ok) {
      message.error((await res.json()).error || '保存失败')
      return
    }
    message.success('保存成功')
    setModalOpen(false)
    form.resetFields()
    load()
  }

  const deleteOne = (id: number) => {
    modal.confirm({
      title: '确认删除',
      content: '删除后无法恢复，是否继续？',
      onOk: () => {
        fetch(`/api/v1/channels/${id}`, { method: 'DELETE', headers: authHeaders() }).then((r) => {
          if (r.ok) {
            message.success('删除成功')
            load()
          } else {
            message.error('删除失败')
          }
        })
      }
    })
  }

  const testSend = async (id: number) => {
    const res = await fetch(`/api/v1/channels/${id}/test`, { method: 'POST', headers: authHeaders() })
    const data = await res.json().catch(() => ({}))
    if (res.ok) {
      message.success(data.message || '测试消息已发送')
    } else {
      message.error(data.error || '测试发送失败')
    }
  }

  const enabledCount = list.filter(c => c.enabled).length

  return (
    <div className="channels-page">
      <PageHeader
        title="通知渠道"
        subtitle="配置告警通知的接收渠道"
        actions={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setModalOpen(true); form.resetFields() }}
            size="large"
          >
            新建渠道
          </Button>
        }
      />

      <Row gutter={[24, 24]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="总渠道数" value={list.length} icon={<NotificationOutlined />} color="#1890ff" delay={0} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="已启用" value={enabledCount} icon={<CheckCircleOutlined />} color="#52c41a" delay={0.1} />
        </Col>
      </Row>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
      <Card variant="borderless" className="channels-table-card">
        <Table
          loading={loading}
          dataSource={list}
          rowKey="id"
          locale={{
            emptyText: <EmptyState type="create" title="暂无通知渠道" description="点击下方按钮添加第一个通知渠道" />
          }}
          columns={[
            {
              title: 'ID',
              dataIndex: 'id',
              width: 70,
              render: (id) => <Tag>#{id}</Tag>
            },
            {
              title: '名称',
              dataIndex: 'name',
              render: (name) => <strong>{name}</strong>
            },
            {
              title: '类型',
              dataIndex: 'type',
              width: 120,
              render: (type) => {
                const colors: Record<string, string> = {
                  telegram: '#0088cc',
                  lark: '#3370ff',
                  wecom: '#07c160',
                  dingtalk: '#0089ff',
                  email: '#ea4335',
                  webhook: '#9c27b0',
                }
                const labels: Record<string, string> = {
                  telegram: 'Telegram',
                  lark: '飞书',
                  wecom: '企业微信',
                  dingtalk: '钉钉',
                  email: '邮件',
                  webhook: 'Webhook',
                }
                return (
                  <Tag color={colors[type] || 'default'}>{labels[type] || type}</Tag>
                )
              }
            },
            {
              title: '状态',
              dataIndex: 'enabled',
              width: 90,
              render: (v: boolean) => <StatusTag status={v ? 'success' : 'default'} text={v ? '已启用' : '已停用'} />
            },
            {
              title: '操作',
              width: 200,
              render: (_, r) => (
                <Space>
                  <Button
                    type="text"
                    size="small"
                    onClick={() => testSend(r.id)}
                  >
                    测试
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
                    onClick={() => deleteOne(r.id)}
                  >
                    删除
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      </motion.div>

      <Modal
        title={modalOpen && typeof modalOpen === 'object' && 'id' in modalOpen ? '编辑渠道' : '新建渠道'}
        open={!!modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        width={560}
      >
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item name="name" label="渠道名称" rules={[{ required: true, message: '请输入渠道名称' }]}>
            <Input placeholder="例如：运维值班群" />
          </Form.Item>
          
          <Form.Item name="type" label="渠道类型" rules={[{ required: true, message: '请选择渠道类型' }]}>
            <Select options={TYPE_OPTIONS} placeholder="选择类型" />
          </Form.Item>
          
          <Form.Item name="config" label="配置信息（JSON）">
            <Input.TextArea
              rows={4}
              placeholder={`{
  "token": "your-bot-token",
  "chat_id": "your-chat-id"
}`}
            />
          </Form.Item>
          
          <Form.Item name="enabled" label="启用状态" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
          
          <Form.Item style={{ marginTop: 24, marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" size="large" block>保存</Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
