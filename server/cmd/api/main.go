package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/admin"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/services"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	jwtExpStr := os.Getenv("JWT_EXPIRATION_HOURS")
	if jwtExpStr == "" {
		log.Println("JWT_EXPIRATION_HOURS environment variable is empty, defaulting to 24 hours")
		jwtExpStr = "24"
	}
	jwtExpHours, err := strconv.Atoi(jwtExpStr)
	if err != nil {
		log.Fatalf("Invalid JWT_EXPIRATION_HOURS value: %v", err)
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		log.Fatal("Database environment variables (DB_HOST, DB_PORT, DB_USER, DB_NAME) are required")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC", dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}

	fmt.Println("Connected to the database")

	err = db.AutoMigrate(
		&models.User{},
		&models.Score{},
		&models.Band{},
		&models.BandMember{},
		&models.SharedBandScore{},
		&models.SharedUserScore{},
		&models.Concert{},
		&models.ConcertItem{},
		&models.BandInvitation{},
		&models.SharedUserScore{},
		&models.SharedBandScore{},
		&models.ShareScoreInvitation{},
		&models.SharedUserConcert{},
		&models.SharedBandConcert{},
		&models.ShareConcertInvitation{},
	)

	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	fmt.Println("Migrated database")

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	minioBucket := os.Getenv("MINIO_BUCKET_NAME")
	useSSL, _ := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, minioBucket)
	if err == nil && !exists {
		err = minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("Warning: failed to create MinIO bucket '%s': %v", minioBucket, err)
		} else {
			log.Printf("Successfully created MinIO bucket: %s", minioBucket)
		}
	} else if err != nil {
		log.Printf("Error checking MinIO bucket: %v", err)
	} else {
		log.Printf("MinIO connected. Bucket '%s' already exists.", minioBucket)
	}

	adminSv := admin.NewAdminServer(db, 8081)
	go func() {
		if err := adminSv.Start(); err != nil {
			log.Printf("Admin HTTP Server error: %v", err)
		}
	}()

	lis, err := net.Listen("tcp4", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(auth.NewAuthInterceptor(jwtSecret, db)))

	userpb.RegisterUserServiceServer(grpcServer, services.NewUserService(db, jwtSecret, jwtExpHours))
	scorepb.RegisterScoreServiceServer(grpcServer, services.NewScoreService(db, minioClient, minioBucket))
	bandpb.RegisterBandServiceServer(grpcServer, services.NewBandService(db))
	concertpb.RegisterConcertServiceServer(grpcServer, services.NewConcertService(db))
	syncpb.RegisterLiveSyncServiceServer(grpcServer, services.NewLiveSyncService())

	log.Println("Listening on", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
