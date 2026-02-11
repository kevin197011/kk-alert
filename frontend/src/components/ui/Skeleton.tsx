import { motion } from 'framer-motion'
import { Skeleton, Card, Space } from 'antd'

interface PageSkeletonProps {
  rows?: number
  showHeader?: boolean
  showFilters?: boolean
}

export function PageSkeleton({ 
  rows = 5, 
  showHeader = true,
  showFilters = true 
}: PageSkeletonProps) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2 }}
    >
      {showHeader && (
        <div style={{ marginBottom: 24 }}>
          <Skeleton.Input active style={{ width: 200, height: 32 }} />
        </div>
      )}
      
      {showFilters && (
        <Space style={{ marginBottom: 16 }} wrap>
          <Skeleton.Input active style={{ width: 120 }} />
          <Skeleton.Input active style={{ width: 120 }} />
          <Skeleton.Input active style={{ width: 160 }} />
          <Skeleton.Button active />
        </Space>
      )}
      
      <Card variant="borderless" className="skeleton-card">
        <Skeleton active paragraph={{ rows }} />
      </Card>
    </motion.div>
  )
}

interface CardSkeletonProps {
  count?: number
}

export function CardSkeleton({ count = 4 }: CardSkeletonProps) {
  return (
    <div style={{ 
      display: 'grid', 
      gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
      gap: 24 
    }}>
      {Array.from({ length: count }, (_, i) => (
        <motion.div
          key={i}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: i * 0.1 }}
        >
          <Card variant="borderless" className="skeleton-card">
            <Skeleton active avatar paragraph={{ rows: 2 }} />
          </Card>
        </motion.div>
      ))}
    </div>
  )
}

interface TableSkeletonProps {
  columns?: number
  rows?: number
}

export function TableSkeleton({ columns = 6, rows = 5 }: TableSkeletonProps) {
  return (
    <Card variant="borderless" className="skeleton-card">
      <div style={{ display: 'flex', gap: 16, marginBottom: 16 }}>
        {Array.from({ length: columns }, (_, i) => (
          <Skeleton.Input 
            key={i} 
            active 
            style={{ flex: 1, height: 40 }} 
          />
        ))}
      </div>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} style={{ display: 'flex', gap: 16, marginBottom: 12 }}>
          {Array.from({ length: columns }, (_, j) => (
            <Skeleton.Input 
              key={j} 
              active 
              style={{ flex: 1, height: 32 }} 
            />
          ))}
        </div>
      ))}
    </Card>
  )
}
