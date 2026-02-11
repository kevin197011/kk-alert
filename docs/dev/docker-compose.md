# Docker Compose 开发与测试

## 服务说明

- **postgres**: PostgreSQL 16，端口 5432，数据卷 `postgres_data`
- **backend**: Go 后端，端口 8080，通过 `DATABASE_URL` 连接 postgres，健康检查 `/api/v1/health`
- **frontend**: Vite 前端开发服务器，端口 3000，代理 `/api` 到 backend
- **backend-test**: 仅用于跑后端单元测试，不常驻运行

## 常用命令

```bash
# 启动后端 + 前端（后台）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 仅启动后端
docker-compose up -d backend

# 运行后端单元测试
docker-compose run --rm backend-test

# 停止并删除容器
docker-compose down
```

## 访问

- 后端 API: http://localhost:8080  
- 前端页面: http://localhost:3000（默认账号 admin / admin123）

## 注意

- 后端默认使用 PostgreSQL（`DATABASE_URL`）；不设置 `DATABASE_URL` 时使用 SQLite（`DB_PATH`，默认 `data/kkalert.db`）
- 数据持久化：PostgreSQL 使用 volume `postgres_data`
- 若 8080/3000/5432 端口被占用，可在 `docker-compose.yml` 中修改 `ports` 或先停止占用端口的进程
