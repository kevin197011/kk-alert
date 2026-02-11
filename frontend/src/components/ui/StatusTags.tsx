import { Tag, Badge } from 'antd'
import { CheckCircleOutlined, ExclamationCircleOutlined, InfoCircleOutlined, CloseCircleOutlined } from '@ant-design/icons'

export type StatusType = 'success' | 'warning' | 'error' | 'info' | 'default'
export type SeverityType = 'critical' | 'warning' | 'info' | 'resolved'
export type AlertStatusType = 'firing' | 'resolved' | 'suppressed'

const statusConfig: Record<StatusType, { color: string; icon: React.ReactNode; label: string }> = {
  success: {
    color: '#52c41a',
    icon: <CheckCircleOutlined />,
    label: '正常',
  },
  warning: {
    color: '#faad14',
    icon: <ExclamationCircleOutlined />,
    label: '警告',
  },
  error: {
    color: '#ff4d4f',
    icon: <CloseCircleOutlined />,
    label: '错误',
  },
  info: {
    color: '#1890ff',
    icon: <InfoCircleOutlined />,
    label: '信息',
  },
  default: {
    color: '#d9d9d9',
    icon: null,
    label: '默认',
  },
}

const severityConfig: Record<SeverityType, { color: string; bg: string; label: string }> = {
  critical: {
    color: '#cf1322',
    bg: '#fff1f0',
    label: '严重',
  },
  warning: {
    color: '#d48806',
    bg: '#fffbe6',
    label: '警告',
  },
  info: {
    color: '#096dd9',
    bg: '#e6f7ff',
    label: '提示',
  },
  resolved: {
    color: '#389e0d',
    bg: '#f6ffed',
    label: '已恢复',
  },
}

const alertStatusConfig: Record<AlertStatusType, { color: string; dot: 'processing' | 'success' | 'default' }> = {
  firing: {
    color: '#ff4d4f',
    dot: 'processing',
  },
  resolved: {
    color: '#52c41a',
    dot: 'success',
  },
  suppressed: {
    color: '#faad14',
    dot: 'default',
  },
}

interface StatusTagProps {
  status: StatusType
  text?: string
  showIcon?: boolean
}

export function StatusTag({ status, text, showIcon = true }: StatusTagProps) {
  const config = statusConfig[status]
  return (
    <Tag 
      color={status}
      icon={showIcon ? config.icon : undefined}
      style={{ 
        borderRadius: 12,
        padding: '2px 10px',
        fontSize: 12,
        fontWeight: 500,
      }}
    >
      {text || config.label}
    </Tag>
  )
}

interface SeverityBadgeProps {
  severity: SeverityType | string
  text?: string
}

export function SeverityBadge({ severity, text }: SeverityBadgeProps) {
  const config = severityConfig[severity as SeverityType] || severityConfig.info
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        padding: '4px 12px',
        borderRadius: 12,
        fontSize: 12,
        fontWeight: 600,
        color: config.color,
        background: config.bg,
        border: `1px solid ${config.color}20`,
      }}
    >
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: '50%',
          background: config.color,
          marginRight: 6,
          display: 'inline-block',
        }}
      />
      {text || config.label}
    </span>
  )
}

interface AlertStatusBadgeProps {
  status: AlertStatusType | string
  text?: string
}

export function AlertStatusBadge({ status, text }: AlertStatusBadgeProps) {
  const config = alertStatusConfig[status as AlertStatusType] || alertStatusConfig.firing
  const labels: Record<string, string> = {
    firing: '告警中',
    resolved: '已恢复',
    suppressed: '已静默',
  }
  
  return (
    <Badge 
      status={config.dot}
      text={text || labels[status] || status}
      color={config.color}
    />
  )
}

interface DataSourceTypeTagProps {
  type: string
}

export function DataSourceTypeTag({ type }: DataSourceTypeTagProps) {
  const typeColors: Record<string, string> = {
    prometheus: '#e6522c',
    elasticsearch: '#fec514',
    doris: '#2c5aa0',
    mysql: '#4479a1',
    postgres: '#336791',
  }
  
  const color = typeColors[type.toLowerCase()] || '#999'
  
  return (
    <Tag
      style={{
        background: `${color}15`,
        borderColor: `${color}40`,
        color: color,
        borderRadius: 4,
        fontSize: 11,
        fontWeight: 600,
        textTransform: 'uppercase',
      }}
    >
      {type}
    </Tag>
  )
}
