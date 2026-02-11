import { motion } from 'framer-motion'
import { Empty, Button } from 'antd'
import { 
  InboxOutlined, 
  SearchOutlined, 
  FileAddOutlined,
  WifiOutlined,
  ReloadOutlined
} from '@ant-design/icons'

interface EmptyStateProps {
  type?: 'empty' | 'search' | 'create' | 'error' | 'offline'
  title?: string
  description?: string
  action?: {
    text: string
    onClick: () => void
  }
  className?: string
}

const config = {
  empty: {
    icon: InboxOutlined,
    defaultTitle: '暂无数据',
    defaultDesc: '当前列表为空，请稍后重试',
  },
  search: {
    icon: SearchOutlined,
    defaultTitle: '未找到匹配结果',
    defaultDesc: '请尝试调整搜索条件或筛选器',
  },
  create: {
    icon: FileAddOutlined,
    defaultTitle: '开始创建您的第一条记录',
    defaultDesc: '点击下方按钮添加新数据',
  },
  error: {
    icon: ReloadOutlined,
    defaultTitle: '加载失败',
    defaultDesc: '数据加载出错，请刷新重试',
  },
  offline: {
    icon: WifiOutlined,
    defaultTitle: '网络连接异常',
    defaultDesc: '请检查网络连接后重试',
  },
}

export function EmptyState({ 
  type = 'empty', 
  title, 
  description, 
  action,
  className 
}: EmptyStateProps) {
  const { icon: Icon, defaultTitle, defaultDesc } = config[type]
  
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className={`empty-state ${className || ''}`}
    >
      <Empty
        image={<Icon style={{ fontSize: 64, color: '#d9d9d9' }} />}
        styles={{ image: { height: 80 } }}
        description={
          <div className="empty-state-content">
            <div className="empty-state-title">{title || defaultTitle}</div>
            <div className="empty-state-desc">{description || defaultDesc}</div>
            {action && (
              <Button 
                type="primary" 
                onClick={action.onClick}
                className="empty-state-action"
              >
                {action.text}
              </Button>
            )}
          </div>
        }
      />
    </motion.div>
  )
}
