
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
│  │  ├─ 000005_refresh_tokens.up.sql
│  │  ├─ 000006_categories.down.sql
│  │  ├─ 000006_categories.up.sql
│  │  ├─ 000007_products.down.sql
│  │  ├─ 000007_products.up.sql
│  │  ├─ 000008_product_variants.down.sql
│  │  ├─ 000008_product_variants.up.sql
│  │  ├─ 000009_product_images.down.sql
│  │  ├─ 000009_product_images.up.sql
│  │  ├─ 000010_inventory_items.down.sql
│  │  ├─ 000010_inventory_items.up.sql
│  │  ├─ 000011_stock_movements.down.sql
│  │  ├─ 000011_stock_movements.up.sql
│  │  ├─ 000012_cart_items.down.sql
│  │  ├─ 000012_cart_items.up.sql
│  │  ├─ 000013_orders.down.sql
│  │  ├─ 000013_orders.up.sql
│  │  ├─ 000014_order_items.down.sql
│  │  ├─ 000014_order_items.up.sql
│  │  ├─ 000015_order_addresses.down.sql
│  │  ├─ 000015_order_addresses.up.sql
│  │  ├─ 000016_order_status_histories.down.sql
│  │  └─ 000016_order_status_histories.up.sql
│  ├─ postgre.go
│  ├─ transaction.go
│  └─ transaction_test.go
├─ deployment
│  ├─ docker-compose.yaml
│  ├─ Dockerfile
│  └─ postgres
│     └─ init
│        └─ 001-create-test-db.sql
├─ go.mod
├─ go.sum
├─ internal
│  ├─ auth
│  │  ├─ dto.go
│  │  ├─ email_verification_repository.go
│  │  ├─ email_verification_repository_integration_test.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ password_reset_repository.go
│  │  ├─ password_reset_repository_integration_test.go
│  │  ├─ refresh_token_repository.go
│  │  ├─ refresh_token_repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  ├─ service_integration_test.go
│  │  └─ service_test.go
│  ├─ cart_items
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ handler_test.go
│  │  ├─ repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  ├─ service_integration_test.go
│  │  └─ service_test.go
│  ├─ category
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ handler_test.go
│  │  ├─ repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  └─ service_test.go
│  ├─ inventory
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ handler_test.go
│  │  ├─ inventory_item_repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  ├─ service_integration_test.go
│  │  ├─ service_test.go
│  │  └─ stock_movement_repository.go
│  ├─ mail
│  │  ├─ sender.go
│  │  └─ smtp.go
│  ├─ middleware
│  │  ├─ auth_middleware.go
│  │  ├─ auth_middleware_test.go
│  │  ├─ request_logger.go
│  │  └─ request_logger_test.go
│  ├─ models
│  │  ├─ cart_item.go
│  │  ├─ category.go
│  │  ├─ email_verification.go
│  │  ├─ inventory_item.go
│  │  ├─ order.go
│  │  ├─ order_address.go
│  │  ├─ order_item.go
│  │  ├─ order_status_history.go
│  │  ├─ password_reset.go
│  │  ├─ product.go
│  │  ├─ product_image.go
│  │  ├─ product_variant.go
│  │  ├─ refresh_token.go
│  │  ├─ role.go
│  │  ├─ stock_movement.go
│  │  └─ user.go
│  ├─ product
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ handler_test.go
│  │  ├─ repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  └─ service_test.go
│  ├─ product_image
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ handler_test.go
│  │  ├─ repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  └─ service_test.go
│  ├─ product_variant
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ handler_test.go
│  │  ├─ repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  ├─ service_integration_test.go
│  │  └─ service_test.go
│  ├─ role
│  │  ├─ dto.go
│  │  ├─ error.go
│  │  ├─ handler.go
│  │  ├─ repository.go
│  │  ├─ repository_integration_test.go
│  │  ├─ route.go
│  │  ├─ service.go
│  │  └─ service_test.go
│  ├─ router
│  │  ├─ router.go
│  │  └─ router_test.go
│  ├─ storage
│  │  ├─ cloudflare
│  │  │  └─ r2.go
│  │  └─ storage.go
│  └─ user
│     ├─ dto.go
│     ├─ error.go
│     ├─ handler.go
│     ├─ repository.go
│     ├─ repository_integration_test.go
│     ├─ route.go
│     ├─ service.go
│     └─ service_test.go
├─ Makefile
├─ README.md
└─ utils
   ├─ crypto
   │  ├─ password.go
   │  └─ password_test.go
   ├─ jwt
   │  ├─ jwt.go
   │  └─ jwt_test.go
   ├─ slug
   │  └─ slug.go
   └─ token
      ├─ token.go
      └─ token_test.go

```