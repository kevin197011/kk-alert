import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { App, Button, Form, Input, Card, Typography, Space, Steps, Result } from 'antd'
import { motion } from 'framer-motion'
import { ArrowLeftOutlined, LockOutlined, SafetyOutlined, CheckCircleOutlined } from '@ant-design/icons'

const { Title, Text, Link } = Typography

export default function ForgotPassword() {
  const navigate = useNavigate()
  const { message } = App.useApp()
  const [currentStep, setCurrentStep] = useState(0)
  const [loading, setLoading] = useState(false)
  const [username, setUsername] = useState('')
  const [resetToken, setResetToken] = useState('')

  // Step 1: Request reset token
  const handleForgotPassword = async (values: { username: string }) => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: values.username }),
      })
      const data = await res.json()
      
      if (res.ok) {
        setUsername(values.username)
        setResetToken(data.token || '')
        setCurrentStep(1)
        message.success('重置令牌已生成')
      } else {
        message.error(data.error || '请求失败')
      }
    } catch {
      message.error('网络错误')
    } finally {
      setLoading(false)
    }
  }

  // Step 2: Reset password with token
  const handleResetPassword = async (values: { token: string; new_password: string }) => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          token: values.token,
          new_password: values.new_password,
        }),
      })
      const data = await res.json()
      
      if (res.ok) {
        setCurrentStep(2)
        message.success('密码重置成功')
      } else {
        message.error(data.error || '重置失败')
      }
    } catch {
      message.error('网络错误')
    } finally {
      setLoading(false)
    }
  }

  const steps = [
    {
      title: '验证身份',
      icon: <SafetyOutlined />,
    },
    {
      title: '重置密码',
      icon: <LockOutlined />,
    },
    {
      title: '完成',
      icon: <CheckCircleOutlined />,
    },
  ]

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        padding: '20px',
      }}
    >
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        style={{ width: '100%', maxWidth: 480 }}
      >
        <Card
          style={{
            borderRadius: 16,
            boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
          }}
        >
          <div style={{ textAlign: 'center', marginBottom: 32 }}>
            <Title level={3} style={{ margin: 0, marginBottom: 8 }}>
              重置密码
            </Title>
            <Text type="secondary">忘记密码？我们来帮您重置</Text>
          </div>

          <Steps
            current={currentStep}
            items={steps}
            style={{ marginBottom: 32 }}
          />

          {currentStep === 0 && (
            <Form
              layout="vertical"
              onFinish={handleForgotPassword}
              autoComplete="off"
            >
              <Form.Item
                name="username"
                label="用户名"
                rules={[{ required: true, message: '请输入用户名' }]}
              >
                <Input
                  prefix={<SafetyOutlined />}
                  placeholder="请输入您的用户名"
                  size="large"
                />
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={loading}
                  size="large"
                  block
                >
                  获取重置令牌
                </Button>
              </Form.Item>

              <div style={{ textAlign: 'center' }}>
                <Link onClick={() => navigate('/login')}>
                  <ArrowLeftOutlined /> 返回登录
                </Link>
              </div>
            </Form>
          )}

          {currentStep === 1 && (
            <Form
              layout="vertical"
              onFinish={handleResetPassword}
              autoComplete="off"
              initialValues={{ token: resetToken }}
            >
              <div style={{ marginBottom: 16, padding: 12, background: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 8 }}>
                <Text type="success">
                  <CheckCircleOutlined style={{ marginRight: 8 }} />
                  重置令牌已生成，请联系管理员获取令牌
                </Text>
              </div>

              <Form.Item
                name="token"
                label="重置令牌"
                rules={[{ required: true, message: '请输入重置令牌' }]}
              >
                <Input.TextArea
                  placeholder="请输入重置令牌"
                  rows={3}
                />
              </Form.Item>

              <Form.Item
                name="new_password"
                label="新密码"
                rules={[
                  { required: true, message: '请输入新密码' },
                  { min: 6, message: '密码至少6位' },
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="请输入新密码（至少6位）"
                  size="large"
                />
              </Form.Item>

              <Form.Item
                name="confirm_password"
                label="确认新密码"
                dependencies={['new_password']}
                rules={[
                  { required: true, message: '请确认新密码' },
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      if (!value || getFieldValue('new_password') === value) {
                        return Promise.resolve()
                      }
                      return Promise.reject(new Error('两次输入的密码不一致'))
                    },
                  }),
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined />}
                  placeholder="请再次输入新密码"
                  size="large"
                />
              </Form.Item>

              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={loading}
                  size="large"
                  block
                >
                  重置密码
                </Button>
              </Form.Item>

              <div style={{ textAlign: 'center' }}>
                <Link onClick={() => setCurrentStep(0)}>
                  <ArrowLeftOutlined /> 上一步
                </Link>
              </div>
            </Form>
          )}

          {currentStep === 2 && (
            <Result
              status="success"
              title="密码重置成功"
              subTitle="您的密码已成功重置，请使用新密码登录"
              extra={[
                <Button
                  type="primary"
                  key="login"
                  onClick={() => navigate('/login')}
                >
                  去登录
                </Button>,
              ]}
            />
          )}
        </Card>
      </motion.div>
    </div>
  )
}
