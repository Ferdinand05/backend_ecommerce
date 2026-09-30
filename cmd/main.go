package main

import (
	"context"
	"ferdinand/ecommerce/config"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/auth"
	"ferdinand/ecommerce/internal/mail"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/router"
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

	emailVerificationRepo := auth.NewEmailVerificationRepository(db)
	emailSender := mail.NewSMTPSender(mail.SMTPConfig(cfg.Mail))

	authSvc := auth.NewService(
		userRepo,
		roleRepo,
		jwtSvc,
		emailSender,
		emailVerificationRepo,
		db,
	)
	authHandler := auth.NewHandler(authSvc)

	handlers := router.RouteHandlers{
		UserHandler: userHandler,
		AuthHandler: authHandler,
		RoleHandler: roleHandler,
	}

	r := router.New(handlers, jwtSvc)
	err = r.Run(":" + cfg.Database.AppPort)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("App Listening to" + cfg.Database.Port)

}
