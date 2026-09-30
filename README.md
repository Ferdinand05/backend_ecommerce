
```
backend_ecommerce
├─ .dockerignore
├─ cmd
│  └─ main.go
├─ config
│  └─ config.go
├─ database
│  ├─ migrations
│  │  ├─ 000001_roles.down.sql
│  │  ├─ 000001_roles.up.sql
│  │  ├─ 000002_users.down.sql
│  │  ├─ 000002_users.up.sql
│  │  ├─ 000003_email_verifications.down.sql
│  │  ├─ 000003_email_verifications.up.sql
│  │  ├─ 000004_password_resets.down.sql
│  │  ├─ 000004_password_resets.up.sql
│  │  ├─ 000005_refresh_tokens.down.sql
│  │  └─ 000005_refresh_tokens.up.sql
│  ├─ postgre.go
│  └─ transaction.go
├─ deployment
│  ├─ docker-compose.yaml
│  └─ Dockerfile
├─ go.mod
├─ go.sum
├─ internal
│  ├─ auth
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ route.go
│  │  └─ service.go
│  ├─ middleware
│  │  └─ auth_middleware.go
│  ├─ models
│  │  ├─ email_verification.go
│  │  ├─ password_reset.go
│  │  ├─ refresh_token.go
│  │  ├─ role.go
│  │  └─ user.go
│  ├─ router
│  │  └─ router.go
│  └─ user
│     ├─ dto.go
│     ├─ error.go
│     ├─ handler.go
│     ├─ repository.go
│     ├─ route.go
│     └─ service.go
├─ Makefile
├─ README.md
└─ utils
   ├─ crypto
   │  └─ password.go
   └─ jwt
      └─ jwt.go

```