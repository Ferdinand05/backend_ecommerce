package main

import (
	"context"
	"ferdinand/ecommerce/config"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/auth"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/mail"
	"ferdinand/ecommerce/internal/product"
	productimage "ferdinand/ecommerce/internal/product_image"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/router"
	"ferdinand/ecommerce/internal/storage/cloudflare"
	"ferdinand/ecommerce/internal/user"
	userjwt "ferdinand/ecommerce/utils/jwt"
	"fmt"
	"log"
	"time"
)

func main() {

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

	jwtSvc := userjwt.NewJWTService(cfg.JWT.SecretKey)

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	roleRepo := role.NewRepository(db)
	roleSvc := role.NewService(roleRepo)
	roleHandler := role.NewHandler(roleSvc)

	cateogryRepo := category.NewRepository(db)
	categoryService := category.NewService(cateogryRepo)
	categoryHandler := category.NewHandler(categoryService)

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo, cateogryRepo)
	productHandler := product.NewHandler(productService)

	productVariantRepo := productvariant.NewRepository(db)
	productVariantService := productvariant.NewService(productVariantRepo, productRepo, db)
	productVariantHandler := productvariant.NewHandler(productVariantService)

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

	productImageRepo := productimage.NewRepository(db)
	productImageService := productimage.NewService(productImageRepo, productRepo, productVariantRepo, r2Storage)
	productImageHandler := productimage.NewHandler(productImageService)

	emailVerificationRepo := auth.NewEmailVerificationRepository(db)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	passwordResetRepo := auth.NewPasswordResetRepository(db)
	emailSender := mail.NewSMTPSender(mail.SMTPConfig(cfg.Mail))

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

	handlers := router.RouteHandlers{
		UserHandler:           userHandler,
		AuthHandler:           authHandler,
		RoleHandler:           roleHandler,
		CategoryHandler:       categoryHandler,
		ProductHandler:        productHandler,
		ProductVariantHandler: productVariantHandler,
		ProductImageHandler:   productImageHandler,
	}

	r := router.New(handlers, jwtSvc)
	err = r.Run(":" + cfg.Database.AppPort)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("App Listening to" + cfg.Database.Port)

}
