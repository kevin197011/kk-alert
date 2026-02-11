import { useEffect, useState } from 'react'
import { Card, Row, Col } from 'antd'
import { motion } from 'framer-motion'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import {
  BellOutlined,
  FilterOutlined,
  DatabaseOutlined,
  NotificationOutlined,
  FileTextOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, StatCard } from '../components/ui'
import dayjs from 'dayjs'

type TrendPoint = { hour: string; count: number; label: string }

export default function Dashboard() {
  const [stats, setStats] = useState({
    alertTotal: 0,
    firing: 0,
    rules: 0,
    datasources: 0,
    channels: 0,
    templates: 0,
  })
  const [trendData, setTrendData] = useState<TrendPoint[]>([])
  const [trendLoading, setTrendLoading] = useState(true)

  useEffect(() => {
    const load = async () => {
      try {
        const res = await fetch('/api/v1/dashboard/stats', { headers: authHeaders() })
        if (!res.ok) return
        const d = await res.json()
        setStats({
          alertTotal: d.alert_total ?? 0,
          firing: d.firing ?? 0,
          rules: d.rules ?? 0,
          datasources: d.datasources ?? 0,
          channels: d.channels ?? 0,
          templates: d.templates ?? 0,
        })
      } catch {
        // ignore
      }
    }
    load()
  }, [])

  useEffect(() => {
    const fetchTrend = (showLoading = true) => {
      if (showLoading) setTrendLoading(true)
      fetch('/api/v1/reports/trend?hours=24', { headers: authHeaders() })
        .then((r) => r.json())
        .then((d) => {
          const raw = (d.data || []).map((x: { hour: string; count: number }) => ({
            hour: x.hour,
            count: Number(x.count) || 0,
            label: dayjs(x.hour).format('MM-DD HH:mm'),
          }))
          if (raw.length > 0) {
            raw[raw.length - 1].label = dayjs().format('MM-DD HH:mm') + '(当前)'
          }
          setTrendData(raw)
        })
        .catch(() => setTrendData([]))
        .finally(() => setTrendLoading(false))
    }
    fetchTrend(true)
    const timer = setInterval(() => fetchTrend(false), 60_000)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="dashboard-page">
      <PageHeader
        title="仪表盘"
        subtitle="系统概览与告警趋势"
      />

      <Row gutter={[24, 24]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={8}>
          <StatCard
            title="告警总数"
            value={stats.alertTotal}
            icon={<BellOutlined />}
            color="#1890ff"
            delay={0}
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard
            title="告警中"
            value={stats.firing}
            icon={<ExclamationCircleOutlined />}
            color="#ff4d4f"
            delay={0.05}
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard
            title="规则数"
            value={stats.rules}
            icon={<FilterOutlined />}
            color="#52c41a"
            delay={0.1}
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard
            title="数据源"
            value={stats.datasources}
            icon={<DatabaseOutlined />}
            color="#722ed1"
            delay={0.15}
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard
            title="通知渠道"
            value={stats.channels}
            icon={<NotificationOutlined />}
            color="#fa8c16"
            delay={0.2}
          />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard
            title="通知模板"
            value={stats.templates}
            icon={<FileTextOutlined />}
            color="#13c2c2"
            delay={0.25}
          />
        </Col>
      </Row>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.2 }}
      >
        <Card
          title="告警数量趋势（24 小时，截至当前时间）"
          extra={<span style={{ fontSize: 12, color: '#999' }}>每分钟更新</span>}
          loading={trendLoading}
        >
          <div style={{ width: '100%', height: 280 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trendData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis
                  dataKey="label"
                  tick={{ fontSize: 12 }}
                  interval="preserveStartEnd"
                  minTickGap={40}
                />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} width={32} />
                <Tooltip
                  formatter={(value: number) => [value, '告警数']}
                  labelFormatter={(label) => label}
                />
                <Area
                  type="monotone"
                  dataKey="count"
                  stroke="#1890ff"
                  fill="#1890ff"
                  fillOpacity={0.3}
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </motion.div>
    </div>
  )
}
