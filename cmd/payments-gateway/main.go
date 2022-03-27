package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"net"
	bank "payments_gateway/aquiring-bank"
	"payments_gateway/server"

	log "github.com/sirupsen/logrus"

	protos "payments_gateway/protos"
	"payments_gateway/storage/postgres"
)

var (
	port               uint
	dbURL              string
	poolMaxConnections int
	poolMinConnections int
	BankServiceAddr    string
)

func init() {
	flag.StringVar(&dbURL, "db-url", "postgres://user1:123@localhost:5432/payments", "connection to DB")
	flag.IntVar(&poolMaxConnections, "max-db-connections", 5, "max db connections")
	flag.IntVar(&poolMaxConnections, "min-db-connections", 1, "min db connections")
	flag.UintVar(&port, "port", 9090, "grpc server port")
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.Parse()

	ctx := context.Background()

	pool, err := postgres.CreatePgPool(ctx, dbURL, poolMaxConnections, poolMinConnections)
	if err != nil {
		log.WithError(err).Fatal("creating database pgClient")
	}

	pgClient := postgres.New(pool)

	aqBankClient := bank.New()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	protos.RegisterPaymentsServer(grpcServer, server.New(pgClient, aqBankClient))

	log.WithField("port", port).Info("Starting payments-gateway gRPC server")

	grpcServer.Serve(lis)

}
