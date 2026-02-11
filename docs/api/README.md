# KK Alert API 文档

## Swagger UI

服务启动后访问：

- **Swagger UI**: [http://localhost:8080/swagger/](http://localhost:8080/swagger/)
- **OpenAPI 规范 (JSON)**: [http://localhost:8080/api/openapi.json](http://localhost:8080/api/openapi.json)

## Token 授权

1. 在 Swagger 页面中先调用 **POST /api/v1/auth/login**，请求体示例：
   ```json
   { "username": "admin", "password": "admin123" }
   ```
2. 响应中的 `token` 即为 JWT。
3. 点击页面右上角 **Authorize**，在弹窗中粘贴该 token（无需手写 `Bearer ` 前缀），确认。
4. 之后所有需认证的接口会自动在请求头中携带 `Authorization: Bearer <token>`。

## 命令行调用示例

```bash
# 登录获取 token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# 带 token 调用接口
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/alerts?page=1
```
