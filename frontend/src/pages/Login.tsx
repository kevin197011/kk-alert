import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { App, Form, Input, Button, Typography } from 'antd'
import { motion } from 'framer-motion'
import {
  UserOutlined,
  LockOutlined,
  BellOutlined,
} from '@ant-design/icons'
import { useAuth } from '../auth'

const { Title, Text } = Typography

// ---------- DevOps Network Canvas (light-theme adapted) ----------

interface Node {
  x: number
  y: number
  vx: number
  vy: number
  radius: number
  type: 'server' | 'database' | 'cloud' | 'monitor'
  pulse: number
  pulseSpeed: number
}

interface DataPacket {
  fromIdx: number
  toIdx: number
  progress: number
  speed: number
  color: string
}

const NODE_COUNT = 24
const CONNECT_DIST = 190

// Muted colors that work on a light background
const NODE_COLORS: Record<Node['type'], string> = {
  server: '23, 23, 23',     // --color-primary
  database: '64, 64, 64',   // --color-secondary
  cloud: '212, 175, 55',    // --color-cta (gold)
  monitor: '100, 116, 139', // slate
}

const PACKET_COLORS = ['#D4AF37', '#171717', '#64748b', '#94a3b8']

function NetworkCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const nodesRef = useRef<Node[]>([])
  const packetsRef = useRef<DataPacket[]>([])
  const animRef = useRef<number>(0)
  const sizeRef = useRef({ w: 0, h: 0 })

  const initNodes = useCallback((w: number, h: number) => {
    const types: Node['type'][] = ['server', 'database', 'cloud', 'monitor']
    const nodes: Node[] = []
    for (let i = 0; i < NODE_COUNT; i++) {
      nodes.push({
        x: Math.random() * w,
        y: Math.random() * h,
        vx: (Math.random() - 0.5) * 0.25,
        vy: (Math.random() - 0.5) * 0.25,
        radius: Math.random() * 2 + 2,
        type: types[Math.floor(Math.random() * types.length)],
        pulse: Math.random() * Math.PI * 2,
        pulseSpeed: 0.008 + Math.random() * 0.015,
      })
    }
    nodesRef.current = nodes
    packetsRef.current = []
  }, [])

  const drawNode = useCallback(
    (ctx: CanvasRenderingContext2D, node: Node, glow: number) => {
      const { x, y, type } = node
      const s = 3.5 + glow * 1.5
      ctx.save()

      // Subtle glow ring
      const alpha = 0.06 + glow * 0.1
      const ringR = s + 5 + glow * 3
      const grad = ctx.createRadialGradient(x, y, 0, x, y, ringR)
      const baseColor = NODE_COLORS[type]
      grad.addColorStop(0, `rgba(${baseColor}, ${alpha})`)
      grad.addColorStop(1, `rgba(${baseColor}, 0)`)
      ctx.fillStyle = grad
      ctx.beginPath()
      ctx.arc(x, y, ringR, 0, Math.PI * 2)
      ctx.fill()

      // Core shape
      ctx.fillStyle = `rgba(${baseColor}, ${0.25 + glow * 0.25})`
      ctx.beginPath()
      if (type === 'server') {
        ctx.rect(x - s / 2, y - s / 2, s, s)
      } else if (type === 'database') {
        ctx.arc(x, y, s / 2, 0, Math.PI * 2)
      } else if (type === 'cloud') {
        ctx.moveTo(x, y - s / 2)
        ctx.lineTo(x + s / 2, y)
        ctx.lineTo(x, y + s / 2)
        ctx.lineTo(x - s / 2, y)
        ctx.closePath()
      } else {
        ctx.moveTo(x, y - s / 2)
        ctx.lineTo(x + s / 2, y + s / 2)
        ctx.lineTo(x - s / 2, y + s / 2)
        ctx.closePath()
      }
      ctx.fill()
      ctx.restore()
    },
    [],
  )

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const resize = () => {
      const dpr = window.devicePixelRatio || 1
      const w = window.innerWidth
      const h = window.innerHeight
      canvas.width = w * dpr
      canvas.height = h * dpr
      canvas.style.width = `${w}px`
      canvas.style.height = `${h}px`
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
      sizeRef.current = { w, h }
      if (nodesRef.current.length === 0) initNodes(w, h)
    }
    resize()
    window.addEventListener('resize', resize)

    const tick = () => {
      const { w, h } = sizeRef.current
      ctx.clearRect(0, 0, w, h)
      const nodes = nodesRef.current
      const packets = packetsRef.current

      // Move nodes
      for (const n of nodes) {
        n.x += n.vx
        n.y += n.vy
        n.pulse += n.pulseSpeed
        if (n.x < 0 || n.x > w) n.vx *= -1
        if (n.y < 0 || n.y > h) n.vy *= -1
        n.x = Math.max(0, Math.min(w, n.x))
        n.y = Math.max(0, Math.min(h, n.y))
      }

      // Draw connections (light gray on light bg)
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const dx = nodes[i].x - nodes[j].x
          const dy = nodes[i].y - nodes[j].y
          const dist = Math.sqrt(dx * dx + dy * dy)
          if (dist < CONNECT_DIST) {
            const alpha = (1 - dist / CONNECT_DIST) * 0.08
            ctx.strokeStyle = `rgba(23, 23, 23, ${alpha})`
            ctx.lineWidth = 0.5
            ctx.beginPath()
            ctx.moveTo(nodes[i].x, nodes[i].y)
            ctx.lineTo(nodes[j].x, nodes[j].y)
            ctx.stroke()
          }
        }
      }

      // Spawn data packets
      if (Math.random() < 0.025 && packets.length < 10) {
        const from = Math.floor(Math.random() * nodes.length)
        let to = Math.floor(Math.random() * nodes.length)
        while (to === from) to = Math.floor(Math.random() * nodes.length)
        const dx = nodes[from].x - nodes[to].x
        const dy = nodes[from].y - nodes[to].y
        if (Math.sqrt(dx * dx + dy * dy) < CONNECT_DIST * 1.5) {
          packets.push({
            fromIdx: from,
            toIdx: to,
            progress: 0,
            speed: 0.004 + Math.random() * 0.008,
            color: PACKET_COLORS[Math.floor(Math.random() * PACKET_COLORS.length)],
          })
        }
      }

      // Draw & update packets
      for (let i = packets.length - 1; i >= 0; i--) {
        const p = packets[i]
        p.progress += p.speed
        if (p.progress >= 1) {
          packets.splice(i, 1)
          continue
        }
        const from = nodes[p.fromIdx]
        const to = nodes[p.toIdx]
        const px = from.x + (to.x - from.x) * p.progress
        const py = from.y + (to.y - from.y) * p.progress
        const alpha = p.progress < 0.1 ? p.progress / 0.1 : p.progress > 0.9 ? (1 - p.progress) / 0.1 : 1

        const r = parseInt(p.color.slice(1, 3), 16)
        const g = parseInt(p.color.slice(3, 5), 16)
        const b = parseInt(p.color.slice(5, 7), 16)

        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.6})`
        ctx.beginPath()
        ctx.arc(px, py, 2, 0, Math.PI * 2)
        ctx.fill()

        // Trail
        ctx.fillStyle = `rgba(${r}, ${g}, ${b}, ${alpha * 0.15})`
        const trail = Math.max(0, p.progress - 0.05)
        const tx = from.x + (to.x - from.x) * trail
        const ty = from.y + (to.y - from.y) * trail
        ctx.beginPath()
        ctx.arc(tx, ty, 1.5, 0, Math.PI * 2)
        ctx.fill()
      }

      // Draw nodes
      for (const n of nodes) {
        const glow = (Math.sin(n.pulse) + 1) / 2
        drawNode(ctx, n, glow)
      }

      animRef.current = requestAnimationFrame(tick)
    }

    animRef.current = requestAnimationFrame(tick)

    return () => {
      window.removeEventListener('resize', resize)
      cancelAnimationFrame(animRef.current)
    }
  }, [initNodes, drawNode])

  return <canvas ref={canvasRef} className="login-canvas" />
}

// ---------- Scrolling Log Lines ----------
const LOG_LINES = [
  '[scheduler] rule evaluation completed: 24 rules checked',
  '[monitor] prometheus scrape OK: 1247 series collected',
  '[engine] alert processed: severity=warning rule_id=7',
  '[sender] lark webhook delivered: 200 OK (142ms)',
  '[health] all datasources healthy: 5/5 online',
  '[scheduler] firing count updated: 12 critical, 3 warning',
  '[query] prometheus range query: 0.8s, 4096 samples',
  '[engine] alert resolved: rule_id=3 duration=2h15m',
  '[sender] telegram message sent: chat_id=ops-alerts',
  '[monitor] node_exporter up: 98.7% availability',
  '[scheduler] threshold match: cpu_usage > 90% on prod-api-3',
  '[engine] dedup: external_id matched, updating state',
  '[health] postgres connection pool: 12/50 active',
  '[sender] rate limiter: 18/20 tokens remaining',
  '[scheduler] next evaluation cycle in 30s',
]

function ScrollingLogs() {
  const [lines, setLines] = useState<string[]>([])

  useEffect(() => {
    const initial = Array.from({ length: 4 }, () =>
      LOG_LINES[Math.floor(Math.random() * LOG_LINES.length)]
    )
    setLines(initial)

    const timer = setInterval(() => {
      setLines((prev) => {
        const next = [...prev, LOG_LINES[Math.floor(Math.random() * LOG_LINES.length)]]
        return next.slice(-5)
      })
    }, 3000)

    return () => clearInterval(timer)
  }, [])

  return (
    <div className="login-logs">
      {lines.map((line, i) => (
        <motion.div
          key={`${i}-${line}`}
          initial={{ opacity: 0, x: -8 }}
          animate={{ opacity: i === lines.length - 1 ? 0.35 : 0.18, x: 0 }}
          transition={{ duration: 0.4 }}
          className="login-log-line"
        >
          <span className="login-log-time">
            {new Date().toLocaleTimeString('en-US', { hour12: false })}
          </span>
          {' '}{line}
        </motion.div>
      ))}
    </div>
  )
}

// ---------- Login Page ----------

export default function Login() {
  const { message } = App.useApp()
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  const onFinish = async (v: { username: string; password: string }) => {
    setLoading(true)
    try {
      await login(v.username, v.password)
      navigate('/')
    } catch (e: any) {
      message.error(e.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <NetworkCanvas />
      <ScrollingLogs />

      <motion.div
        className="login-panel"
        initial={{ opacity: 0, y: 24, scale: 0.97 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.5, ease: [0.25, 0.1, 0.25, 1] }}
      >
        {/* Brand */}
        <motion.div
          className="login-brand"
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.25, duration: 0.4 }}
        >
          <div className="login-logo">
            <BellOutlined />
          </div>
          <div>
            <Title level={3} className="login-title">
              KK Alert
            </Title>
            <Text className="login-subtitle">
              Intelligent Alert Management
            </Text>
          </div>
        </motion.div>

        {/* Divider */}
        <div className="login-divider" />

        {/* Form */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.4, duration: 0.4 }}
        >
          <Form onFinish={onFinish} layout="vertical" size="large" className="login-form">
            <Form.Item
              name="username"
              label="用户名"
              rules={[{ required: true, message: '请输入用户名' }]}
            >
              <Input
                prefix={<UserOutlined className="login-input-icon" />}
                placeholder="请输入用户名"
                autoComplete="username"
              />
            </Form.Item>

            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password
                prefix={<LockOutlined className="login-input-icon" />}
                placeholder="请输入密码"
                autoComplete="current-password"
              />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0, marginTop: 12 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                block
                className="login-submit-btn"
              >
                {loading ? '登录中...' : '登 录'}
              </Button>
            </Form.Item>
          </Form>
        </motion.div>

        <motion.div
          className="login-footer"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.7 }}
        >
          &copy; {new Date().getFullYear()} KK Alert &middot; 系统运行部驱动
        </motion.div>
      </motion.div>
    </div>
  )
}
