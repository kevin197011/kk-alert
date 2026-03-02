import { useEffect, useState } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, Card, Tag, Transfer, Typography } from 'antd'
import { motion } from 'framer-motion'
import { PlusOutlined, EditOutlined, DeleteOutlined, SafetyOutlined, TeamOutlined } from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, EmptyState } from '../components/ui'

const { Text } = Typography

type Permission = {
  code: string
  name: string
  category: string
}

type Role = {
  id: number
  name: string
  description: string
  permissions: string
  created_at: string
}

const PERMISSION_CATEGORIES: Record<string, { label: string; color: string }> = {
  menu: { label: '菜单', color: 'blue' },
  action: { label: '操作', color: 'green' },
  resource: { label: '资源', color: 'orange' },
}

export default function Roles() {
  const { message, modal } = App.useApp()
  const [roles, setRoles] = useState<Role[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<Role | null>(null)
  const [form] = Form.useForm()
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])

  const loadData = () => {
    setLoading(true)
    Promise.all([
      fetch('/api/v1/roles', { headers: authHeaders() }).then(r => r.json()),
      fetch('/api/v1/permissions', { headers: authHeaders() }).then(r => r.json()),
    ])
      .then(([rolesData, permsData]) => {
        setRoles(Array.isArray(rolesData) ? rolesData : [])
        setPermissions(Array.isArray(permsData) ? permsData : [])
      })
      .catch(() => message.error('加载数据失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadData()
  }, [])

  const handleCreate = () => {
    setEditingRole(null)
    setSelectedPermissions([])
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (role: Role) => {
    setEditingRole(role)
    let perms: string[] = []
    try {
      perms = JSON.parse(role.permissions || '[]')
    } catch {
      perms = []
    }
    setSelectedPermissions(perms)
    form.setFieldsValue({
      name: role.name,
      description: role.description,
    })
    setModalOpen(true)
  }

  const handleDelete = (role: Role) => {
    if (role.name === 'admin' || role.name === 'user') {
      message.error('系统默认角色不能删除')
      return
    }

    modal.confirm({
      title: '确认删除',
      content: `确定要删除角色「${role.name}」吗？`,
      onOk: () => {
        fetch(`/api/v1/roles/${role.id}`, {
          method: 'DELETE',
          headers: authHeaders(),
        })
          .then(r => {
            if (r.ok) {
              message.success('删除成功')
              loadData()
            } else {
              r.json().then(d => message.error(d.error || '删除失败'))
            }
          })
          .catch(() => message.error('删除失败'))
      },
    })
  }

  const handleSave = async (values: { name: string; description: string }) => {
    const body = {
      ...values,
      permissions: selectedPermissions,
    }

    const url = editingRole ? `/api/v1/roles/${editingRole.id}` : '/api/v1/roles'
    const method = editingRole ? 'PUT' : 'POST'

    try {
      const res = await fetch(url, {
        method,
        headers: authHeaders(),
        body: JSON.stringify(body),
      })
      const data = await res.json()

      if (res.ok) {
        message.success(editingRole ? '更新成功' : '创建成功')
        setModalOpen(false)
        loadData()
      } else {
        message.error(data.error || '保存失败')
      }
    } catch {
      message.error('保存失败')
    }
  }

  const transferData = permissions.map(p => ({
    key: p.code,
    title: p.name,
    description: p.code,
    category: p.category,
  }))

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '角色名称',
      dataIndex: 'name',
      render: (name: string) => (
        <Space>
          {name === 'admin' ? (
            <SafetyOutlined style={{ color: '#1890ff' }} />
          ) : name === 'user' ? (
            <TeamOutlined style={{ color: '#52c41a' }} />
          ) : null}
          <strong>{name}</strong>
          {name === 'admin' && <Tag color="blue">系统</Tag>}
          {name === 'user' && <Tag color="green">系统</Tag>}
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      render: (desc: string) => desc || '-',
    },
    {
      title: '权限数量',
      dataIndex: 'permissions',
      width: 120,
      render: (perms: string) => {
        try {
          const list = JSON.parse(perms || '[]')
          return <Tag>{list.length} 个权限</Tag>
        } catch {
          return <Tag>0 个权限</Tag>
        }
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: any, record: Role) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            disabled={record.name === 'admin' || record.name === 'user'}
            onClick={() => handleDelete(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div className="roles-page">
      <PageHeader
        title="角色管理"
        subtitle="管理系统角色和权限配置"
        actions={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleCreate}
          >
            新建角色
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
            loading={loading}
            dataSource={roles}
            rowKey="id"
            columns={columns}
            locale={{
              emptyText: <EmptyState title="暂无角色" description="点击下方按钮创建第一个角色" />,
            }}
          />
        </Card>
      </motion.div>

      <Modal
        title={editingRole ? '编辑角色' : '新建角色'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        width={700}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{ name: '', description: '' }}
        >
          <Form.Item
            name="name"
            label="角色名称"
            rules={[
              { required: true, message: '请输入角色名称' },
              { pattern: /^[a-z0-9_]+$/, message: '只能使用小写字母、数字和下划线' },
            ]}
          >
            <Input
              placeholder="例如：operator"
              disabled={editingRole?.name === 'admin' || editingRole?.name === 'user'}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea
              rows={2}
              placeholder="角色描述（可选）"
            />
          </Form.Item>

          <Form.Item label="权限配置">
            <Transfer
              dataSource={transferData}
              titles={['可用权限', '已选权限']}
              targetKeys={selectedPermissions}
              onChange={(keys) => setSelectedPermissions(keys as string[])}
              render={item => (
                <Space direction="vertical" size={0} style={{ display: 'flex' }}>
                  <Text strong>{item.title}</Text>
                  <Space>
                    <Tag color={PERMISSION_CATEGORIES[item.category]?.color || 'default'}>
                      {PERMISSION_CATEGORIES[item.category]?.label || item.category}
                    </Tag>
                    <Text type="secondary" style={{ fontSize: 12 }}>{item.description}</Text>
                  </Space>
                </Space>
              )}
              listStyle={{
                width: 300,
                height: 400,
              }}
              showSearch
              filterOption={(inputValue, item) =>
                item.title!.toLowerCase().includes(inputValue.toLowerCase()) ||
                item.description!.toLowerCase().includes(inputValue.toLowerCase())
              }
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
                {editingRole ? '更新' : '创建'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
