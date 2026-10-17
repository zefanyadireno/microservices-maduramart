package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"google.golang.org/grpc"

	"github.com/nusantara-supermart/backend/internal/auth"
	"github.com/nusantara-supermart/backend/internal/catalog"
	"github.com/nusantara-supermart/backend/internal/inventory"
	"github.com/nusantara-supermart/backend/internal/logistics"
	"github.com/nusantara-supermart/backend/internal/order"
	"github.com/nusantara-supermart/backend/internal/payment"
	"github.com/nusantara-supermart/backend/internal/procurement"
	"github.com/nusantara-supermart/backend/internal/promotion"
	"github.com/nusantara-supermart/backend/internal/support"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/database"
	"github.com/nusantara-supermart/backend/pkg/middleware"
	"github.com/nusantara-supermart/backend/pkg/response"
	pb "github.com/nusantara-supermart/backend/proto/catalog/v1"
)

type catalogGRPCHandler struct {
	pb.UnimplementedCatalogServiceServer
	catalogSvc catalog.Service
}

func (h *catalogGRPCHandler) GetCatalogItem(ctx context.Context, req *pb.GetCatalogItemRequest) (*pb.GetCatalogItemResponse, error) {
	product, _, _, err := h.catalogSvc.GetProductByID(req.GetProductId())
	if err != nil {
		return nil, err
	}

	item := &pb.CatalogItem{
		ProductId: req.GetProductId(),
		Sku:       "",
		Name:      "",
		Price: &pb.Money{
			CurrencyCode: "IDR",
			Amount:       0,
		},
		IsActive: false,
	}

	if product != nil {
		item.ProductId = product.ID
		item.Sku = product.SKU
		item.Name = product.Title
		item.IsActive = product.IsPublished
		item.Price = &pb.Money{
			CurrencyCode: "IDR",
			Amount:       int64(product.BasePrice),
		}
	}

	return &pb.GetCatalogItemResponse{
		Item: item,
	}, nil
}

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting PT Nusantara SuperMart Monolith Backend in [%s] mode...", cfg.AppEnv)

	db, err := database.ConnectMySQL(cfg)
	if err != nil {
		log.Printf("WARNING: MySQL connection failed: %v", err)
		log.Println("Backend will retry connecting when serving requests or initializing...")
	} else {
		log.Println("Connected to MySQL database successfully!")
		defer db.Close()
	}

	app := fiber.New(fiber.Config{
		AppName:      "PT Nusantara SuperMart Indonesia API v1.0",
		ServerHeader: "NusantaraSuperMart-Monolith",
	})

	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(middleware.SetupCORS(cfg))

	api := app.Group("/api/v1")

	api.Get("/health", func(c fiber.Ctx) error {
		dbStatus := "healthy"
		if db == nil || db.Ping() != nil {
			dbStatus = "unreachable"
		}
		return response.Success(c, fiber.StatusOK, "Nusantara SuperMart Monolith API is operational", fiber.Map{
			"timestamp": time.Now().Format(time.RFC3339),
			"database":  dbStatus,
			"version":   "1.0.0",
		})
	})

	authRepo := auth.NewRepository(db)
	catalogRepo := catalog.NewRepository(db)
	inventoryRepo := inventory.NewRepository(db)
	orderRepo := order.NewRepository(db)
	paymentRepo := payment.NewRepository(db)
	promotionRepo := promotion.NewRepository(db)
	logisticsRepo := logistics.NewRepository(db)
	procurementRepo := procurement.NewRepository(db)
	supportRepo := support.NewRepository(db)

	authSvc := auth.NewService(authRepo, cfg)
	catalogSvc := catalog.NewService(catalogRepo)
	inventorySvc := inventory.NewService(inventoryRepo)
	orderSvc := order.NewService(orderRepo, catalogSvc, inventorySvc)
	paymentSvc := payment.NewService(paymentRepo, orderSvc)
	promotionSvc := promotion.NewService(promotionRepo)
	logisticsSvc := logistics.NewService(logisticsRepo, orderSvc)
	procurementSvc := procurement.NewService(procurementRepo)
	supportSvc := support.NewService(supportRepo)

	authHandler := auth.NewHandler(authSvc)
	catalogHandler := catalog.NewHandler(catalogSvc)
	inventoryHandler := inventory.NewHandler(inventorySvc)
	orderHandler := order.NewHandler(orderSvc)
	paymentHandler := payment.NewHandler(paymentSvc)
	promotionHandler := promotion.NewHandler(promotionSvc)
	logisticsHandler := logistics.NewHandler(logisticsSvc)
	procurementHandler := procurement.NewHandler(procurementSvc)
	supportHandler := support.NewHandler(supportSvc)

	auth.RegisterRoutes(api, authHandler, cfg)
	catalog.RegisterRoutes(api, catalogHandler, cfg)
	inventory.RegisterRoutes(api, inventoryHandler, cfg)
	order.RegisterRoutes(api, orderHandler, cfg)
	payment.RegisterRoutes(api, paymentHandler, cfg)
	promotion.RegisterRoutes(api, promotionHandler, cfg)
	logistics.RegisterRoutes(api, logisticsHandler, cfg)
	procurement.RegisterRoutes(api, procurementHandler, cfg)
	support.RegisterRoutes(api, supportHandler, cfg)

	// Inisialisasi gRPC Server
	grpcServer := grpc.NewServer()
	pb.RegisterCatalogServiceServer(grpcServer, &catalogGRPCHandler{catalogSvc: catalogSvc})

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC on port %s: %v", grpcPort, err)
	}

	go func() {
		log.Printf("Nusantara SuperMart gRPC server listening on port :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	serverShutdown := make(chan os.Signal, 1)
	signal.Notify(serverShutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-serverShutdown
		log.Println("Shutting down Nusantara SuperMart server gracefully...")
		grpcServer.GracefulStop()
		_ = app.Shutdown()
	}()

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Nusantara SuperMart server listening on http://0.0.0.0%s", addr)
	if err := app.Listen(addr); err != nil {
		log.Printf("Server closed: %v", err)
	}
}
