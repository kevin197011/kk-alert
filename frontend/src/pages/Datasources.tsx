import { useEffect, useState } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, Select, Switch, Card, Tag, Row, Col } from 'antd'
import { motion } from 'framer-motion'
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, DatabaseOutlined } from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, StatusTag, EmptyState, StatCard } from '../components/ui'

type Datasource = { id: number; name: string; type: string; endpoint: string; enabled: boolean }

const TYPE_OPTIONS = [
  { value: 'prometheus', label: 'Prometheus' },
  { value: 'victoriametrics', label: 'VictoriaMetrics' },
  { value: 'elasticsearch', label: 'Elasticsearch' },
  { value: 'doris', label: 'Doris' },
]

export default function Datasources() {
  const { message, modal } = App.useApp()
  const [list, setList] = useState<Datasource[]>([])
  const [loading, setLoading] = useState(true)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [modalOpen, setModalOpen] = useState<boolean | { id: number }>(false)
  const [form] = Form.useForm()

  const load = () => {
    setLoading(true)
    fetch('/api/v1/datasources', { headers: authHeaders() })
      .then((r) => r.json())
      .then((data) => setList(Array.isArray(data) ? data : []))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [])

  const onFinish = async (v: any) => {
    const isEdit = typeof modalOpen === 'object' && modalOpen !== null && 'id' in modalOpen
    const url = isEdit ? `/api/v1/datasources/${(modalOpen as any).id}` : '/api/v1/datasources'
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
        fetch(`/api/v1/datasources/${id}`, { method: 'DELETE', headers: authHeaders() }).then((r) => {
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

  const testOne = async (id: number) => {
    setTestingId(id)
    try {
      const res = await fetch(`/api/v1/datasources/${id}/test`, { method: 'POST', headers: authHeaders() })
      const data = await res.json().catch(() => ({}))
      if (res.ok) {
        message.success(data.message || '连接测试成功')
      } else {
        message.error(data.error || '连接测试失败')
      }
    } finally {
      setTestingId(null)
    }
  }

  const enabledCount = list.filter(d => d.enabled).length

  return (
    <div className="datasources-page">
      <PageHeader
        title="数据源"
        subtitle="配置和管理告警数据源"
        actions={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setModalOpen(true); form.resetFields() }}
            size="large"
          >
            新建数据源
          </Button>
        }
      />

      <Row gutter={[24, 24]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="总数据源" value={list.length} icon={<DatabaseOutlined />} color="#1890ff" delay={0} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="已启用" value={enabledCount} icon={<ThunderboltOutlined />} color="#52c41a" delay={0.1} />
        </Col>
      </Row>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
      <Card variant="borderless" className="datasources-table-card">
        <Table
          loading={loading}
          dataSource={list}
          rowKey="id"
          locale={{
            emptyText: <EmptyState type="create" title="暂无数据源" description="点击下方按钮添加第一个数据源" />
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
              width: 150,
              render: (type) => (
                <Tag color="blue">{type}</Tag>
              )
            },
            {
              title: '地址',
              dataIndex: 'endpoint',
              ellipsis: true,
              render: (endpoint) => endpoint || '-'
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
                    loading={testingId === r.id}
                    onClick={() => testOne(r.id)}
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
        title={typeof modalOpen === 'object' && modalOpen && 'id' in modalOpen ? '编辑数据源' : '新建数据源'}
        open={!!modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        width={560}
      >
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item name="name" label="数据源名称" rules={[{ required: true, message: '请输入数据源名称' }]}>
            <Input placeholder="例如：生产环境 Prometheus" />
          </Form.Item>
          
          <Form.Item name="type" label="数据源类型" rules={[{ required: true, message: '请选择数据源类型' }]}>
            <Select options={TYPE_OPTIONS} placeholder="选择类型" />
          </Form.Item>
          
          <Form.Item name="endpoint" label="连接地址">
            <Input placeholder="例如：http://localhost:9090" />
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
