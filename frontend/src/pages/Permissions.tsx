import { Card, Table, Tag, Typography } from 'antd'
import { motion } from 'framer-motion'
import { SafetyOutlined, TeamOutlined } from '@ant-design/icons'
import { PageHeader } from '../components/ui'

const { Text } = Typography

const ROLE_PERMISSIONS = [
  {
    role: '管理员',
    roleTag: 'admin',
    menus: ['仪表盘', '告警历史', '统计报表', '规则管理', '数据源', '通知渠道', '通知模板', '用户管理', '权限管理'],
    desc: '拥有所有菜单和功能的访问权限',
  },
  {
    role: '普通用户',
    roleTag: 'user',
    menus: ['仪表盘', '告警历史', '统计报表'],
    desc: '仅可查看仪表盘、告警历史与统计报表',
  },
]

export default function Permissions() {
  return (
    <div className="permissions-page">
      <PageHeader
        title="权限管理"
        subtitle="查看各角色可访问的菜单与功能"
      />

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
        <Card variant="borderless">
          <Table
            dataSource={ROLE_PERMISSIONS}
            rowKey="roleTag"
            pagination={false}
            columns={[
              {
                title: '角色',
                dataIndex: 'role',
                width: 120,
                render: (role: string, row: (typeof ROLE_PERMISSIONS)[0]) => (
                  <span>
                    {row.roleTag === 'admin' ? (
                      <SafetyOutlined style={{ marginRight: 8, color: '#1890ff' }} />
                    ) : (
                      <TeamOutlined style={{ marginRight: 8, color: '#52c41a' }} />
                    )}
                    <Tag color={row.roleTag === 'admin' ? 'blue' : 'green'}>{role}</Tag>
                  </span>
                ),
              },
              {
                title: '可访问菜单',
                dataIndex: 'menus',
                render: (menus: string[]) => (
                  <span>
                    {menus.map((m) => (
                      <Tag key={m} style={{ marginBottom: 4 }}>
                        {m}
                      </Tag>
                    ))}
                  </span>
                ),
              },
              {
                title: '说明',
                dataIndex: 'desc',
                render: (desc: string) => <Text type="secondary">{desc}</Text>,
              },
            ]}
          />
        </Card>
      </motion.div>
    </div>
  )
}
