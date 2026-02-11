import { useEffect, useState, useCallback } from 'react'
import { App, Card, Button, DatePicker, Space, Table, Select, Row, Col, Tag, Empty, Tooltip } from 'antd'
import {
  FileTextOutlined,
  FileExcelOutlined,
  SearchOutlined,
  ReloadOutlined,
  DownloadOutlined,
  AlertOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  FilterOutlined,
} from '@ant-design/icons'
import { motion } from 'framer-motion'
import type { Dayjs } from 'dayjs'
import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import { authHeaders } from '../auth'
import {
  PageHeader,
  StatCard,
  SeverityBadge,
  AlertStatusBadge,
} from '../components/ui'

dayjs.extend(utc)
dayjs.extend(timezone)

const { RangePicker } = DatePicker

// Default: last 7 days
const defaultRange: [Dayjs, Dayjs] = [
  dayjs().subtract(7, 'day').startOf('day'),
  dayjs().endOf('day'),
]

interface PreviewAlert {
  alert_id: string
  title: string
  severity: string
  status: string
  firing_at: string
  resolved_at: string
  impact_duration: string
  value: string
  labels: string
  source_type: string
}

interface PreviewSummary {
  severity: Record<string, number>
  status: Record<string, number>
}

interface PreviewResponse {
  total: number
  page: number
  pageSize: number
  alerts: PreviewAlert[]
  summary: PreviewSummary
}

const severityLabel: Record<string, string> = {
  critical: '严重',
  warning: '警告',
  info: '提示',
}

const statusLabel: Record<string, string> = {
  firing: '告警中',
  resolved: '已恢复',
  suppressed: '已静默',
}

export default function Reports() {
  const { message } = App.useApp()
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null]>(defaultRange)
  const [exporting, setExporting] = useState(false)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [alerts, setAlerts] = useState<PreviewAlert[]>([])
  const [summary, setSummary] = useState<PreviewSummary>({ severity: {}, status: {} })
  const [filterStatus, setFilterStatus] = useState<string>('')
  const [filterSeverity, setFilterSeverity] = useState<string>('')

  const buildDateParams = useCallback(() => {
    const [from, to] = dateRange
    const start = (from || defaultRange[0]).startOf('day').toISOString()
    const end = (to || defaultRange[1]).endOf('day').toISOString()
    return { from: start, to: end }
  }, [dateRange])

  const fetchPreview = useCallback(async (p = page, ps = pageSize) => {
    setLoading(true)
    try {
      const { from, to } = buildDateParams()
      const params = new URLSearchParams({
        from, to,
        page: String(p),
        page_size: String(ps),
      })
      if (filterStatus) params.set('status', filterStatus)
      if (filterSeverity) params.set('severity', filterSeverity)

      const res = await fetch(`/api/v1/reports/preview?${params}`, { headers: authHeaders() })
      if (!res.ok) throw new Error('Failed to load preview')
      const data: PreviewResponse = await res.json()
      setTotal(data.total)
      setAlerts(data.alerts || [])
      setSummary(data.summary || { severity: {}, status: {} })
    } catch {
      message.error('加载预览数据失败')
      setAlerts([])
    } finally {
      setLoading(false)
    }
  }, [buildDateParams, filterStatus, filterSeverity, page, pageSize, message])

  // Fetch on mount & when filters change
  useEffect(() => {
    setPage(1)
    fetchPreview(1, pageSize)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dateRange, filterStatus, filterSeverity])

  const handleSearch = () => {
    setPage(1)
    fetchPreview(1, pageSize)
  }

  const handleTableChange = (pagination: { current?: number; pageSize?: number }) => {
    const p = pagination.current || 1
    const ps = pagination.pageSize || 20
    setPage(p)
    setPageSize(ps)
    fetchPreview(p, ps)
  }

  const buildExportUrl = (format: 'json' | 'xlsx') => {
    const { from, to } = buildDateParams()
    const base = format === 'xlsx' ? '/api/v1/reports/export?format=xlsx' : '/api/v1/reports/export'
    const sep = format === 'xlsx' ? '&' : '?'
    return `${base}${sep}from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
  }

  const exportAlerts = (format: 'json' | 'xlsx') => {
    setExporting(true)
    const url = buildExportUrl(format)
    fetch(url, { headers: authHeaders() })
      .then((r) => {
        if (!r.ok) throw new Error('Export failed')
        return format === 'xlsx' ? r.arrayBuffer() : r.json()
      })
      .then((data) => {
        const blob =
          format === 'xlsx'
            ? new Blob([data as ArrayBuffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
            : new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
        const dateStr = dayjs().format('YYYY-MM-DD')
        const a = document.createElement('a')
        a.href = URL.createObjectURL(blob)
        a.download = format === 'xlsx' ? `alerts-${dateStr}.xlsx` : `alerts-${dateStr}.json`
        a.click()
        URL.revokeObjectURL(a.href)
        message.success('导出成功')
      })
      .catch(() => message.error('导出失败'))
      .finally(() => setExporting(false))
  }

  // Parse labels JSON to display
  const parseLabels = (labelsStr: string): Record<string, string> => {
    if (!labelsStr) return {}
    try {
      return JSON.parse(labelsStr)
    } catch {
      return {}
    }
  }

  const totalAlerts = total
  const firingCount = summary.status?.firing || 0
  const resolvedCount = summary.status?.resolved || 0
  const criticalCount = summary.severity?.critical || 0

  const columns = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      width: 280,
      ellipsis: true,
      render: (text: string) => (
        <Tooltip title={text}>
          <span style={{ fontWeight: 500 }}>{text}</span>
        </Tooltip>
      ),
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (sev: string) => <SeverityBadge severity={sev} />,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <AlertStatusBadge status={s} />,
    },
    {
      title: '告警时间',
      dataIndex: 'firing_at',
      key: 'firing_at',
      width: 170,
      render: (t: string) => <span style={{ fontFamily: 'var(--font-heading)', fontSize: 12 }}>{t || '–'}</span>,
    },
    {
      title: '恢复时间',
      dataIndex: 'resolved_at',
      key: 'resolved_at',
      width: 170,
      render: (t: string) => <span style={{ fontFamily: 'var(--font-heading)', fontSize: 12 }}>{t || '–'}</span>,
    },
    {
      title: '影响时长',
      dataIndex: 'impact_duration',
      key: 'impact_duration',
      width: 130,
      render: (d: string) => (
        <Tag color={d && d !== '–' ? 'orange' : 'default'} style={{ borderRadius: 4 }}>
          {d || '–'}
        </Tag>
      ),
    },
    {
      title: '当前值',
      dataIndex: 'value',
      key: 'value',
      width: 100,
      render: (v: string) => (
        <span style={{ fontFamily: 'var(--font-heading)', fontSize: 12, color: v ? '#d48806' : '#999' }}>
          {v || '–'}
        </span>
      ),
    },
    {
      title: '标签',
      dataIndex: 'labels',
      key: 'labels',
      width: 240,
      ellipsis: true,
      render: (labelsStr: string) => {
        const labels = parseLabels(labelsStr)
        const entries = Object.entries(labels).slice(0, 3)
        if (entries.length === 0) return <span style={{ color: '#999' }}>–</span>
        return (
          <Tooltip title={Object.entries(labels).map(([k, v]) => `${k}=${v}`).join(', ')}>
            <Space size={4} wrap>
              {entries.map(([k, v]) => (
                <Tag key={k} style={{ fontSize: 11, borderRadius: 4, margin: 0 }}>
                  {k}={v}
                </Tag>
              ))}
              {Object.keys(labels).length > 3 && (
                <Tag style={{ fontSize: 11, borderRadius: 4, margin: 0 }}>+{Object.keys(labels).length - 3}</Tag>
              )}
            </Space>
          </Tooltip>
        )
      },
    },
  ]

  return (
    <div className="reports-page">
      <PageHeader
        title="报表中心"
        subtitle="按日期区间查询和导出告警数据"
        actions={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={handleSearch}
              loading={loading}
            >
              刷新
            </Button>
          </Space>
        }
      />

      {/* Summary Stat Cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <StatCard
            title="告警总数"
            value={totalAlerts}
            icon={<AlertOutlined />}
            color="#1890ff"
            delay={0}
          />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard
            title="告警中"
            value={firingCount}
            icon={<ExclamationCircleOutlined />}
            color="#ff4d4f"
            delay={0.05}
          />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard
            title="已恢复"
            value={resolvedCount}
            icon={<CheckCircleOutlined />}
            color="#52c41a"
            delay={0.1}
          />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard
            title="严重告警"
            value={criticalCount}
            icon={<CloseCircleOutlined />}
            color="#cf1322"
            delay={0.15}
          />
        </Col>
      </Row>

      {/* Filter & Export Bar */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.2 }}
      >
        <Card
          className="reports-filter-card"
          style={{ borderRadius: 12, boxShadow: '0 2px 8px rgba(0, 0, 0, 0.04)', marginBottom: 24 }}
          styles={{ body: { padding: '20px 24px' } }}
        >
          <Row gutter={[16, 16]} align="middle">
            <Col flex="auto">
              <Space wrap size="middle">
                <div>
                  <span style={{ marginRight: 8, color: 'var(--color-secondary)', fontSize: 13 }}>
                    <FilterOutlined style={{ marginRight: 4 }} />日期区间
                  </span>
                  <RangePicker
                    value={dateRange}
                    onChange={(dates) => {
                      const next = dates || [null, null]
                      setDateRange([
                        next[0] ? next[0].startOf('day') : null,
                        next[1] ? next[1].endOf('day') : null,
                      ])
                    }}
                    allowClear={false}
                    style={{ minWidth: 260 }}
                  />
                </div>
                <div>
                  <span style={{ marginRight: 8, color: 'var(--color-secondary)', fontSize: 13 }}>状态</span>
                  <Select
                    value={filterStatus || undefined}
                    placeholder="全部"
                    allowClear
                    onChange={(v) => setFilterStatus(v || '')}
                    style={{ width: 120 }}
                    options={[
                      { value: 'firing', label: '告警中' },
                      { value: 'resolved', label: '已恢复' },
                      { value: 'suppressed', label: '已静默' },
                    ]}
                  />
                </div>
                <div>
                  <span style={{ marginRight: 8, color: 'var(--color-secondary)', fontSize: 13 }}>严重程度</span>
                  <Select
                    value={filterSeverity || undefined}
                    placeholder="全部"
                    allowClear
                    onChange={(v) => setFilterSeverity(v || '')}
                    style={{ width: 120 }}
                    options={[
                      { value: 'critical', label: '严重' },
                      { value: 'warning', label: '警告' },
                      { value: 'info', label: '提示' },
                    ]}
                  />
                </div>
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={handleSearch}
                  loading={loading}
                >
                  查询
                </Button>
              </Space>
            </Col>
            <Col>
              <Space>
                <Tooltip title="导出 Excel 文件">
                  <Button
                    icon={<FileExcelOutlined />}
                    onClick={() => exportAlerts('xlsx')}
                    loading={exporting}
                    style={{ borderColor: '#52c41a', color: '#52c41a' }}
                  >
                    导出 Excel
                  </Button>
                </Tooltip>
                <Tooltip title="导出 JSON 文件">
                  <Button
                    icon={<DownloadOutlined />}
                    onClick={() => exportAlerts('json')}
                    loading={exporting}
                  >
                    导出 JSON
                  </Button>
                </Tooltip>
              </Space>
            </Col>
          </Row>
        </Card>
      </motion.div>

      {/* Severity Breakdown Tags */}
      {(summary.severity && Object.keys(summary.severity).length > 0) && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.3, delay: 0.25 }}
          style={{ marginBottom: 16 }}
        >
          <Space size={8}>
            <span style={{ color: 'var(--color-secondary)', fontSize: 13 }}>严重程度分布：</span>
            {Object.entries(summary.severity)
              .sort(([a], [b]) => {
                const order = ['critical', 'warning', 'info']
                return order.indexOf(a) - order.indexOf(b)
              })
              .map(([sev, count]) => {
                const colorMap: Record<string, string> = {
                  critical: '#cf1322',
                  warning: '#d48806',
                  info: '#096dd9',
                }
                return (
                  <Tag
                    key={sev}
                    color={colorMap[sev] || '#999'}
                    style={{ borderRadius: 12, padding: '2px 12px', fontWeight: 500 }}
                  >
                    {severityLabel[sev] || sev} {count}
                  </Tag>
                )
              })}
          </Space>
          {summary.status && Object.keys(summary.status).length > 0 && (
            <Space size={8} style={{ marginLeft: 24 }}>
              <span style={{ color: 'var(--color-secondary)', fontSize: 13 }}>状态分布：</span>
              {Object.entries(summary.status).map(([st, count]) => {
                const colorMap: Record<string, string> = {
                  firing: 'red',
                  resolved: 'green',
                  suppressed: 'orange',
                }
                return (
                  <Tag
                    key={st}
                    color={colorMap[st] || 'default'}
                    style={{ borderRadius: 12, padding: '2px 12px', fontWeight: 500 }}
                  >
                    {statusLabel[st] || st} {count}
                  </Tag>
                )
              })}
            </Space>
          )}
        </motion.div>
      )}

      {/* Alerts Data Table */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.3 }}
      >
        <Card
          className="reports-table-card"
          title={
            <Space>
              <InfoCircleOutlined />
              <span>告警数据预览</span>
              {total > 0 && (
                <Tag style={{ borderRadius: 10, marginLeft: 8, fontWeight: 600 }}>{total} 条记录</Tag>
              )}
            </Space>
          }
          style={{ borderRadius: 12, boxShadow: '0 2px 8px rgba(0, 0, 0, 0.04)' }}
          styles={{ body: { padding: '12px 20px 20px' } }}
        >
          <Table
            dataSource={alerts}
            columns={columns}
            rowKey="alert_id"
            loading={loading}
            size="middle"
            scroll={{ x: 1300 }}
            pagination={{
              current: page,
              pageSize: pageSize,
              total: total,
              showSizeChanger: true,
              showQuickJumper: true,
              pageSizeOptions: ['10', '20', '50', '100'],
              showTotal: (t, range) => `第 ${range[0]}-${range[1]} 条，共 ${t} 条`,
            }}
            onChange={(pag) => handleTableChange(pag)}
            locale={{
              emptyText: (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="选定时间范围内暂无告警数据"
                />
              ),
            }}
          />
        </Card>
      </motion.div>

      {/* Footer Hint */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.5 }}
        style={{
          marginTop: 16,
          padding: '12px 16px',
          background: '#fafafa',
          borderRadius: 8,
          fontSize: 12,
          color: 'var(--color-secondary)',
          lineHeight: 1.8,
        }}
      >
        <FileTextOutlined style={{ marginRight: 6 }} />
        数据说明：按告警发生时间（firing_at）筛选，起始日 00:00 至结束日 24:00。导出支持 Excel 和 JSON 格式，最多导出 10,000 条记录。
      </motion.div>
    </div>
  )
}
