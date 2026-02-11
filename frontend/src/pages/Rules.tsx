import { useEffect, useState, useRef } from 'react'
import { App, Table, Button, Space, Modal, Form, Input, InputNumber, Select, Switch, Upload, Collapse, Card, Badge, Tag, Typography, Tooltip, Row, Col, Divider } from 'antd'
import { motion } from 'framer-motion'
import {
  PlusOutlined,
  ExportOutlined,
  ImportOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ThunderboltOutlined,
  ExperimentOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons'
import { authHeaders } from '../auth'
import { PageHeader, StatusTag, EmptyState, StatCard } from '../components/ui'

type Rule = {
  id: number
  name: string
  enabled: boolean
  priority: number
  datasource_ids?: string
  channel_ids?: string
  jira_enabled?: boolean
  jira_after_n?: number
  jira_config?: string
  check_interval?: string
  last_run_at?: string | null
  query_language?: string
  query_expression?: string
  match_labels?: string
  match_severity?: string
  thresholds?: string
}

type DatasourceOption = { id: number; name: string; type?: string }
type ChannelOption = { id: number; name: string; type?: string }
type TemplateOption = { id: number; name: string; channel_type?: string }

function parseIds(value: string | number[] | undefined): number[] {
  if (value == null) return []
  if (Array.isArray(value)) return value.filter((x): x is number => typeof x === 'number')
  if (typeof value !== 'string') return []
  try {
    const a = JSON.parse(value)
    return Array.isArray(a) ? a.filter((x): x is number => typeof x === 'number') : []
  } catch {
    return []
  }
}

const QUERY_LANG_OPTIONS = [
  { value: '', label: '无' },
  { value: 'promql', label: 'PromQL (Prometheus)' },
  { value: 'elasticsearch_sql', label: 'ES SQL (Elasticsearch)' },
  { value: 'sql', label: 'SQL (Doris)' },
]

function formatLastRunExact(lastRunAt: string | null | undefined): string {
  if (!lastRunAt) return '—'
  const d = new Date(lastRunAt)
  if (Number.isNaN(d.getTime())) return '—'
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  const sec = String(d.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${day} ${h}:${min}:${sec}`
}

function formatLastRunRelative(lastRunAt: string | null | undefined): string {
  if (!lastRunAt) return ''
  const d = new Date(lastRunAt)
  if (Number.isNaN(d.getTime())) return ''
  const diffMs = Date.now() - d.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  if (diffSec < 60) return '刚刚'
  if (diffMin < 60) return `${diffMin} 分钟前`
  if (diffHour < 24) return `${diffHour} 小时前`
  return `${Math.floor(diffHour / 24)} 天前`
}

export default function Rules() {
  const { message } = App.useApp()
  const [list, setList] = useState<Rule[]>([])
  const [firingCounts, setFiringCounts] = useState<Record<string, { total: number; critical: number; warning: number; info: number }>>({})
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState<boolean | { id: number }>(false)
  const [datasources, setDatasources] = useState<DatasourceOption[]>([])
  const [channels, setChannels] = useState<ChannelOption[]>([])
  const [templates, setTemplates] = useState<TemplateOption[]>([])
  const [form] = Form.useForm()
  const queryLang = Form.useWatch('query_language', form) ?? ''
  const [testModalOpen, setTestModalOpen] = useState(false)
  const [testLoading, setTestLoading] = useState(false)
  const [testResult, setTestResult] = useState<any>(null)
  const [testRuleName, setTestRuleName] = useState<string | null>(null)
  const [testingRuleId, setTestingRuleId] = useState<number | null>(null)
  const [togglingId, setTogglingId] = useState<number | null>(null)
  const [triggeringId, setTriggeringId] = useState<number | null>(null)
  const [nameSearch, setNameSearch] = useState('')

  const load = (nameFilter?: string) => {
    setLoading(true)
    const name = nameFilter !== undefined ? nameFilter : nameSearch
    const params = new URLSearchParams()
    if (name.trim()) params.set('name', name.trim())
    const qs = params.toString()
    fetch(`/api/v1/rules${qs ? `?${qs}` : ''}`, { headers: authHeaders() })
      .then((r) => r.json())
      .then((data) => {
        const rules = Array.isArray(data) ? data : (data.rules || [])
        setList(rules)
        setFiringCounts(data.firing_counts ?? {})
      })
      .finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  useEffect(() => {
    if (!modalOpen) return
    Promise.all([
      fetch('/api/v1/datasources', { headers: authHeaders() }).then((r) => r.json()),
      fetch('/api/v1/channels', { headers: authHeaders() }).then((r) => r.json()),
      fetch('/api/v1/templates', { headers: authHeaders() }).then((r) => r.json()),
    ]).then(([ds, ch, tpl]) => {
      const templates = Array.isArray(tpl) ? tpl : []
      setDatasources(Array.isArray(ds) ? ds : [])
      setChannels(Array.isArray(ch) ? ch : [])
      setTemplates(templates)
      const defaultT = templates.find((x: { is_default?: boolean }) => x.is_default)
      if (defaultT) {
        const isAdd = modalOpen === true
        const current = form.getFieldValue('template_id')
        if (isAdd || current === undefined || current === null || current === '') {
          form.setFieldValue('template_id', defaultT.id)
        }
      }
    })
  }, [modalOpen])

  const onFinish = async (v: any) => {
    const id = modalOpen && typeof modalOpen === 'object' && 'id' in modalOpen ? (modalOpen as any).id : null
    const payload = { ...v }
    payload.datasource_ids = Array.isArray(v.datasource_ids) ? JSON.stringify(v.datasource_ids) : (v.datasource_ids ?? '[]')
    payload.channel_ids = Array.isArray(v.channel_ids) ? JSON.stringify(v.channel_ids) : (v.channel_ids ?? '[]')
    payload.match_severity = Array.isArray(v.match_severity) ? v.match_severity.join(',') : (v.match_severity ?? '')
    // Ensure template_id is sent as number so backend persists it (string would be ignored by *uint)
    payload.template_id = (v.template_id !== undefined && v.template_id !== null && v.template_id !== '') ? Number(v.template_id) : null
    if (v.jira_enabled && (v.jira_base_url || v.jira_project)) {
      payload.jira_config = JSON.stringify({
        base_url: v.jira_base_url || '',
        email: v.jira_email || '',
        token: v.jira_token || '',
        project: v.jira_project || '',
        issue_type: v.jira_issue_type || 'Task',
      })
    }
    // Serialize multi-level thresholds to JSON string
    if (Array.isArray(v.thresholds) && v.thresholds.length > 0) {
      const cleaned = v.thresholds.filter((t: any) => t && t.value !== undefined && t.value !== null)
      payload.thresholds = cleaned.length > 0 ? JSON.stringify(cleaned) : ''
    } else {
      payload.thresholds = ''
    }
    delete payload.jira_base_url
    delete payload.jira_email
    delete payload.jira_token
    delete payload.jira_project
    delete payload.jira_issue_type
    const url = id ? `/api/v1/rules/${id}` : '/api/v1/rules'
    const res = await fetch(url, { method: id ? 'PUT' : 'POST', headers: authHeaders(), body: JSON.stringify(payload) })
    if (!res.ok) {
      message.error((await res.json()).error || '保存失败')
      return
    }
    message.success('保存成功')
    setModalOpen(false)
    form.resetFields()
    load()
  }

  const buildTestMatchBody = (r: {
    datasource_ids?: string
    query_language?: string
    query_expression?: string
    match_labels?: string
    match_severity?: string
    thresholds?: string | unknown[]
  }) => {
    let thresholdsStr = ''
    if (typeof r.thresholds === 'string') {
      thresholdsStr = r.thresholds
    } else if (Array.isArray(r.thresholds) && r.thresholds.length > 0) {
      const cleaned = r.thresholds.filter((t: any) => t && t.value !== undefined && t.value !== null)
      thresholdsStr = cleaned.length > 0 ? JSON.stringify(cleaned) : ''
    }
    return {
      datasource_ids: typeof r.datasource_ids === 'string' ? r.datasource_ids : (Array.isArray(r.datasource_ids) ? JSON.stringify(r.datasource_ids) : '[]'),
      query_language: r.query_language || '',
      query_expression: r.query_expression || '',
      match_labels: r.match_labels || '{}',
      match_severity: Array.isArray(r.match_severity) ? (r.match_severity as string[]).join(',') : (r.match_severity ?? ''),
      thresholds: thresholdsStr,
    }
  }

  const testMatch = async () => {
    const values = form.getFieldsValue()
    if (!values.query_language && !values.match_labels && !values.match_severity && !values.datasource_ids) {
      message.warning('请至少配置一项匹配条件')
      return
    }

    setTestRuleName(null)
    setTestLoading(true)
    try {
      const body = buildTestMatchBody({
        datasource_ids: values.datasource_ids,
        query_language: values.query_language,
        query_expression: values.query_expression,
        match_labels: values.match_labels,
        match_severity: values.match_severity,
        thresholds: values.thresholds,
      })
      const res = await fetch('/api/v1/rules/test-match', {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (res.ok) {
        setTestResult(data)
        setTestModalOpen(true)
        if (data.matched) {
          message.success(`匹配成功！在 ${data.total_alerts} 条告警中匹配到 ${data.matched_alerts.length} 条`)
        } else {
          message.warning(data.message || '未匹配到告警')
        }
      } else {
        message.error(data.error || '测试失败')
      }
    } catch (e) {
      message.error('测试请求失败')
    } finally {
      setTestLoading(false)
    }
  }

  /** Test match for a single rule from the list (row action). */
  const testRuleMatch = async (r: Rule) => {
    if (!r.query_language && !r.match_labels && !r.match_severity && !r.datasource_ids) {
      message.warning('该规则未配置查询或匹配条件，无法测试')
      return
    }
    setTestRuleName(r.name)
    setTestingRuleId(r.id)
    try {
      const body = buildTestMatchBody(r)
      const res = await fetch('/api/v1/rules/test-match', {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (res.ok) {
        setTestResult(data)
        setTestModalOpen(true)
        if (data.matched) {
          message.success(`「${r.name}」匹配 ${data.matched_alerts?.length ?? 0} 条告警`)
        } else {
          message.info(data.message || `「${r.name}」未匹配到告警`)
        }
      } else {
        message.error(data.error || '测试失败')
      }
    } catch (e) {
      message.error('测试请求失败')
    } finally {
      setTestingRuleId(null)
    }
  }

  const batch = (action: 'enable' | 'disable' | 'delete', ids: number[]) => {
    if (!ids.length) { message.info('请先选择规则'); return }
    fetch('/api/v1/rules/batch', { method: 'POST', headers: authHeaders(), body: JSON.stringify({ ids, action }) })
      .then((r) => r.json())
      .then((d) => message.success(`成功: ${d.success}, 失败: ${d.failed}`))
      .then(load)
  }

  const toggleEnabled = (r: Rule) => {
    const action = r.enabled ? 'disable' : 'enable'
    setTogglingId(r.id)
    fetch('/api/v1/rules/batch', { method: 'POST', headers: authHeaders(), body: JSON.stringify({ ids: [r.id], action }) })
      .then((res) => res.json())
      .then((d) => {
        if (d.failed != null && d.failed > 0) {
          message.error('切换失败')
          return
        }
        message.success(r.enabled ? '已停用' : '已启用')
        setList((prev) => prev.map((x) => (x.id === r.id ? { ...x, enabled: !r.enabled } : x)))
      })
      .catch(() => message.error('切换失败'))
      .finally(() => setTogglingId(null))
  }

  // Trigger a rule to execute immediately
  const triggerRule = (r: Rule) => {
    setTriggeringId(r.id)
    fetch(`/api/v1/rules/${r.id}/trigger`, { method: 'POST', headers: authHeaders() })
      .then((res) => res.json())
      .then((d) => {
        if (d.ok) {
          message.success(`规则「${r.name}」已触发执行`)
        } else {
          message.error(d.error || '触发失败')
        }
      })
      .catch(() => message.error('触发失败'))
      .finally(() => setTriggeringId(null))
  }

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [importMode, setImportMode] = useState<'add' | 'overwrite'>('add')
  const [importModalOpen, setImportModalOpen] = useState(false)
  const importFileRef = useRef<File | null>(null)

  const exportRules = () => {
    const body = selectedRowKeys.length > 0 ? { ids: selectedRowKeys } : {}
    fetch('/api/v1/rules/export', { method: 'POST', headers: authHeaders(), body: JSON.stringify(body) })
      .then((r) => r.json())
      .then((d) => {
        const blob = new Blob([JSON.stringify(d, null, 2)], { type: 'application/json' })
        const a = document.createElement('a')
        a.href = URL.createObjectURL(blob)
        a.download = `rules-export-${Date.now()}.json`
        a.click()
        URL.revokeObjectURL(a.href)
        message.success('导出成功')
      })
      .catch(() => message.error('导出失败'))
  }

  const doImport = () => {
    if (!importFileRef.current) {
      message.warning('请先选择文件')
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      try {
        const json = JSON.parse(reader.result as string)
        const rules = json.rules || (Array.isArray(json) ? json : [])
        if (!rules.length) {
          message.error('文件中没有规则')
          return
        }
        fetch('/api/v1/rules/import', {
          method: 'POST',
          headers: authHeaders(),
          body: JSON.stringify({ rules, mode: importMode }),
        })
          .then(async (r) => {
            const d = await r.json().catch(() => ({}))
            if (!r.ok) {
              message.error(d?.error || '导入失败')
              return
            }
            const imported = d.imported ?? 0
            const failed = d.failed ?? 0
            message.success(`导入成功: ${imported}, 失败: ${failed}`)
            setImportModalOpen(false)
            importFileRef.current = null
            load()
          })
          .catch(() => message.error('导入失败'))
      } catch (e) {
        message.error('无效的 JSON 文件')
      }
    }
    reader.readAsText(importFileRef.current)
  }

  const enabledCount = list.filter(r => r.enabled).length
  const disabledCount = list.length - enabledCount

  return (
    <div className="rules-page">
      <PageHeader
        title="规则管理"
        subtitle="配置告警路由规则和分发策略"
        actions={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setModalOpen(true); form.resetFields() }}
            size="large"
          >
            新建规则
          </Button>
        }
      />

      <Row gutter={[24, 24]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="总规则数" value={list.length} icon={<ThunderboltOutlined />} color="#1890ff" delay={0} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="已启用" value={enabledCount} icon={<PlayCircleOutlined />} color="#52c41a" delay={0.1} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="已停用" value={disabledCount} icon={<PauseCircleOutlined />} color="#faad14" delay={0.2} />
        </Col>
      </Row>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
      <Card variant="borderless" className="rules-table-card">
        <Space style={{ marginBottom: 16 }} wrap>
          <Input.Search
            placeholder="按规则名称模糊查询"
            allowClear
            value={nameSearch}
            onChange={(e) => {
              const v = e.target.value
              setNameSearch(v)
              if (!v.trim()) load('')
            }}
            onSearch={() => load()}
            style={{ width: 260 }}
            enterButton="查询"
          />
          <Button icon={<ExportOutlined />} onClick={exportRules}>导出</Button>
          <Button icon={<ImportOutlined />} onClick={() => setImportModalOpen(true)}>导入</Button>
          <Button icon={<PlayCircleOutlined />} onClick={() => batch('enable', selectedRowKeys as number[])}>批量启用</Button>
          <Button icon={<PauseCircleOutlined />} onClick={() => batch('disable', selectedRowKeys as number[])}>批量停用</Button>
          <Button icon={<DeleteOutlined />} danger onClick={() => batch('delete', selectedRowKeys as number[])}>批量删除</Button>
        </Space>
        <Table
          loading={loading}
          dataSource={list}
          rowKey="id"
          rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
          locale={{
            emptyText: <EmptyState type="empty" title="暂无规则" description="点击右上角新建规则按钮开始创建" />
          }}
          columns={[
            {
              title: 'ID',
              dataIndex: 'id',
              width: 70,
              render: (id) => <Tag>#{id}</Tag>
            },
            { title: '规则名称', dataIndex: 'name' },
            {
              title: '状态',
              dataIndex: 'enabled',
              width: 100,
              render: (v: boolean) => <StatusTag status={v ? 'success' : 'default'} text={v ? '已启用' : '已停用'} />
            },
            {
              title: '告警状态',
              key: 'firing',
              width: 180,
              render: (_: unknown, r: Rule) => {
                const sc = firingCounts[String(r.id)]
                const total = sc?.total ?? 0
                if (total === 0) return <Tag color="green">正常</Tag>
                // If there are severity breakdowns, show per-level tags
                const hasSeverityBreakdown = (sc?.critical ?? 0) + (sc?.warning ?? 0) + (sc?.info ?? 0) > 0
                if (!hasSeverityBreakdown) {
                  return <Tag color="red">告警中 ({total})</Tag>
                }
                return (
                  <span style={{ display: 'inline-flex', gap: 2, flexWrap: 'wrap' }}>
                    {(sc?.critical ?? 0) > 0 && <Tag color="red">严重 {sc!.critical}</Tag>}
                    {(sc?.warning ?? 0) > 0 && <Tag color="orange">警告 {sc!.warning}</Tag>}
                    {(sc?.info ?? 0) > 0 && <Tag color="blue">提示 {sc!.info}</Tag>}
                  </span>
                )
              },
            },
            {
              title: '优先级',
              dataIndex: 'priority',
              width: 90,
              render: (p) => <Badge count={p} style={{ backgroundColor: p > 0 ? '#1890ff' : '#8c8c8c' }} />
            },
            {
              title: '执行状态',
              key: 'last_run_at',
              width: 200,
              render: (_: unknown, r: Rule) => {
                const exact = formatLastRunExact(r.last_run_at)
                const relative = formatLastRunRelative(r.last_run_at)
                return (
                  <Tooltip title={relative ? `${relative}（${exact}）` : exact !== '—' ? exact : undefined}>
                    <span style={{ color: '#666', fontSize: 13 }}>
                      {r.check_interval ? `间隔 ${r.check_interval}` : ''}
                      {r.check_interval && r.last_run_at ? ' · ' : ''}
                      {exact}
                    </span>
                  </Tooltip>
                )
              }
            },
            {
              title: '操作',
              width: 340,
              render: (_, r) => (
                <Space wrap>
                  <Tooltip title={r.enabled ? '点击停用' : '点击启用'}>
                    <Switch
                      checked={r.enabled}
                      loading={togglingId === r.id}
                      disabled={togglingId !== null && togglingId !== r.id}
                      onChange={() => toggleEnabled(r)}
                      checkedChildren="启用"
                      unCheckedChildren="停用"
                    />
                  </Tooltip>
                  <Tooltip title="立即执行一次规则检测">
                    <Button
                      type="text"
                      size="small"
                      icon={<ThunderboltOutlined />}
                      loading={triggeringId === r.id}
                      disabled={!r.enabled || !!(triggeringId !== null && triggeringId !== r.id)}
                      onClick={() => triggerRule(r)}
                    >
                      触发
                    </Button>
                  </Tooltip>
                  <Tooltip title="使用当前规则配置测试匹配告警">
                    <Button
                      type="text"
                      size="small"
                      icon={<ExperimentOutlined />}
                      loading={testingRuleId === r.id}
                      disabled={!!(testingRuleId !== null && testingRuleId !== r.id)}
                      onClick={() => testRuleMatch(r)}
                    >
                      测试
                    </Button>
                  </Tooltip>
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => {
                      setModalOpen({ id: r.id })
                      let thresholds: any[] = []
                      try {
                        const parsed = typeof (r as any).thresholds === 'string' ? JSON.parse((r as any).thresholds) : (r as any).thresholds
                        if (Array.isArray(parsed)) thresholds = parsed
                      } catch { /* empty */ }
                      form.setFieldsValue({
                        ...r,
                        datasource_ids: parseIds(r.datasource_ids),
                        channel_ids: parseIds(r.channel_ids),
                        thresholds: thresholds.length > 0 ? thresholds : undefined,
                      })
                    }}
                  >
                    编辑
                  </Button>
                  <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => batch('delete', [r.id])}>删除</Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      </motion.div>

      <Modal
        title="导入规则"
        open={importModalOpen}
        onOk={doImport}
        onCancel={() => { setImportModalOpen(false); importFileRef.current = null }}
        okText="导入"
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Select
            value={importMode}
            onChange={setImportMode}
            options={[
              { value: 'add', label: '仅添加（保留现有）' },
              { value: 'overwrite', label: '按名称覆盖' },
            ]}
            style={{ width: 260 }}
          />
          <Upload
            accept=".json"
            maxCount={1}
            beforeUpload={(file) => { importFileRef.current = file; return false }}
            onRemove={() => { importFileRef.current = null }}
          >
            <Button>选择 JSON 文件</Button>
          </Upload>
        </Space>
      </Modal>

      <Modal
        title={modalOpen && typeof modalOpen === 'object' ? '编辑规则' : '新建规则'}
        open={!!modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        width={800}
        className="rule-modal"
        styles={{ body: { padding: '24px 24px 8px', maxHeight: 'calc(100vh - 200px)', overflow: 'auto' } }}
      >
        <Form form={form} layout="vertical" onFinish={onFinish} requiredMark={false}>
          {/* ── Section 1: Name + Core Settings ── */}
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]} style={{ marginBottom: 12 }}>
            <Input placeholder="例如：生产应用节点磁盘告警" size="large" />
          </Form.Item>
          <Form.Item name="description" label="描述" style={{ marginBottom: 16 }}>
            <Input.TextArea rows={2} placeholder="规则用途说明（可在通知模板中通过 {{.RuleDescription}} 引用）" />
          </Form.Item>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 12, marginBottom: 20 }}>
            <Form.Item name="enabled" label="状态" valuePropName="checked" initialValue={true} style={{ marginBottom: 0 }}>
              <Switch checkedChildren="启用" unCheckedChildren="停用" />
            </Form.Item>
            <Form.Item name="priority" label="优先级" initialValue={0} style={{ marginBottom: 0 }}>
              <InputNumber placeholder="0" style={{ width: '100%' }} min={0} />
            </Form.Item>
            <Form.Item name="check_interval" label="检测频率" initialValue="1m" style={{ marginBottom: 0 }}>
              <Select options={[
                { value: '10s', label: '10秒' },
                { value: '30s', label: '30秒' },
                { value: '1m', label: '1分钟' },
                { value: '5m', label: '5分钟' },
                { value: '10m', label: '10分钟' },
              ]} />
            </Form.Item>
            <Form.Item name="datasource_ids" label="数据源" style={{ marginBottom: 0 }}>
              <Select
                mode="multiple"
                placeholder="全部"
                allowClear
                maxTagCount={1}
                options={datasources.map((d) => ({ value: d.id, label: d.type ? `${d.name} (${d.type})` : d.name }))}
              />
            </Form.Item>
          </div>

          <Divider style={{ margin: '0 0 16px' }} />

          {/* ── Section 2: Query ── */}
          <div style={{ display: 'grid', gridTemplateColumns: '160px 1fr auto', gap: 12, alignItems: 'start', marginBottom: 16 }}>
            <Form.Item name="query_language" label="查询语言" style={{ marginBottom: 0 }}>
              <Select options={QUERY_LANG_OPTIONS} placeholder="无" allowClear />
            </Form.Item>
            {queryLang ? (
              <Form.Item
                name="query_expression"
                label="查询语句"
                style={{ marginBottom: 0 }}
              >
                <Input.TextArea
                  rows={2}
                  placeholder={queryLang === 'promql' ? 'up == 0' : queryLang === 'elasticsearch_sql' ? 'SELECT * FROM index WHERE ...' : 'SELECT ...'}
                  style={{ fontFamily: 'monospace', fontSize: 13 }}
                />
              </Form.Item>
            ) : (
              <Form.Item label="查询语句" style={{ marginBottom: 0 }}>
                <Input.TextArea rows={2} placeholder="请先选择查询语言" disabled />
              </Form.Item>
            )}
            <Form.Item label=" " style={{ marginBottom: 0 }}>
              <Button icon={<ExperimentOutlined />} onClick={testMatch} loading={testLoading}>
                测试
              </Button>
            </Form.Item>
          </div>

          <Form.Item name="match_labels" label="匹配标签" style={{ marginBottom: 16 }}>
            <Input placeholder='可选，如 {"job":"api","env":"prod"}' />
          </Form.Item>

          <Divider style={{ margin: '0 0 16px' }} />

          {/* ── Section 3: Notification ── */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 4 }}>
            <Form.Item name="channel_ids" label="通知渠道" style={{ marginBottom: 12 }}>
              <Select
                mode="multiple"
                placeholder="选择通知渠道"
                allowClear
                options={channels.map((c) => ({ value: c.id, label: c.type ? `${c.name} (${c.type})` : c.name }))}
              />
            </Form.Item>
            <Form.Item name="template_id" label="通知模板" style={{ marginBottom: 12 }}>
              <Select
                placeholder="默认模板"
                allowClear
                options={templates.map((t: { id: number; name: string; is_default?: boolean }) => ({ value: t.id, label: t.is_default ? `${t.name} (默认)` : t.name }))}
              />
            </Form.Item>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12, marginBottom: 16 }}>
            <Form.Item name="duration" label="持续时间" style={{ marginBottom: 0 }} tooltip="告警持续多久后才通知，如 5m">
              <Input placeholder="0（立即通知）" />
            </Form.Item>
            <Form.Item name="send_interval" label="发送间隔" style={{ marginBottom: 0 }} tooltip="同一告警最小通知间隔，如 5m">
              <Input placeholder="0（不限制）" />
            </Form.Item>
            <Form.Item name="recovery_notify" label="恢复通知" valuePropName="checked" initialValue={false} style={{ marginBottom: 0 }}>
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
          </div>

          {/* ── Section 4: Multi-level thresholds (collapsible) ── */}
          <Collapse
            ghost
            style={{ marginBottom: 8 }}
            items={[
              {
                key: 'thresholds',
                label: <span style={{ fontWeight: 500 }}>多级阈值 <span style={{ fontWeight: 400, color: '#8c8c8c', fontSize: 12 }}>— 按不同阈值分级发送到不同渠道</span></span>,
                children: (
                  <div style={{ padding: '8px 0' }}>
                    <Form.List name="thresholds">
                      {(fields, { add, remove }) => (
                        <>
                          {fields.length > 0 && (
                            <div style={{ display: 'grid', gridTemplateColumns: '80px 90px 110px 1fr 28px', gap: 8, marginBottom: 4, padding: '0 0 4px', color: '#8c8c8c', fontSize: 12 }}>
                              <span>比较</span><span>阈值</span><span>级别</span><span>通知渠道</span><span />
                            </div>
                          )}
                          {fields.map(({ key, name, ...restField }) => (
                            <div key={key} style={{ display: 'grid', gridTemplateColumns: '80px 90px 110px 1fr 28px', gap: 8, alignItems: 'center', marginBottom: 6 }}>
                              <Form.Item {...restField} name={[name, 'operator']} style={{ marginBottom: 0 }} initialValue=">">
                                <Select size="small" options={[
                                  { value: '>', label: '>' }, { value: '>=', label: '>=' },
                                  { value: '<', label: '<' }, { value: '<=', label: '<=' },
                                  { value: '==', label: '==' }, { value: '!=', label: '!=' },
                                ]} />
                              </Form.Item>
                              <Form.Item {...restField} name={[name, 'value']} style={{ marginBottom: 0 }} rules={[{ required: true, message: '' }]}>
                                <InputNumber size="small" placeholder="值" style={{ width: '100%' }} />
                              </Form.Item>
                              <Form.Item {...restField} name={[name, 'severity']} style={{ marginBottom: 0 }} initialValue="warning">
                                <Select size="small" options={[
                                  { value: 'critical', label: '严重' },
                                  { value: 'warning', label: '警告' },
                                  { value: 'info', label: '信息' },
                                ]} />
                              </Form.Item>
                              <Form.Item {...restField} name={[name, 'channel_ids']} style={{ marginBottom: 0 }}>
                                <Select size="small" mode="multiple" placeholder="默认渠道" allowClear maxTagCount={2}
                                  options={channels.map((c) => ({ value: c.id, label: c.type ? `${c.name} (${c.type})` : c.name }))}
                                />
                              </Form.Item>
                              <MinusCircleOutlined style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 14 }} onClick={() => remove(name)} />
                            </div>
                          ))}
                          <Button type="dashed" size="small" onClick={() => add({ operator: '>', severity: 'warning' })} icon={<PlusOutlined />} block>
                            添加阈值级别
                          </Button>
                        </>
                      )}
                    </Form.List>
                  </div>
                )
              },
              {
                key: 'advanced',
                label: <span style={{ fontWeight: 500 }}>高级设置 <span style={{ fontWeight: 400, color: '#8c8c8c', fontSize: 12 }}>— 聚合、排除时段、静默</span></span>,
                children: (
                  <div style={{ padding: '8px 0' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: 'auto 1fr 1fr', gap: 12, alignItems: 'end', marginBottom: 12 }}>
                      <Form.Item name="aggregation_enabled" label="聚合" valuePropName="checked" initialValue={false} style={{ marginBottom: 0 }}>
                        <Switch size="small" checkedChildren="开" unCheckedChildren="关" />
                      </Form.Item>
                      <Form.Item name="aggregate_by" label="聚合字段" style={{ marginBottom: 0 }}>
                        <Input size="small" placeholder="hostname / instance" />
                      </Form.Item>
                      <Form.Item name="aggregate_window" label="聚合窗口" style={{ marginBottom: 0 }}>
                        <Input size="small" placeholder="5m" />
                      </Form.Item>
                    </div>
                    <Form.Item name="exclude_windows" label="排除时段" style={{ marginBottom: 12 }}>
                      <Input size="small" placeholder='[{"start":"22:00","end":"08:00"}]' />
                    </Form.Item>
                    <Form.Item name="suppression" label="静默配置" style={{ marginBottom: 0 }}>
                      <Input size="small" placeholder='{"source_labels":{...},"suppressed_labels":{...},"duration":"30m"}' />
                    </Form.Item>
                  </div>
                )
              },
              {
                key: 'jira',
                label: <span style={{ fontWeight: 500 }}>Jira 集成 <span style={{ fontWeight: 400, color: '#8c8c8c', fontSize: 12 }}>— 自动创建工单</span></span>,
                children: (
                  <div style={{ padding: '8px 0' }}>
                    <div style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: 12, alignItems: 'end', marginBottom: 12 }}>
                      <Form.Item name="jira_enabled" label="启用" valuePropName="checked" initialValue={false} style={{ marginBottom: 0 }}>
                        <Switch size="small" />
                      </Form.Item>
                      <Form.Item name="jira_after_n" label="N 次告警后创建工单" initialValue={3} style={{ marginBottom: 0 }}>
                        <InputNumber size="small" min={1} style={{ width: '100%' }} />
                      </Form.Item>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 12 }}>
                      <Form.Item name="jira_base_url" label="Jira 地址" style={{ marginBottom: 0 }}>
                        <Input size="small" placeholder="https://your.atlassian.net" />
                      </Form.Item>
                      <Form.Item name="jira_project" label="项目 Key" style={{ marginBottom: 0 }}>
                        <Input size="small" placeholder="PROJ" />
                      </Form.Item>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 12 }}>
                      <Form.Item name="jira_email" label="登录邮箱" style={{ marginBottom: 0 }}>
                        <Input size="small" placeholder="user@example.com" />
                      </Form.Item>
                      <Form.Item name="jira_issue_type" label="问题类型" style={{ marginBottom: 0 }}>
                        <Input size="small" placeholder="Task" />
                      </Form.Item>
                    </div>
                    <Form.Item name="jira_token" label="API Token" style={{ marginBottom: 0 }}>
                      <Input.Password size="small" placeholder="编辑时留空保持原值" />
                    </Form.Item>
                  </div>
                )
              }
            ]}
          />

          <Form.Item style={{ marginTop: 20, marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" size="large" block>保存规则</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={
          <Space>
            {testRuleName && <span style={{ fontWeight: 600, marginRight: 8 }}>「{testRuleName}」</span>}
            {testResult?.matched ? (
              <><CheckCircleOutlined style={{ color: '#52c41a' }} /> 匹配成功</>
            ) : (
              <><CloseCircleOutlined style={{ color: '#ff4d4f' }} /> 未匹配到告警</>
            )}
          </Space>
        }
        open={testModalOpen}
        onCancel={() => { setTestModalOpen(false); setTestRuleName(null) }}
        footer={[
          <Button key="close" onClick={() => setTestModalOpen(false)}>
            关闭
          </Button>
        ]}
        width={840}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
      >
        {testResult && (
          <div style={{ minWidth: 0 }}>
            {testResult.message && (
              <p style={{ marginBottom: 16, color: testResult.matched ? undefined : '#666' }}>
                {testResult.message}
              </p>
            )}
            <div style={{ marginBottom: 16, padding: '12px 16px', background: '#f6f8fa', borderRadius: 8 }}>
              <span style={{ color: '#666' }}>
                {typeof testResult.raw_series_count === 'number' && <>PromQL {testResult.raw_series_count} 条</>}
                {typeof testResult.raw_series_count === 'number' && (testResult.total_alerts != null || testResult.matched_alerts?.length != null) && ' → '}
                {testResult.total_alerts != null && <>触发 {testResult.total_alerts} 条</>}
                {(typeof testResult.raw_series_count === 'number' || testResult.total_alerts != null) && ' → '}
                <strong style={{ color: testResult.matched ? '#52c41a' : '#ff4d4f' }}>
                  匹配 {testResult.matched_alerts?.length ?? 0} 条
                </strong>
              </span>
              {/* Severity breakdown when multi-level thresholds produce mixed severities */}
              {(() => {
                const alerts = testResult.matched_alerts || []
                const critical = alerts.filter((a: any) => a.severity === 'critical').length
                const warning = alerts.filter((a: any) => a.severity === 'warning').length
                const info = alerts.filter((a: any) => a.severity === 'info').length
                const hasMultiple = [critical, warning, info].filter(n => n > 0).length > 1
                                    || (critical > 0 || warning > 0 || info > 0)
                if (!hasMultiple || alerts.length === 0) return null
                return (
                  <div style={{ marginTop: 8 }}>
                    {critical > 0 && <Tag color="red">严重 {critical}</Tag>}
                    {warning > 0 && <Tag color="orange">警告 {warning}</Tag>}
                    {info > 0 && <Tag color="blue">提示 {info}</Tag>}
                  </div>
                )
              })()}
            </div>

            {testResult.matched_alerts?.length > 0 && (
              <div style={{ overflow: 'hidden' }}>
                <h4 style={{ marginBottom: 12 }}>匹配到的告警</h4>
                <Table
                  dataSource={testResult.matched_alerts}
                  rowKey="id"
                  size="small"
                  pagination={{ pageSize: 10 }}
                  scroll={{ x: 740 }}
                  columns={[
                    {
                      title: '关键标签',
                      key: 'labels_short',
                      width: 200,
                      ellipsis: true,
                      render: (_, r) => {
                        const raw = (r as { labels?: string | Record<string, unknown> }).labels
                        const lb = typeof raw === 'string' ? (() => { try { return JSON.parse(raw) || {} } catch { return {} } })() : (raw || {})
                        const job = lb.job != null ? String(lb.job) : ''
                        const instance = lb.instance != null ? String(lb.instance) : ''
                        const parts = [job, instance].filter(Boolean)
                        const text = parts.length ? parts.join(' · ') : '-'
                        return (
                          <Tooltip title={text.length > 30 ? text : null}>
                            <Typography.Text ellipsis style={{ display: 'block' }}>
                              {text || '-'}
                            </Typography.Text>
                          </Tooltip>
                        )
                      }
                    },
                    {
                      title: '标题',
                      dataIndex: 'title',
                      width: 220,
                      ellipsis: true,
                      render: (title) => (
                        <Tooltip title={title}>
                          <Typography.Text ellipsis style={{ display: 'block' }}>
                            {title}
                          </Typography.Text>
                        </Tooltip>
                      )
                    },
                    {
                      title: '值',
                      dataIndex: 'value',
                      width: 80,
                      render: (v: number) => <Typography.Text code>{typeof v === 'number' ? v.toFixed(2) : '-'}</Typography.Text>
                    },
                    {
                      title: '严重程度',
                      dataIndex: 'severity',
                      width: 88,
                      filters: [
                        { text: '严重', value: 'critical' },
                        { text: '警告', value: 'warning' },
                        { text: '提示', value: 'info' },
                      ],
                      onFilter: (value, record: any) => record.severity === value,
                      render: (severity) => (
                        <Tag color={severity === 'critical' ? 'red' : severity === 'warning' ? 'orange' : 'blue'}>
                          {severity === 'critical' ? '严重' : severity === 'warning' ? '警告' : '提示'}
                        </Tag>
                      )
                    },
                    {
                      title: '状态',
                      dataIndex: 'status',
                      width: 76,
                      render: (status) => (
                        <Tag color={status === 'firing' ? 'red' : 'green'}>
                          {status === 'firing' ? '告警中' : '已恢复'}
                        </Tag>
                      )
                    },
                  ]}
                />
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}
