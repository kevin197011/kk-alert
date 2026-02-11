import { useEffect, useState } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, Select, Card } from 'antd'
import { motion } from 'framer-motion'
import { PlusOutlined, EditOutlined, DeleteOutlined, UserOutlined } from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, EmptyState } from '../components/ui'
import dayjs from 'dayjs'

type UserRow = { id: number; username: string; role: string; created_at: string }

const ROLE_OPTIONS = [
  { value: 'admin', label: '管理员' },
  { value: 'user', label: '普通用户' },
]

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
            subTitle="仅管理员可在此创建用户"
            action={
              <Button type="primary" icon={<PlusOutlined />} onClick={() => { setModalOpen(true); form.resetFields() }}>
                新建用户
              </Button>
            }
          />
        ) : (
          <EmptyState title="无权限访问" subTitle="仅管理员可管理用户" />
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
