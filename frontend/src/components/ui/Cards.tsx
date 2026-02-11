import { motion } from 'framer-motion'
import { Card, CardProps } from 'antd'
import { ReactNode } from 'react'

interface AnimatedCardProps extends CardProps {
  children: ReactNode
  delay?: number
  hover?: boolean
  className?: string
}

export function AnimatedCard({ 
  children, 
  delay = 0, 
  hover = true,
  className,
  ...cardProps 
}: AnimatedCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ 
        duration: 0.4, 
        delay,
        ease: [0.25, 0.1, 0.25, 1]
      }}
      whileHover={hover ? { 
        y: -4,
        transition: { duration: 0.2 }
      } : undefined}
      className={className}
    >
      <Card {...cardProps}>{children}</Card>
    </motion.div>
  )
}

interface StatCardProps {
  title: string
  value: string | number
  change?: {
    value: number
    type: 'up' | 'down'
  }
  icon: ReactNode
  color?: string
  delay?: number
}

export function StatCard({ 
  title, 
  value, 
  change, 
  icon, 
  color = '#1890ff',
  delay = 0 
}: StatCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay }}
      whileHover={{ y: -4, transition: { duration: 0.2 } }}
      className="stat-card-wrapper"
    >
      <Card variant="borderless" className="stat-card">
        <div className="stat-card-content">
          <div 
            className="stat-icon"
            style={{ background: `${color}15`, color }}
          >
            {icon}
          </div>
          <div className="stat-info">
            <div className="stat-title">{title}</div>
            <div className="stat-value-wrapper">
              <span className="stat-value">{value}</span>
              {change && (
                <span 
                  className={`stat-change ${change.type}`}
                >
                  {change.type === 'up' ? '+' : ''}{change.value}%
                </span>
              )}
            </div>
          </div>
        </div>
      </Card>
    </motion.div>
  )
}

interface PageHeaderProps {
  title: string
  subtitle?: string
  actions?: ReactNode
  breadcrumbs?: ReactNode
}

export function PageHeader({ title, subtitle, actions, breadcrumbs }: PageHeaderProps) {
  return (
    <motion.div 
      className="page-header"
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      {breadcrumbs && <div className="page-breadcrumbs">{breadcrumbs}</div>}
      <div className="page-header-content">
        <div>
          <h1 className="page-title">{title}</h1>
          {subtitle && <p className="page-subtitle">{subtitle}</p>}
        </div>
        {actions && <div className="page-actions">{actions}</div>}
      </div>
    </motion.div>
  )
}
