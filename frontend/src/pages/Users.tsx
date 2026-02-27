import { useEffect, useState } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, Select, Card, Switch, Collapse, Tag, Typography } from 'antd'
import { motion } from 'framer-motion'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UserOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, EmptyState } from '../components/ui'
import dayjs from 'dayjs'

const { Text } = Typography

type UserRow = { id: number; username: string; role: string; created_at: string }

const ROLE_OPTIONS = [
  { value: 'admin', label: '管理员' },
  { value: 'user', label: '普通用户' },
]

// ---------- OIDC Configuration Panel ----------

type OIDCFormValues = {
  enabled: boolean
  issuer: string
  client_id: string
  client_secret: string
  redirect_uri: string
  scopes: string
  display_name: string
  auto_register: boolean
  default_role: string
}

function OIDCSettings() {
  const { message } = App.useApp()
  const [form] = Form.useForm<OIDCFormValues>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; error?: string } | null>(null)

  const loadConfig = () => {
    setLoading(true)
    fetch('/api/v1/oidc/config', { headers: authHeaders() })
      .then((r) => r.json())
      .then((data) => {
        form.setFieldsValue({
          enabled: data.enabled ?? false,
          issuer: data.issuer ?? '',
          client_id: data.client_id ?? '',
          client_secret: data.client_secret ?? '',
          redirect_uri: data.redirect_uri ?? '',
          scopes: data.scopes ?? 'openid profile email',
          display_name: data.display_name ?? 'SSO',
          auto_register: data.auto_register ?? true,
          default_role: data.default_role ?? 'user',
        })
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadConfig() }, [])

  const onSave = async (values: OIDCFormValues) => {
    setSaving(true)
    try {
      const res = await fetch('/api/v1/oidc/config', {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify(values),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        message.error(data.error || '保存失败')
        return
      }
      message.success('OIDC 配置已保存')
      setTestResult(null)
    } catch {
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  const onTest = async () => {
    setTesting(true)
    setTestResult(null)
    try {
      const res = await fetch('/api/v1/oidc/test', {
        method: 'POST',
        headers: authHeaders(),
      })
      const data = await res.json()
      setTestResult(data)
      if (data.success) {
        message.success('OIDC 连接测试成功')
      } else {
        message.error(`OIDC 测试失败: ${data.error}`)
      }
    } catch {
      setTestResult({ success: false, error: 'network error' })
      message.error('OIDC 连接测试失败')
    } finally {
      setTesting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={onSave} disabled={loading}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <Form.Item name="enabled" valuePropName="checked" noStyle>
          <Switch />
        </Form.Item>
        <Text strong>启用 OIDC 单点登录</Text>
        {testResult && (
          <Tag
            icon={testResult.success ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
            color={testResult.success ? 'success' : 'error'}
          >
            {testResult.success ? '连接正常' : '连接失败'}
          </Tag>
        )}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 20px' }}>
        <Form.Item name="issuer" label="Issuer URL" rules={[{ required: true, message: '请输入 Issuer URL' }]}>
          <Input placeholder="https://accounts.google.com" />
        </Form.Item>
        <Form.Item name="client_id" label="Client ID" rules={[{ required: true, message: '请输入 Client ID' }]}>
          <Input placeholder="your-client-id" />
        </Form.Item>
        <Form.Item name="client_secret" label="Client Secret" rules={[{ required: true, message: '请输入 Client Secret' }]}>
          <Input.Password placeholder="your-client-secret" />
        </Form.Item>
        <Form.Item name="redirect_uri" label="Redirect URI" rules={[{ required: true, message: '请输入 Redirect URI' }]}
          extra="格式: http(s)://your-domain/api/v1/auth/oidc/callback"
        >
          <Input placeholder="http://localhost:3000/api/v1/auth/oidc/callback" />
        </Form.Item>
        <Form.Item name="scopes" label="Scopes">
          <Input placeholder="openid profile email" />
        </Form.Item>
        <Form.Item name="display_name" label="登录按钮名称">
          <Input placeholder="SSO" />
        </Form.Item>
        <Form.Item name="auto_register" label="自动注册" valuePropName="checked"
          extra="开启后，首次通过 OIDC 登录的用户将自动创建本地账号"
        >
          <Switch />
        </Form.Item>
        <Form.Item name="default_role" label="默认角色" extra="自动注册用户的默认角色">
          <Select options={ROLE_OPTIONS} />
        </Form.Item>
      </div>

      <Space>
        <Button type="primary" htmlType="submit" loading={saving}>保存配置</Button>
        <Button onClick={onTest} loading={testing}>测试连接</Button>
      </Space>
    </Form>
  )
}

export default function Users() {
  const { message, modal } = App.useApp()
  const [list, setList] = useState<UserRow[]>([])
  const [loading, setLoading] = useState(true)
  const [canManage, setCanManage] = useState(true)
  const [modalOpen, setModalOpen] = useState<boolean | { id: number; username: string; role: string }>(false)
  const [form] = Form.useForm()

  const load = () => {
    setLoading(true)
    fetch('/api/v1/users', { headers: authHeaders() })
      .then((r) => {
        if (r.status === 403) {
          setCanManage(false)
          return []
        }
        return r.json()
      })
      .then((data) => setList(Array.isArray(data) ? data : []))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [])

  const onFinish = async (v: { username?: string; password?: string; role: string }) => {
    const isEdit = typeof modalOpen === 'object' && modalOpen !== null && 'id' in modalOpen
    if (isEdit) {
      const body: { role: string; password?: string } = { role: v.role }
      if (v.password && v.password.trim()) body.password = v.password
      const res = await fetch(`/api/v1/users/${(modalOpen as { id: number }).id}`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        message.error(data.error || '更新失败')
        return
      }
    } else {
      if (!v.username?.trim()) {
        message.error('请输入用户名')
        return
      }
      const res = await fetch('/api/v1/users', {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify({ username: v.username.trim(), password: v.password || '', role: v.role }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        message.error(data.error || '创建失败')
        return
      }
    }
    message.success(isEdit ? '更新成功' : '创建成功')
    setModalOpen(false)
    form.resetFields()
    load()
  }

  const deleteOne = (row: UserRow) => {
    if (row.username === 'admin') {
      message.error('不能删除管理员账号')
      return
    }
    modal.confirm({
      title: '确认删除',
      content: `确定要删除用户「${row.username}」吗？`,
      onOk: () =>
        fetch(`/api/v1/users/${row.id}`, { method: 'DELETE', headers: authHeaders() }).then((r) => {
          if (r.ok) {
            message.success('删除成功')
            load()
          } else {
            r.json().then((d) => message.error(d.error || '删除失败'))
          }
        }),
    })
  }

  if (list.length === 0 && !loading) {
    return (
      <div className="users-page">
        <PageHeader title="用户管理" subtitle="管理系统用户与角色" />
        {canManage ? (
          <EmptyState
            title="暂无用户"
            description="仅管理员可在此创建用户"
            action={{ text: '新建用户', onClick: () => { setModalOpen(true); form.resetFields() } }}
          />
        ) : (
          <EmptyState title="无权限访问" description="仅管理员可管理用户" />
        )}
        <Modal
          title="新建用户"
          open={!!modalOpen && modalOpen === true}
          onCancel={() => setModalOpen(false)}
          footer={null}
          destroyOnHidden
        >
          <Form form={form} layout="vertical" onFinish={onFinish} initialValues={{ role: 'user' }}>
            <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
              <Input prefix={<UserOutlined />} placeholder="用户名" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password placeholder="密码" />
            </Form.Item>
            <Form.Item name="role" label="角色" rules={[{ required: true }]}>
              <Select options={ROLE_OPTIONS} />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit">确定</Button>
                <Button onClick={() => setModalOpen(false)}>取消</Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>
      </div>
    )
  }

  return (
    <div className="users-page">
      <PageHeader
        title="用户管理"
        subtitle="管理系统用户与角色"
        actions={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { setModalOpen(true); form.resetFields() }}>
            新建用户
          </Button>
        }
      />

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
        <Card variant="borderless">
          <Table
            dataSource={list}
            rowKey="id"
            loading={loading}
            columns={[
              { title: 'ID', dataIndex: 'id', width: 80 },
              { title: '用户名', dataIndex: 'username' },
              {
                title: '角色',
                dataIndex: 'role',
                render: (role: string) => (role === 'admin' ? '管理员' : '普通用户'),
              },
              {
                title: '创建时间',
                dataIndex: 'created_at',
                render: (t: string) => (t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-'),
              },
              {
                title: '操作',
                key: 'actions',
                width: 160,
                render: (_, row) => (
                  <Space>
                    <Button
                      type="link"
                      size="small"
                      icon={<EditOutlined />}
                      onClick={() => {
                        setModalOpen({ id: row.id, username: row.username, role: row.role })
                        form.setFieldsValue({ username: row.username, role: row.role })
                      }}
                    >
                      编辑
                    </Button>
                    <Button
                      type="link"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      disabled={row.username === 'admin'}
                      onClick={() => deleteOne(row)}
                    >
                      删除
                    </Button>
                  </Space>
                ),
              },
            ]}
            pagination={false}
          />
        </Card>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.2 }}
        style={{ marginTop: 20 }}
      >
        <Collapse
          items={[
            {
              key: 'oidc',
              label: (
                <span style={{ fontWeight: 500 }}>
                  <SafetyCertificateOutlined style={{ marginRight: 8 }} />
                  OIDC 单点登录配置
                </span>
              ),
              children: <OIDCSettings />,
            },
          ]}
        />
      </motion.div>

      <Modal
        title={typeof modalOpen === 'object' && modalOpen !== null && 'id' in modalOpen ? '编辑用户' : '新建用户'}
        open={!!modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{ role: 'user' }}
        >
          {typeof modalOpen === 'object' && modalOpen !== null && 'id' in modalOpen ? (
            <>
              <Form.Item name="username" label="用户名">
                <Input prefix={<UserOutlined />} disabled />
              </Form.Item>
              <Form.Item name="password" label="新密码（不填则不修改）">
                <Input.Password placeholder="留空表示不修改" />
              </Form.Item>
              <Form.Item name="role" label="角色" rules={[{ required: true }]}>
                <Select options={ROLE_OPTIONS} />
              </Form.Item>
            </>
          ) : (
            <>
              <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
                <Input prefix={<UserOutlined />} placeholder="用户名" />
              </Form.Item>
              <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
                <Input.Password placeholder="密码" />
              </Form.Item>
              <Form.Item name="role" label="角色" rules={[{ required: true }]}>
                <Select options={ROLE_OPTIONS} />
              </Form.Item>
            </>
          )}
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">确定</Button>
              <Button onClick={() => setModalOpen(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
