package main

import (
	"context"
	"fmt"
	"grpc-example/pb"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

const serviceName = "assets-service"

func StartServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	pb.RegisterAssetsServer(s, &Server{})

	serverAddr := "localhost:50051"

	DefaultRegistry.Register(&ServiceInfo{
		ServiceName: serviceName,
		Address:     serverAddr,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
	})

	fmt.Printf("Server started and registered at %s\n", serverAddr)

	go func() {
		err = s.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down server...")
	DefaultRegistry.Unregister(serviceName, serverAddr)
	s.GracefulStop()
	fmt.Println("Server stopped")
}

var cache sync.Map

type Server struct {
	pb.UnimplementedAssetsServer
}

func (Server) InitAssets(ctx context.Context, args *pb.InitAssetsArgs) (*pb.InitAssetsReply, error) {
	fmt.Println("Init Assets", args)
	cache.Store(args.UID, &pb.FinalAssets{
		Gold: args.GetGold(),
	})
	return &pb.InitAssetsReply{
		Gold: args.GetGold(),
	}, nil
}

func (Server) FetchAssets(ctx context.Context, args *pb.FetchAssetsArgs) (*pb.FetchAssetsReply, error) {
	fmt.Println("Fetch Assets", args)

	value, ok := cache.Load(args.UID)
	if !ok {
		return nil, fmt.Errorf("fetch Assets failed ")
	}

	fmt.Println("Fetch Assets", value)

	return &pb.FetchAssetsReply{
		Gold: value.(*pb.FinalAssets).Gold,
	}, nil
}

func (Server) ChangeAssets(ctx context.Context, args *pb.ChangeAssetsArgs) (*pb.ChangeAssetsReply, error) {
	fmt.Println("Change Assets", args)
	value, ok := cache.LoadOrStore(args.UID, &pb.FinalAssets{
		Gold: args.GetGold(),
	})
	if !ok {
		return nil, fmt.Errorf("fetch Assets failed ")
	}

	time.Sleep(4 * time.Second)

	return &pb.ChangeAssetsReply{
		ChangeAssets: &pb.ChangeAssets{
			Gold: value.(*pb.FinalAssets).Gold,
		},
	}, nil
}
