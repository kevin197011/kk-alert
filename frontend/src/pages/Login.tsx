import { useState, useEffect, useRef } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { App, Form, Input, Button, Typography, Checkbox } from 'antd'
import { motion, AnimatePresence } from 'framer-motion'
import {
  UserOutlined,
  LockOutlined,
  BellOutlined,
  LoginOutlined,
  SafetyOutlined,
  DashboardOutlined,
  NotificationOutlined,
} from '@ant-design/icons'
import { useAuth } from '../auth'

const { Title, Text } = Typography

// ---------- Particle Background ----------
interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  size: number
  opacity: number
  color: string
}

const PARTICLE_COLORS = ['#6366f1', '#8b5cf6', '#ec4899', '#10b981', '#f59e0b']

function ParticleBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const particlesRef = useRef<Particle[]>([])
  const mouseRef = useRef({ x: 0, y: 0 })
  const animRef = useRef<number>(0)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const resize = () => {
      canvas.width = window.innerWidth
      canvas.height = window.innerHeight
    }
    resize()
    window.addEventListener('resize', resize)

    // Initialize particles
    const particleCount = Math.min(50, Math.floor((canvas.width * canvas.height) / 25000))
    particlesRef.current = Array.from({ length: particleCount }, () => ({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      vx: (Math.random() - 0.5) * 0.5,
      vy: (Math.random() - 0.5) * 0.5,
      size: Math.random() * 3 + 1,
      opacity: Math.random() * 0.5 + 0.2,
      color: PARTICLE_COLORS[Math.floor(Math.random() * PARTICLE_COLORS.length)],
    }))

    const handleMouseMove = (e: MouseEvent) => {
      mouseRef.current = { x: e.clientX, y: e.clientY }
    }
    window.addEventListener('mousemove', handleMouseMove)

    const tick = () => {
      ctx.fillStyle = 'rgba(15, 23, 42, 0.1)'
      ctx.fillRect(0, 0, canvas.width, canvas.height)

      particlesRef.current.forEach((p) => {
        // Update position
        p.x += p.vx
        p.y += p.vy

        // Bounce off edges
        if (p.x < 0 || p.x > canvas.width) p.vx *= -1
        if (p.y < 0 || p.y > canvas.height) p.vy *= -1

        // Mouse interaction
        const dx = mouseRef.current.x - p.x
        const dy = mouseRef.current.y - p.y
        const dist = Math.sqrt(dx * dx + dy * dy)
        if (dist < 150) {
          const force = (150 - dist) / 150
          p.vx -= (dx / dist) * force * 0.02
          p.vy -= (dy / dist) * force * 0.02
        }

        // Draw particle
        ctx.beginPath()
        ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2)
        ctx.fillStyle = p.color
        ctx.globalAlpha = p.opacity
        ctx.fill()
        ctx.globalAlpha = 1
      })

      // Draw connections
      particlesRef.current.forEach((p1, i) => {
        particlesRef.current.slice(i + 1).forEach((p2) => {
          const dx = p1.x - p2.x
          const dy = p1.y - p2.y
          const dist = Math.sqrt(dx * dx + dy * dy)
          if (dist < 120) {
            ctx.beginPath()
            ctx.moveTo(p1.x, p1.y)
            ctx.lineTo(p2.x, p2.y)
            ctx.strokeStyle = `rgba(99, 102, 241, ${0.15 * (1 - dist / 120)})`
            ctx.lineWidth = 1
            ctx.stroke()
          }
        })
      })

      animRef.current = requestAnimationFrame(tick)
    }

    tick()

    return () => {
      window.removeEventListener('resize', resize)
      window.removeEventListener('mousemove', handleMouseMove)
      cancelAnimationFrame(animRef.current)
    }
  }, [])

  return (
    <canvas
      ref={canvasRef}
      style={{
        position: 'absolute',
        inset: 0,
        background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%)',
      }}
    />
  )
}

// ---------- Floating Icons ----------
const FLOATING_ICONS = [
  { Icon: BellOutlined, color: '#f59e0b', delay: 0, x: '10%', y: '20%' },
  { Icon: SafetyOutlined, color: '#10b981', delay: 0.5, x: '85%', y: '15%' },
  { Icon: DashboardOutlined, color: '#6366f1', delay: 1, x: '15%', y: '75%' },
  { Icon: NotificationOutlined, color: '#ec4899', delay: 1.5, x: '80%', y: '80%' },
]

function FloatingIcons() {
  return (
    <>
      {FLOATING_ICONS.map(({ Icon, color, delay, x, y }, i) => (
        <motion.div
          key={i}
          style={{
            position: 'absolute',
            left: x,
            top: y,
            color,
            fontSize: 32,
            opacity: 0.3,
          }}
          initial={{ opacity: 0, scale: 0 }}
          animate={{
            opacity: 0.3,
            scale: 1,
            y: [0, -20, 0],
          }}
          transition={{
            opacity: { delay: delay + 0.5, duration: 0.5 },
            scale: { delay: delay + 0.5, duration: 0.5 },
            y: {
              delay: delay + 1,
              duration: 3,
              repeat: Infinity,
              ease: 'easeInOut',
            },
          }}
        >
          <Icon />
        </motion.div>
      ))}
    </>
  )
}

// ---------- Login Page ----------
export default function Login() {
  const { message } = App.useApp()
  const [loading, setLoading] = useState(false)
  const [rememberMe, setRememberMe] = useState(false)
  const { login, loginWithToken, token } = useAuth()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [oidcStatus, setOidcStatus] = useState<{ enabled: boolean; display_name: string } | null>(null)

  useEffect(() => {
    const oidcToken = searchParams.get('oidc_token')
    if (oidcToken) {
      loginWithToken(oidcToken)
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, loginWithToken, setSearchParams])

  useEffect(() => {
    if (token) navigate('/', { replace: true })
  }, [token, navigate])

  useEffect(() => {
    fetch('/api/v1/auth/oidc/status')
      .then((r) => r.json())
      .then((data) => setOidcStatus(data))
      .catch(() => setOidcStatus({ enabled: false, display_name: '' }))
  }, [])

  const onFinish = async (v: { username: string; password: string }) => {
    setLoading(true)
    try {
      await login(v.username, v.password)
      navigate('/')
    } catch (e: any) {
      message.error(e.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleOIDCLogin = () => {
    window.location.href = '/api/v1/auth/oidc/login'
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      <ParticleBackground />
      <FloatingIcons />

      {/* Glass Card */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, ease: 'easeOut' }}
        style={{
          width: '100%',
          maxWidth: 420,
          margin: '20px',
          position: 'relative',
          zIndex: 10,
        }}
      >
        <div
          style={{
            background: 'rgba(255, 255, 255, 0.95)',
            backdropFilter: 'blur(20px)',
            borderRadius: 24,
            padding: '48px 40px',
            boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25), 0 0 0 1px rgba(255, 255, 255, 0.1)',
          }}
        >
          {/* Logo */}
          <motion.div
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ delay: 0.2, duration: 0.4 }}
            style={{ textAlign: 'center', marginBottom: 32 }}
          >
            <div
              style={{
                width: 72,
                height: 72,
                margin: '0 auto 16px',
                background: 'linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%)',
                borderRadius: 20,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: '0 10px 25px -5px rgba(99, 102, 241, 0.4)',
              }}
            >
              <BellOutlined style={{ fontSize: 36, color: '#fff' }} />
            </div>
            <Title
              level={3}
              style={{
                margin: 0,
                fontSize: 28,
                fontWeight: 700,
                background: 'linear-gradient(135deg, #1e293b 0%, #475569 100%)',
                WebkitBackgroundClip: 'text',
                WebkitTextFillColor: 'transparent',
              }}
            >
              KK Alert
            </Title>
            <Text style={{ color: '#64748b', fontSize: 14 }}>
              智能告警管理平台
            </Text>
          </motion.div>

          {/* SSO Button */}
          <AnimatePresence>
            {oidcStatus?.enabled && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
              >
                <Button
                  block
                  size="large"
                  icon={<LoginOutlined />}
                  onClick={handleOIDCLogin}
                  style={{
                    height: 48,
                    borderRadius: 12,
                    background: '#f1f5f9',
                    border: '1px solid #e2e8f0',
                    color: '#475569',
                    fontSize: 15,
                    fontWeight: 500,
                    marginBottom: 16,
                  }}
                >
                  {oidcStatus.display_name || 'SSO'} 登录
                </Button>

                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 16,
                    margin: '20px 0',
                  }}
                >
                  <div style={{ flex: 1, height: 1, background: '#e2e8f0' }} />
                  <Text style={{ color: '#94a3b8', fontSize: 13 }}>或使用账号密码</Text>
                  <div style={{ flex: 1, height: 1, background: '#e2e8f0' }} />
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Login Form */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.3 }}
          >
            <Form onFinish={onFinish} layout="vertical" size="large">
              <Form.Item
                name="username"
                rules={[{ required: true, message: '请输入用户名' }]}
                style={{ marginBottom: 20 }}
              >
                <Input
                  prefix={
                    <UserOutlined style={{ color: '#94a3b8', fontSize: 16 }} />
                  }
                  placeholder="用户名"
                  autoComplete="username"
                  style={{
                    height: 48,
                    borderRadius: 12,
                    border: '1px solid #e2e8f0',
                    background: '#f8fafc',
                    fontSize: 15,
                  }}
                />
              </Form.Item>

              <Form.Item
                name="password"
                rules={[{ required: true, message: '请输入密码' }]}
                style={{ marginBottom: 16 }}
              >
                <Input.Password
                  prefix={
                    <LockOutlined style={{ color: '#94a3b8', fontSize: 16 }} />
                  }
                  placeholder="密码"
                  autoComplete="current-password"
                  style={{
                    height: 48,
                    borderRadius: 12,
                    border: '1px solid #e2e8f0',
                    background: '#f8fafc',
                    fontSize: 15,
                  }}
                />
              </Form.Item>

              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 24,
                }}
              >
                <Checkbox
                  checked={rememberMe}
                  onChange={(e) => setRememberMe(e.target.checked)}
                  style={{ color: '#64748b' }}
                >
                  记住我
                </Checkbox>
                <Link
                  to="/forgot-password"
                  style={{
                    color: '#6366f1',
                    fontSize: 14,
                    textDecoration: 'none',
                  }}
                >
                  忘记密码？
                </Link>
              </div>

              <Form.Item style={{ marginBottom: 0 }}>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={loading}
                  block
                  size="large"
                  style={{
                    height: 48,
                    borderRadius: 12,
                    fontSize: 16,
                    fontWeight: 600,
                    background: 'linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%)',
                    border: 'none',
                    boxShadow: '0 4px 14px 0 rgba(99, 102, 241, 0.39)',
                  }}
                >
                  {loading ? '登录中...' : '登 录'}
                </Button>
              </Form.Item>
            </Form>
          </motion.div>

          {/* Footer */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.5 }}
            style={{
              textAlign: 'center',
              marginTop: 32,
              paddingTop: 24,
              borderTop: '1px solid #f1f5f9',
            }}
          >
            <Text style={{ color: '#94a3b8', fontSize: 13 }}>
              © {new Date().getFullYear()} KK Alert · 系统运行部驱动
            </Text>
          </motion.div>
        </div>
      </motion.div>
    </div>
  )
}
