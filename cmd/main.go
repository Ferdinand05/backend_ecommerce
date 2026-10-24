package main

import (
	"context"
	"ferdinand/ecommerce/config"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/database/redis"
	"ferdinand/ecommerce/internal/auth"
	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/checkout"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/mail"
	"ferdinand/ecommerce/internal/order"
	"ferdinand/ecommerce/internal/payment"
	paymentxendit "ferdinand/ecommerce/internal/payment/xendit"
	"ferdinand/ecommerce/internal/product"
	productimage "ferdinand/ecommerce/internal/product_image"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/router"
	"ferdinand/ecommerce/internal/shipment"
	"ferdinand/ecommerce/internal/shipping"
	"ferdinand/ecommerce/internal/shipping/biteship"
	"ferdinand/ecommerce/internal/storage/cloudflare"
	"ferdinand/ecommerce/internal/user"
	userjwt "ferdinand/ecommerce/utils/jwt"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"
)

func main() {

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgres(cfg.Database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB, err := db.DB()
	if err := sqlDB.PingContext(ctx); err != nil {
		fmt.Printf("failed to connect to database:%v", err)
		return
	}

	fmt.Println("Connected to database!")

	defer sqlDB.Close()

	//jwt
	jwtSvc := userjwt.NewJWTService(cfg.JWT.SecretKey)
	//redis
	redisClientService := redis.NewRedisClient(cfg.Redis)

	_, err = redisClientService.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	fmt.Println("Connected to redis!")

	redisClientCache := redis.NewCache(redisClientService)

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	roleRepo := role.NewRepository(db)
	roleSvc := role.NewService(roleRepo)
	roleHandler := role.NewHandler(roleSvc)

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo, redisClientCache)
	categoryHandler := category.NewHandler(categoryService)

	r2Storage, err := cloudflare.NewStorage(ctx, cloudflare.Config{
		AccountID:       cfg.R2.AccountID,
		AccessKeyID:     cfg.R2.AccessKeyID,
		SecretAccessKey: cfg.R2.SecretAccessKey,
		Bucket:          cfg.R2.Bucket,
		PublicURL:       cfg.R2.PublicURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo, categoryRepo, redisClientCache, r2Storage)
	productHandler := product.NewHandler(productService, r2Storage)

	productVariantRepo := productvariant.NewRepository(db)

	inventoryItemRepo := inventory.NewInventoryRepository(db)

	productVariantService := productvariant.NewService(productVariantRepo, productRepo, inventoryItemRepo, db)
	productVariantHandler := productvariant.NewHandler(productVariantService, r2Storage)

	productImageRepo := productimage.NewRepository(db)
	productImageService := productimage.NewService(productImageRepo, productRepo, productVariantRepo, r2Storage)
	productImageHandler := productimage.NewHandler(productImageService)

	stockMovementRepo := inventory.NewStockMovementRepository(db)
	inventoryService := inventory.NewService(inventoryItemRepo, stockMovementRepo, productVariantRepo, db)
	inventoryHandler := inventory.NewHandler(inventoryService)

	cartRepo := cartitems.NewRepository(db)
	cartService := cartitems.NewService(cartRepo, productVariantRepo)
	cartHandler := cartitems.NewHandler(cartService)

	orderRepo := order.NewRepository(db)

	emailVerificationRepo := auth.NewEmailVerificationRepository(db)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	passwordResetRepo := auth.NewPasswordResetRepository(db)
	emailSender := mail.NewSMTPSender(mail.SMTPConfig(cfg.Mail))

	// Biteship client
	timeout, _ := strconv.Atoi(cfg.Biteship.Timeout)
	timeoutInSecond := time.Duration(timeout) * time.Second
	biteshipClient := biteship.NewClient(cfg.Biteship.BaseURL, cfg.Biteship.ApiKey, timeoutInSecond)

	// Biteship provider
	biteshipProvider := biteship.NewProvider(biteshipClient)

	shippingService := shipping.NewService(
		redisClientCache,
		biteshipProvider,
		productVariantRepo,
		cfg.Biteship.OriginAreaID,
		cfg.Biteship.Couriers,
	)
	shippingHandler := shipping.NewHandler(shippingService)

	orderService := order.NewService(
		orderRepo,
		db,
		cartService,
		inventoryItemRepo,
		productVariantRepo,
		shippingService,
		stockMovementRepo,
	)
	orderHandler := order.NewHandler(orderService)

	authSvc := auth.NewService(
		userRepo,
		roleRepo,
		jwtSvc,
		emailSender,
		emailVerificationRepo,
		refreshTokenRepo,
		passwordResetRepo,
		db,
	)
	authHandler := auth.NewHandler(authSvc)

	checkoutService := checkout.NewService(cartService, inventoryItemRepo, productVariantRepo)
	checkoutHandler := checkout.NewHandler(checkoutService)

	shipmentRepo := shipment.NewRepository(db)
	shipmentService := shipment.NewService(
		shipmentRepo,
		orderRepo,
		biteshipProvider,
		shipment.Origin{
			Name:    cfg.Biteship.OriginContactName,
			Phone:   cfg.Biteship.OriginContactPhone,
			Address: cfg.Biteship.OriginAddress,
			AreaID:  cfg.Biteship.OriginAreaID,
		},
	)
	shipmentHandler := shipment.NewHandler(shipmentService)

	xenditClient := paymentxendit.NewClient(cfg.Xendit.ApiKey)
	paymentRepo := payment.NewRepository(db)
	paymentService := payment.NewService(db, orderRepo, paymentRepo, inventoryItemRepo, stockMovementRepo, xenditClient, shipmentService)
	paymentHandler := payment.NewHandler(paymentService, cfg.Xendit.WebhookToken)

	handlers := router.RouteHandlers{
		UserHandler:           userHandler,
		AuthHandler:           authHandler,
		RoleHandler:           roleHandler,
		CategoryHandler:       categoryHandler,
		ProductHandler:        productHandler,
		ProductVariantHandler: productVariantHandler,
		ProductImageHandler:   productImageHandler,
		InventoryHandler:      inventoryHandler,
		CartHandler:           cartHandler,
		OrderHandler:          orderHandler,
		ShippingHandler:       shippingHandler,
		CheckoutHandler:       checkoutHandler,
		PaymentHandler:        paymentHandler,
		ShipmentHandler:       shipmentHandler,
	}

	r := router.New(handlers, jwtSvc)
	err = r.Run(":" + cfg.Database.AppPort)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("App Listening to" + cfg.Database.Port)

}
