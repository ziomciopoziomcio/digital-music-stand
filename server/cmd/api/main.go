package main

import (
	"fmt"
	"log"
	"net"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/services"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=devuser password=devpassword dbname=musicstand port=5432 sslmode=disable"

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
	)

	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	fmt.Println("Migrated database")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.AuthInterceptor),
	)

	userpb.RegisterUserServiceServer(grpcServer, services.NewUserService(db))
	scorepb.RegisterScoreServiceServer(grpcServer, services.NewScoreService(db))
	bandpb.RegisterBandServiceServer(grpcServer, services.NewBandService(db))
	concertpb.RegisterConcertServiceServer(grpcServer, services.NewConcertService(db))
	syncpb.RegisterLiveSyncServiceServer(grpcServer, services.NewLiveSyncService())

	log.Println("Listening on", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
