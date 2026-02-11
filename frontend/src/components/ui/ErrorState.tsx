import { motion } from 'framer-motion'
import { Result, Button } from 'antd'
import { 
  CloseCircleOutlined,
  ReloadOutlined,
  HomeOutlined
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'

interface ErrorStateProps {
  title?: string
  subTitle?: string
  showRetry?: boolean
  showHome?: boolean
  onRetry?: () => void
  status?: '403' | '404' | '500'
}

export function ErrorState({
  title = '操作失败',
  subTitle = '抱歉，操作过程中发生了错误',
  showRetry = true,
  showHome = true,
  onRetry,
  status
}: ErrorStateProps) {
  const navigate = useNavigate()

  const getStatusConfig = () => {
    switch (status) {
      case '403':
        return {
          title: '403',
          subTitle: '抱歉，您没有权限访问此页面',
          icon: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
        }
      case '404':
        return {
          title: '404',
          subTitle: '抱歉，您访问的页面不存在',
          icon: <CloseCircleOutlined style={{ color: '#faad14' }} />,
        }
      case '500':
        return {
          title: '500',
          subTitle: '抱歉，服务器发生错误',
          icon: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
        }
      default:
        return {
          title,
          subTitle,
          icon: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
        }
    }
  }

  const config = getStatusConfig()

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.3 }}
    >
      <Result
        status={status as any}
        icon={!status ? config.icon : undefined}
        title={config.title}
        subTitle={config.subTitle}
        extra={
          <div className="error-actions">
            {showRetry && onRetry && (
              <Button 
                type="primary" 
                icon={<ReloadOutlined />}
                onClick={onRetry}
                className="error-action-btn"
              >
                重试
              </Button>
            )}
            {showHome && (
              <Button 
                icon={<HomeOutlined />}
                onClick={() => navigate('/')}
              >
                返回首页
              </Button>
            )}
          </div>
        }
      />
    </motion.div>
  )
}
