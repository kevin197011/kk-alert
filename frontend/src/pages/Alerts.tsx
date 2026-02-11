import { useEffect, useState } from 'react'
import { App, Table, Button, Select, Space, Modal, Card, Tag, Typography, Row, Col, Input, Tooltip, Drawer } from 'antd'
const { Search } = Input
import { motion } from 'framer-motion'
import { 
  ReloadOutlined, 
  EyeOutlined, 
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  BellOutlined,
  DashboardOutlined,
  NotificationOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  StopOutlined,
  UnorderedListOutlined,
  DownloadOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import { authHeaders } from '../auth'
import { 
  EmptyState, 
  ErrorState,
  SeverityBadge, 
  AlertStatusBadge,
  PageHeader,
  StatCard 
} from '../components/ui'

dayjs.extend(utc)
dayjs.extend(timezone)

const { Text } = Typography

/** Format ISO time string to Asia/Shanghai (UTC+8) for display */
function formatTimeShanghai(iso: string | null | undefined): string {
  if (!iso) return '–'
  const d = dayjs(iso)
  if (!d.isValid()) return iso
  return d.tz('Asia/Shanghai').format('YYYY-MM-DD HH:mm:ss')
}

/** Format duration in ms to human string (e.g. "2 小时 30 分", "已持续 15 分钟") */
function formatDuration(ms: number, prefix = ''): string {
  if (ms < 0 || !Number.isFinite(ms)) return '–'
  const totalSec = Math.floor(ms / 1000)
  const days = Math.floor(totalSec / 86400)
  const hours = Math.floor((totalSec % 86400) / 3600)
  const minutes = Math.floor((totalSec % 3600) / 60)
  const seconds = totalSec % 60
  const parts: string[] = []
  if (days > 0) parts.push(`${days} 天`)
  if (hours > 0) parts.push(`${hours} 小时`)
  if (minutes > 0) parts.push(`${minutes} 分`)
  if (seconds > 0 && parts.length === 0) parts.push(`${seconds} 秒`)
  else if (seconds > 0 && days === 0) parts.push(`${seconds} 秒`)
  const text = parts.length ? parts.join(' ') : '0 秒'
  return prefix ? `${prefix}${text}` : text
}

/** Compute impact duration for an alert: firing => now - firing_at, resolved => resolved_at - firing_at */
function getImpactDuration(record: Alert): { text: string; ms: number } {
  const firingAt = record.firing_at ? dayjs(record.firing_at).valueOf() : 0
  if (!firingAt || !dayjs(record.firing_at).isValid()) return { text: '–', ms: 0 }
  const endMs = record.status === 'resolved' && record.resolved_at && dayjs(record.resolved_at).isValid()
    ? dayjs(record.resolved_at).valueOf()
    : Date.now()
  const ms = endMs - firingAt
  const prefix = record.status === 'firing' ? '已持续 ' : '持续 '
  return { text: formatDuration(ms, prefix), ms }
}

type Alert = { 
  alert_id: string
  title: string
  severity: string
  status: string
  source_type: string
  source_id?: number
  created_at: string
  firing_at?: string
  resolved_at?: string | null
  notify_success_count?: number
  notify_fail_count?: number
}

type Stats = {
  total: number
  firing: number
  resolved: number
  notifyCount: number
}

type SilenceItem = {
  id: number
  alert_id: string
  silence_until: string
  created_at: string
  title?: string
}

const SILENCE_DURATIONS = [
  { label: '30 分钟', value: 30 },
  { label: '1 小时', value: 60 },
  { label: '4 小时', value: 240 },
  { label: '24 小时', value: 1440 },
  { label: '3 天', value: 4320 },
  { label: '7 天', value: 10080 },
]

export default function Alerts() {
  const { message } = App.useApp()
  const [list, setList] = useState<Alert[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [detail, setDetail] = useState<any>(null)
  const [severity, setSeverity] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [datasourceId, setDatasourceId] = useState<string | null>(null)
  const [alertIdSearch, setAlertIdSearch] = useState('')
  const [titleSearch, setTitleSearch] = useState('')
  const [datasources, setDatasources] = useState<{ id: number; name: string }[]>([])
  const [stats, setStats] = useState<Stats>({ total: 0, firing: 0, resolved: 0, notifyCount: 0 })
  const [silenceModal, setSilenceModal] = useState<Alert | null>(null)
  const [silenceDuration, setSilenceDuration] = useState(60)
  const [silenceSubmitting, setSilenceSubmitting] = useState(false)
  const [silencesDrawerOpen, setSilencesDrawerOpen] = useState(false)
  const [silencesList, setSilencesList] = useState<SilenceItem[]>([])
  const [silencesLoading, setSilencesLoading] = useState(false)
  const [exporting, setExporting] = useState(false)

  // Export current filtered alerts as Excel
  const exportExcel = async () => {
    setExporting(true)
    try {
      const params = new URLSearchParams()
      if (severity) params.set('severity', severity)
      if (status) params.set('status', status)
      if (datasourceId) params.set('datasource_id', datasourceId)
      if (alertIdSearch.trim()) params.set('alert_id', alertIdSearch.trim())
      if (titleSearch.trim()) params.set('title', titleSearch.trim())
      const res = await fetch(`/api/v1/alerts/export?${params}`, { headers: authHeaders() })
      if (!res.ok) throw new Error('export failed')
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      const disposition = res.headers.get('Content-Disposition')
      const match = disposition?.match(/filename=(.+)/)
      a.download = match?.[1] ?? 'alerts.xlsx'
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      message.success('导出成功')
    } catch {
      message.error('导出失败')
    } finally {
      setExporting(false)
    }
  }

  useEffect(() => {
    fetch('/api/v1/datasources', { headers: authHeaders() })
      .then((r) => (r.ok ? r.json() : []))
      .then((list) => setDatasources(Array.isArray(list) ? list : []))
      .catch(() => setDatasources([]))
  }, [])

  type LoadOverrides = {
    page?: number
    severity?: string | null
    status?: string | null
    datasourceId?: string | null
    alertIdSearch?: string
    titleSearch?: string
  }

  const load = async (silent = false, overrides?: LoadOverrides) => {
    if (!silent) {
      setLoading(true)
      setError(false)
    }
    const p = overrides?.page ?? page
    const sev = overrides !== undefined && 'severity' in overrides ? overrides.severity : severity
    const st = overrides !== undefined && 'status' in overrides ? overrides.status : status
    const dsId = overrides !== undefined && 'datasourceId' in overrides ? overrides.datasourceId : datasourceId
    const aId = overrides !== undefined && 'alertIdSearch' in overrides ? overrides.alertIdSearch : alertIdSearch
    const tit = overrides !== undefined && 'titleSearch' in overrides ? overrides.titleSearch : titleSearch
    try {
      const params = new URLSearchParams({ page: String(p), page_size: String(pageSize) })
      if (sev) params.set('severity', sev)
      if (st) params.set('status', st)
      if (dsId) params.set('datasource_id', dsId)
      if ((aId ?? '').trim()) params.set('alert_id', (aId ?? '').trim())
      if ((tit ?? '').trim()) params.set('title', (tit ?? '').trim())

      const res = await fetch(`/api/v1/alerts?${params}`, { headers: authHeaders() })
      if (!res.ok) throw new Error('Failed to fetch')
      const d = await res.json()
      setList(d.items || [])
      setTotal(d.total || 0)

      const listTotal = d.total || 0

      // Global total (no filters) for "总告警数" stat
      const fetchGlobalTotal = async () => {
        const r = await fetch('/api/v1/alerts?page=1&page_size=1', { headers: authHeaders() })
        if (!r.ok) return 0
        const data = await r.json()
        return data.total ?? 0
      }

      // Base params for stats (same filters as list, no pagination)
      const baseParams = new URLSearchParams({ page: '1', page_size: '1' })
      if (sev) baseParams.set('severity', sev)
      if (dsId) baseParams.set('datasource_id', dsId)
      if ((aId ?? '').trim()) baseParams.set('alert_id', (aId ?? '').trim())
      if ((tit ?? '').trim()) baseParams.set('title', (tit ?? '').trim())

      const fetchTotal = async (extra: Record<string, string>) => {
        const p = new URLSearchParams(baseParams)
        Object.entries(extra).forEach(([k, v]) => p.set(k, v))
        const r = await fetch(`/api/v1/alerts?${p}`, { headers: authHeaders() })
        if (!r.ok) return 0
        const data = await r.json()
        return data.total ?? 0
      }

      const fetchNotifyTotal = async () => {
        const r = await fetch('/api/v1/alerts/notify-total', { headers: authHeaders() })
        if (!r.ok) return 0
        const data = await r.json()
        return data.total ?? 0
      }

      const [totalAll, firingTotal, resolvedTotal, notifyTotal] = await Promise.all([
        fetchGlobalTotal(),
        st === 'firing' ? listTotal : fetchTotal({ status: 'firing' }),
        st === 'resolved' ? listTotal : fetchTotal({ status: 'resolved' }),
        fetchNotifyTotal(),
      ])

      setStats({
        total: totalAll,
        firing: firingTotal,
        resolved: resolvedTotal,
        notifyCount: notifyTotal,
      })
    } catch {
      if (!silent) setError(true)
    } finally {
      if (!silent) setLoading(false)
    }
  }

  useEffect(() => { load() }, [page, pageSize, severity, status, datasourceId])

  // Auto-refresh list and stats every 1 minute (silent, no loading spinner)
  useEffect(() => {
    const timer = setInterval(() => load(true), 60 * 1000)
    return () => clearInterval(timer)
  }, [page, pageSize, severity, status, datasourceId, alertIdSearch, titleSearch])

  const loadDetail = (id: string) => {
    fetch(`/api/v1/alerts/${id}`, { headers: authHeaders() })
      .then((r) => r.json())
      .then(setDetail)
  }

  const clearFilters = () => {
    setSeverity(null)
    setStatus(null)
    setDatasourceId(null)
    setAlertIdSearch('')
    setTitleSearch('')
    setPage(1)
    load(false, { page: 1, severity: null, status: null, datasourceId: null, alertIdSearch: '', titleSearch: '' })
  }

  const loadSilences = () => {
    setSilencesLoading(true)
    fetch('/api/v1/silences', { headers: authHeaders() })
      .then((r) => r.json())
      .then((d) => setSilencesList(d.items ?? []))
      .finally(() => setSilencesLoading(false))
  }

  const doSilence = () => {
    if (!silenceModal) return
    setSilenceSubmitting(true)
    fetch(`/api/v1/alerts/${encodeURIComponent(silenceModal.alert_id)}/silence`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ duration_minutes: silenceDuration }),
    })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed'))))
      .then(() => {
        message.success(`已静默该告警 ${SILENCE_DURATIONS.find((d) => d.value === silenceDuration)?.label ?? silenceDuration + ' 分钟'}`)
        setSilenceModal(null)
        load()
      })
      .catch(() => message.error('静默失败'))
      .finally(() => setSilenceSubmitting(false))
  }

  const cancelSilence = (alertId: string) => {
    fetch(`/api/v1/silences/${encodeURIComponent(alertId)}`, {
      method: 'DELETE',
      headers: authHeaders(),
    })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error('Failed'))))
      .then(() => {
        message.success('已取消静默')
        loadSilences()
      })
      .catch(() => message.error('取消静默失败'))
  }

  if (error) {
    return (
      <ErrorState 
        onRetry={load}
        title="加载告警失败"
        subTitle="无法获取告警列表，请检查网络连接后重试"
      />
    )
  }

  return (
    <div className="alerts-page">
      <PageHeader
        title="告警历史"
        subtitle="查看和管理所有系统告警"
        actions={
          <Space>
            <Button
              icon={<DownloadOutlined />}
              onClick={exportExcel}
              loading={exporting}
            >
              导出 Excel
            </Button>
            <Button
              icon={<UnorderedListOutlined />}
              onClick={() => {
                setSilencesDrawerOpen(true)
                loadSilences()
              }}
            >
              静默管理
            </Button>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={() => load()}
              loading={loading}
            >
              刷新
            </Button>
          </Space>
        }
      />

      <Row gutter={[24, 24]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="总告警数"
            value={stats.total}
            icon={<BellOutlined />}
            color="#1890ff"
            delay={0}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="告警中"
            value={stats.firing}
            icon={<ExclamationCircleOutlined />}
            color="#ff4d4f"
            change={{ value: 12, type: 'up' }}
            delay={0.1}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="已恢复"
            value={stats.resolved}
            icon={<CheckCircleOutlined />}
            color="#52c41a"
            change={{ value: 8, type: 'up' }}
            delay={0.2}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="通知数"
            value={stats.notifyCount}
            icon={<NotificationOutlined />}
            color="#722ed1"
            delay={0.3}
          />
        </Col>
      </Row>

      <Card variant="borderless" className="alerts-table-card">
        <Space style={{ marginBottom: 16 }} wrap className="filter-bar">
          <Search
            placeholder="按告警 ID 查询"
            allowClear
            value={alertIdSearch}
            onChange={(e) => {
              const v = e.target.value
              setAlertIdSearch(v)
              if (!v.trim()) { setPage(1); load(false, { alertIdSearch: '', page: 1 }) }
            }}
            onSearch={() => { setPage(1); load(false, { page: 1 }) }}
            style={{ width: 220 }}
            enterButton="查询"
          />
          <Search
            placeholder="按标题模糊查询"
            allowClear
            value={titleSearch}
            onChange={(e) => {
              const v = e.target.value
              setTitleSearch(v)
              if (!v.trim()) { setPage(1); load(false, { titleSearch: '', page: 1 }) }
            }}
            onSearch={() => { setPage(1); load(false, { page: 1 }) }}
            style={{ width: 220 }}
            enterButton="查询"
          />
          <Select
            placeholder="严重程度"
            allowClear
            style={{ width: 140 }}
            value={severity ?? undefined}
            onChange={(v) => {
              setPage(1)
              setSeverity(v ?? null)
              load(false, { severity: v ?? null, page: 1 })
            }}
            options={[
              { value: 'critical', label: '严重' },
              { value: 'warning', label: '警告' },
              { value: 'info', label: '提示' },
            ]}
          />
          <Select
            placeholder="告警状态"
            allowClear
            style={{ width: 140 }}
            value={status ?? undefined}
            onChange={(v) => {
              setPage(1)
              setStatus(v ?? null)
              load(false, { status: v ?? null, page: 1 })
            }}
            options={[
              { value: 'firing', label: '告警中' },
              { value: 'resolved', label: '已恢复' },
              { value: 'suppressed', label: '已静默' },
            ]}
          />
          <Select
            placeholder="数据源"
            allowClear
            style={{ width: 180 }}
            value={datasourceId ?? undefined}
            onChange={(v) => {
              setPage(1)
              setDatasourceId(v ?? null)
              load(false, { datasourceId: v ?? null, page: 1 })
            }}
            options={datasources.map((d) => ({ value: String(d.id), label: d.name || `#${d.id}` }))}
          />
          
          <Button onClick={clearFilters}>清除筛选</Button>
        </Space>

        <Table
          loading={loading}
          dataSource={list}
          rowKey="alert_id"
          locale={{
            emptyText: (
              <EmptyState
                type="empty"
                title="暂无告警数据"
                description="当前没有符合条件的告警记录"
                action={{ text: '清除筛选', onClick: clearFilters }}
              />
            )
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps || 20) },
          }}
          columns={[
            {
              title: '告警ID',
              dataIndex: 'alert_id',
              ellipsis: true,
              width: 260,
              render: (id) => (
                <Text code style={{ fontSize: 12 }}>{id}</Text>
              ),
            },
            {
              title: '标题',
              dataIndex: 'title',
              ellipsis: true,
              render: (title, record) => (
                <div>
                  <Text strong>{title}</Text>
                  <br />
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {record.source_type}
                  </Text>
                </div>
              ),
            },
            {
              title: '严重程度',
              dataIndex: 'severity',
              width: 110,
              render: (severity) => (
                <SeverityBadge severity={severity as any} />
              ),
            },
            {
              title: '状态',
              dataIndex: 'status',
              width: 110,
              render: (status) => (
                <AlertStatusBadge status={status as any} />
              ),
            },
            {
              title: '通知',
              key: 'notify',
              width: 120,
              render: (_, r) => {
                const ok = r.notify_success_count ?? 0
                const fail = r.notify_fail_count ?? 0
                if (ok === 0 && fail === 0) return <Text type="secondary">–</Text>
                return (
                  <Space size={4}>
                    <Tooltip title="成功">
                      <span style={{ color: '#52c41a' }}><CheckCircleFilled /> {ok}</span>
                    </Tooltip>
                    <Tooltip title="失败">
                      <span style={{ color: fail > 0 ? '#ff4d4f' : undefined }}><CloseCircleFilled /> {fail}</span>
                    </Tooltip>
                  </Space>
                )
              },
            },
            {
              title: '告警时间',
              dataIndex: 'firing_at',
              width: 170,
              render: (time: string) => (
                <Text type="secondary">{formatTimeShanghai(time)}</Text>
              ),
            },
            {
              title: '影响时长',
              key: 'impact_duration',
              width: 140,
              render: (_: unknown, record: Alert) => {
                const { text, ms } = getImpactDuration(record)
                const detail = ms >= 0 ? formatDuration(ms) : '–'
                return (
                  <Tooltip title={detail}>
                    <Text type="secondary">{text}</Text>
                  </Tooltip>
                )
              },
            },
            {
              title: '恢复时间',
              dataIndex: 'resolved_at',
              width: 170,
              render: (time: string | null) => (
                <Text type="secondary">{formatTimeShanghai(time)}</Text>
              ),
            },
            {
              title: '操作',
              width: 140,
              fixed: 'right',
              render: (_, r) => (
                <Space size="small">
                  <Button 
                    type="text" 
                    size="small"
                    icon={<StopOutlined />}
                    onClick={() => { setSilenceModal(r); setSilenceDuration(60) }}
                  >
                    静默
                  </Button>
                  <Button 
                    type="text" 
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => loadDetail(r.alert_id)}
                  >
                    详情
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      {detail && (
        <Modal 
          title={
            <Space>
              <BellOutlined />
              <span>告警详情</span>
              <Tag color="processing">{detail.alert?.alert_id ?? detail.alert_id}</Tag>
            </Space>
          }
          open 
          onCancel={() => setDetail(null)} 
          footer={null} 
          width={720}
          className="alert-detail-modal"
        >
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            {detail.sends?.length > 0 && (
              <Card size="small" title="通知记录" style={{ marginBottom: 16 }}>
                <Table
                  dataSource={detail.sends}
                  rowKey="id"
                  size="small"
                  pagination={false}
                  columns={[
                    { title: '渠道 ID', dataIndex: 'channel_id', width: 90, render: (v) => <Text code>{v}</Text> },
                    {
                      title: '结果',
                      dataIndex: 'success',
                      width: 80,
                      render: (success) =>
                        success ? (
                          <Tag color="success" icon={<CheckCircleFilled />}>成功</Tag>
                        ) : (
                          <Tag color="error" icon={<CloseCircleFilled />}>失败</Tag>
                        ),
                    },
                    {
                      title: '错误信息',
                      dataIndex: 'error',
                      ellipsis: true,
                      render: (err) => (err ? <Text type="danger">{err}</Text> : '–'),
                    },
                  ]}
                />
              </Card>
            )}
            <Card className="detail-card" variant="borderless" size="small" title="原始数据">
              <pre style={{ 
                whiteSpace: 'pre-wrap', 
                maxHeight: 400, 
                overflow: 'auto',
                background: '#f6f8fa',
                padding: 16,
                borderRadius: 8,
                fontSize: 12,
              }}>
                {JSON.stringify(detail, null, 2)}
              </pre>
            </Card>
          </motion.div>
        </Modal>
      )}

      <Modal
        title="静默告警"
        open={!!silenceModal}
        onCancel={() => setSilenceModal(null)}
        onOk={doSilence}
        okText="确定静默"
        confirmLoading={silenceSubmitting}
        destroyOnHidden
      >
        {silenceModal && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Text type="secondary">选择静默时长，该告警在此时长内将不会触发通知。</Text>
            <div>
              <Text strong>告警：</Text>
              <Text code style={{ fontSize: 12 }}>{silenceModal.alert_id}</Text>
              <br />
              <Text type="secondary">{silenceModal.title}</Text>
            </div>
            <div>
              <Text strong>静默时长：</Text>
              <Select
                value={silenceDuration}
                onChange={setSilenceDuration}
                options={SILENCE_DURATIONS}
                style={{ width: 160, marginLeft: 8 }}
              />
            </div>
          </Space>
        )}
      </Modal>

      <Drawer
        title="静默管理"
        placement="right"
        width={560}
        open={silencesDrawerOpen}
        onClose={() => setSilencesDrawerOpen(false)}
      >
        <Table
          loading={silencesLoading}
          dataSource={silencesList}
          rowKey="alert_id"
          size="small"
          pagination={false}
          locale={{ emptyText: '当前没有进行中的静默' }}
          columns={[
            {
              title: '告警 ID',
              dataIndex: 'alert_id',
              ellipsis: true,
              render: (id: string) => <Text code style={{ fontSize: 12 }}>{id}</Text>,
            },
            {
              title: '标题',
              dataIndex: 'title',
              ellipsis: true,
              render: (t: string) => t || '–',
            },
            {
              title: '静默至',
              dataIndex: 'silence_until',
              width: 170,
              render: (iso: string) => formatTimeShanghai(iso),
            },
            {
              title: '操作',
              width: 100,
              render: (_: unknown, row: SilenceItem) => (
                <Button type="link" size="small" danger onClick={() => cancelSilence(row.alert_id)}>
                  取消静默
                </Button>
              ),
            },
          ]}
        />
      </Drawer>
    </div>
  )
}
