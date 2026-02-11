import { useState, useEffect, useMemo } from 'react'
import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom'
import { authHeaders } from '../auth'
import { App, Layout as AntLayout, Menu, Button, Badge, Tooltip, Avatar, Typography, Modal, Form, InputNumber, Space, Input, Dropdown } from 'antd'
import { motion, AnimatePresence } from 'framer-motion'
import {
  DashboardOutlined,
  HistoryOutlined,
  FilterOutlined,
  DatabaseOutlined,
  NotificationOutlined,
  FileTextOutlined,
  BarChartOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
  BellOutlined,
  UserOutlined,
  SettingOutlined,
  ApiOutlined,
  KeyOutlined,
} from '@ant-design/icons'
import { useAuth, type UserRole } from '../auth'

const { Header, Sider, Content, Footer } = AntLayout
const { Text } = Typography

const allNavItems = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘', roles: ['admin', 'user'] as UserRole[] },
  { key: '/alerts', icon: <HistoryOutlined />, label: '告警历史', badgeFromFiring: true, roles: ['admin', 'user'] as UserRole[] },
  { key: '/reports', icon: <BarChartOutlined />, label: '统计报表', roles: ['admin', 'user'] as UserRole[] },
  { key: '/rules', icon: <FilterOutlined />, label: '规则管理', roles: ['admin'] as UserRole[] },
  { key: '/datasources', icon: <DatabaseOutlined />, label: '数据源', roles: ['admin'] as UserRole[] },
  { key: '/channels', icon: <NotificationOutlined />, label: '通知渠道', roles: ['admin'] as UserRole[] },
  { key: '/templates', icon: <FileTextOutlined />, label: '通知模板', roles: ['admin'] as UserRole[] },
  { key: '/users', icon: <UserOutlined />, label: '用户管理', roles: ['admin'] as UserRole[] },
  { key: '/permissions', icon: <SettingOutlined />, label: '权限管理', roles: ['admin'] as UserRole[] },
]

const FIRING_POLL_INTERVAL_MS = 15000

const DEFAULT_RETENTION_DAYS = 90

export default function Layout() {
  const [collapsed, setCollapsed] = useState(false)
  const [firingCount, setFiringCount] = useState(0)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [tokenModalOpen, setTokenModalOpen] = useState(false)
  const [retentionDays, setRetentionDays] = useState(DEFAULT_RETENTION_DAYS)
  const [settingsSaving, setSettingsSaving] = useState(false)
  const { logout, user, token } = useAuth()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const { message } = App.useApp()

  const navItems = useMemo(() => {
    const role = user?.role ?? 'user'
    return allNavItems.filter((item) => item.roles.includes(role))
  }, [user?.role])

  useEffect(() => {
    if (settingsOpen) {
      fetch('/api/v1/settings', { headers: authHeaders() })
        .then((r) => r.ok ? r.json() : { retention_days: DEFAULT_RETENTION_DAYS })
        .then((d) => setRetentionDays(d.retention_days ?? DEFAULT_RETENTION_DAYS))
        .catch(() => setRetentionDays(DEFAULT_RETENTION_DAYS))
    }
  }, [settingsOpen])

  const onSaveSettings = () => {
    setSettingsSaving(true)
    fetch('/api/v1/settings', {
      method: 'PUT',
      headers: authHeaders(),
      body: JSON.stringify({ retention_days: retentionDays }),
    })
      .then((r) => {
        if (r.ok) {
          message.success('保存成功')
          setSettingsOpen(false)
        } else {
          return r.json().then((d) => {
            message.error(d?.error || '保存失败')
          })
        }
      })
      .catch(() => message.error('保存失败'))
      .finally(() => setSettingsSaving(false))
  }

  useEffect(() => {
    const fetchFiringCount = () => {
      fetch('/api/v1/alerts?page=1&page_size=1&status=firing', { headers: authHeaders() })
        .then((r) => r.json())
        .then((d) => setFiringCount(d.total ?? 0))
        .catch(() => {})
    }
    fetchFiringCount()
    const timer = setInterval(fetchFiringCount, FIRING_POLL_INTERVAL_MS)
    return () => clearInterval(timer)
  }, [])

  const menuItems = useMemo(() => navItems.map((item: (typeof allNavItems)[0]) => {
    const badgeCount = item.badgeFromFiring ? firingCount : undefined
    return {
      key: item.key,
      icon: item.icon,
      label: (
        <Link to={item.key} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span>{item.label}</span>
          {badgeCount !== undefined && badgeCount > 0 && (
            <Badge
              count={badgeCount}
              size="small"
              style={{
                backgroundColor: 'var(--color-cta)',
                color: 'var(--color-primary)',
                fontWeight: 600,
              }}
            />
          )}
        </Link>
      ),
    }
  }), [navItems, firingCount])

  return (
    <AntLayout className="app-layout" style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={240}
        collapsedWidth={80}
        style={{
          background: 'var(--color-primary)',
          overflow: 'hidden',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 100,
          boxShadow: '2px 0 8px rgba(0,0,0,0.15)',
        }}
      >
        <motion.div
          initial={false}
          animate={{ 
            paddingLeft: collapsed ? 0 : 20,
            justifyContent: collapsed ? 'center' : 'flex-start'
          }}
          transition={{ duration: 0.2 }}
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            color: 'var(--color-background)',
            fontFamily: 'var(--font-heading)',
            fontWeight: 700,
            fontSize: 20,
            borderBottom: '1px solid rgba(255,255,255,0.1)',
            cursor: 'pointer',
          }}
          onClick={() => navigate('/')}
        >
          <motion.div
            initial={false}
            animate={{ scale: collapsed ? 0.9 : 1 }}
            transition={{ duration: 0.2 }}
            style={{
              width: 40,
              height: 40,
              background: 'var(--color-cta)',
              borderRadius: 10,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--color-primary)',
              fontSize: 22,
              marginRight: collapsed ? 0 : 12,
              flexShrink: 0,
            }}
          >
            <BellOutlined />
          </motion.div>
          <AnimatePresence>
            {!collapsed && (
              <motion.span
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -10 }}
                transition={{ duration: 0.15 }}
              >
                KK Alert
              </motion.span>
            )}
          </AnimatePresence>
        </motion.div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[pathname]}
          items={menuItems}
          style={{
            marginTop: 12,
            background: 'transparent',
            border: 'none',
            color: 'rgba(255,255,255,0.85)',
          }}
          className="sidebar-menu"
        />

        <motion.div
          initial={false}
          style={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            padding: '16px',
            borderTop: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          <Tooltip title={collapsed ? '展开侧边栏' : '收起侧边栏'} placement="right">
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
              style={{
                width: collapsed ? 48 : '100%',
                height: 40,
                color: 'rgba(255,255,255,0.7)',
                transition: 'all 200ms ease',
                display: 'flex',
                alignItems: 'center',
                justifyContent: collapsed ? 'center' : 'flex-start',
                paddingLeft: collapsed ? 0 : 16,
              }}
              className="sidebar-toggle"
            >
              <AnimatePresence>
                {!collapsed && (
                  <motion.span
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.15 }}
                    style={{ marginLeft: 8 }}
                  >
                    收起
                  </motion.span>
                )}
              </AnimatePresence>
            </Button>
          </Tooltip>
        </motion.div>
      </Sider>

      <AntLayout 
        style={{ 
          marginLeft: collapsed ? 80 : 240, 
          transition: 'margin-left 200ms cubic-bezier(0.4, 0, 0.2, 1)',
          minHeight: '100vh',
        }}
      >
        <Header
          style={{
            background: 'var(--color-background)',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: '1px solid var(--color-border)',
            height: 64,
            position: 'sticky',
            top: 0,
            zIndex: 99,
            boxShadow: '0 1px 4px rgba(0,0,0,0.05)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Text type="secondary" style={{ fontSize: 14 }}>
              {allNavItems.find(item => item.key === pathname)?.label || 'Dashboard'}
            </Text>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Tooltip title={firingCount > 0 ? `当前告警中：${firingCount} 条` : '告警通知'}>
              <Badge count={firingCount} size="small" offset={[-2, 2]}>
                <Button
                  type="text"
                  icon={<BellOutlined />}
                  style={{ 
                    color: 'var(--color-secondary)',
                    width: 40,
                    height: 40,
                  }}
                  className="header-btn"
                  onClick={() => navigate('/alerts')}
                />
              </Badge>
            </Tooltip>

            <Tooltip title="API 文档">
              <Button
                type="text"
                icon={<ApiOutlined />}
                onClick={() => window.open('/swagger/', '_blank')}
                style={{ 
                  color: 'var(--color-secondary)',
                  width: 40,
                  height: 40,
                }}
                className="header-btn"
              />
            </Tooltip>

            <Tooltip title="设置">
              <Button
                type="text"
                icon={<SettingOutlined />}
                onClick={() => setSettingsOpen(true)}
                style={{ 
                  color: 'var(--color-secondary)',
                  width: 40,
                  height: 40,
                }}
                className="header-btn"
              />
            </Tooltip>

            <div style={{ 
              width: 1, 
              height: 24, 
              background: 'var(--color-border)', 
              margin: '0 8px' 
            }} />

            <Dropdown
              menu={{
                items: [
                  {
                    key: 'token',
                    icon: <KeyOutlined />,
                    label: 'Token 管理',
                    onClick: () => setTokenModalOpen(true),
                  },
                  {
                    key: 'logout',
                    icon: <LogoutOutlined />,
                    label: '退出登录',
                    onClick: () => { logout(); navigate('/login') },
                  },
                ],
              }}
              trigger={['click']}
              placement="bottomRight"
            >
              <div
                className="user-info"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  cursor: 'pointer',
                  padding: '6px 12px',
                  borderRadius: 8,
                  transition: 'background 0.2s',
                  flexShrink: 0,
                  minWidth: 0,
                }}
              >
                <Avatar
                  size="small"
                  icon={<UserOutlined />}
                  style={{
                    background: 'var(--color-cta)',
                    color: 'var(--color-primary)',
                    flexShrink: 0,
                  }}
                />
                <div
                  style={{
                    overflow: 'hidden',
                    minWidth: 0,
                    maxWidth: 160,
                  }}
                >
                  <Text
                    strong
                    ellipsis
                    style={{ display: 'block', fontSize: 13, lineHeight: 1.4 }}
                  >
                    {user?.username ?? '-'}
                  </Text>
                  <Text
                    type="secondary"
                    style={{ display: 'block', fontSize: 12, lineHeight: 1.4 }}
                  >
                    {user?.role === 'admin' ? '管理员' : '普通用户'}
                  </Text>
                </div>
              </div>
            </Dropdown>
          </div>
        </Header>

        <Content
          style={{
            padding: 24,
            background: '#f8fafc',
            minHeight: 'calc(100vh - 64px - 48px)',
          }}
        >
          <motion.div
            key={pathname}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Outlet />
          </motion.div>
        </Content>
        <Footer
          style={{
            textAlign: 'center',
            color: 'var(--color-secondary)',
            fontSize: 12,
            padding: '12px 24px',
            borderTop: '1px solid var(--color-border)',
            background: '#fff',
          }}
        >
          系统运行部驱动
        </Footer>
      </AntLayout>

      <Modal
        title="系统设置"
        open={settingsOpen}
        onCancel={() => setSettingsOpen(false)}
        footer={[
          <Button key="cancel" onClick={() => setSettingsOpen(false)}>取消</Button>,
          user?.role === 'admin' ? (
            <Button key="save" type="primary" loading={settingsSaving} onClick={onSaveSettings}>
              保存
            </Button>
          ) : null,
        ]}
        destroyOnHidden
      >
        <Form layout="vertical" style={{ marginTop: 8 }}>
          <Form.Item
            label="历史数据保留天数"
            extra="默认 3 个月（90 天）。超过保留天数的告警历史将被自动清理，避免数据过大影响性能。"
          >
            <Space.Compact style={{ width: '100%' }}>
              <InputNumber
                min={1}
                max={3650}
                value={retentionDays}
                onChange={(v) => setRetentionDays(v ?? DEFAULT_RETENTION_DAYS)}
                style={{ width: '100%' }}
                disabled={user?.role !== 'admin'}
              />
              <Input readOnly value="天" style={{ width: 40, textAlign: 'center', background: 'rgba(0,0,0,0.02)' }} />
            </Space.Compact>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Token 管理"
        open={tokenModalOpen}
        onCancel={() => setTokenModalOpen(false)}
        footer={[
          <Button key="close" onClick={() => setTokenModalOpen(false)}>关闭</Button>,
          <Button
            key="copy"
            type="primary"
            onClick={() => {
              if (token) {
                navigator.clipboard.writeText(token)
                message.success('Token 已复制到剪贴板')
              }
            }}
            disabled={!token}
          >
            复制 Token
          </Button>,
        ]}
        destroyOnHidden
      >
        <p style={{ color: 'var(--color-secondary)', marginBottom: 12 }}>
          当前 Token 与登录账号权限一致：{user?.role === 'admin' ? '管理员' : '普通用户'}。用于 API、Swagger 等场景时，在请求头携带 <Text code>Authorization: Bearer &lt;token&gt;</Text>。
        </p>
        <Input.TextArea
          readOnly
          value={token ?? ''}
          placeholder="未获取到 Token，请重新登录"
          rows={4}
          style={{ fontFamily: 'monospace', fontSize: 12 }}
        />
      </Modal>
    </AntLayout>
  )
}
