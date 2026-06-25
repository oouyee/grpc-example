package main

import (
	"context"
	"flag"
	"fmt"
	"grpc-example/etcd-dns/register"
	"grpc-example/pb"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"google.golang.org/grpc"
)

const serviceName = "assets-service"

var (
	port   = flag.Int("port", 50051, "服务器端口")
	weight = flag.Int("weight", 10, "服务权重（用于负载均衡）")
)

func main() {
	flag.Parse()

	reg, err := register.NewEtcdRegister([]string{"localhost:2379"})
	if err != nil {
		panic(err)
	}
	defer reg.Close()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	pb.RegisterAssetsServer(s, &Server{})

	serverAddr := fmt.Sprintf("localhost:%d", *port)

	err = reg.Register(&register.ServiceInfo{
		ServiceName: serviceName,
		Address:     serverAddr,
		Metadata: map[string]string{
			"version": "1.0.0",
			"weight":  strconv.Itoa(*weight),
		},
	}, 10)
	if err != nil {
		panic(err)
	}

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
	reg.Unregister(serviceName, serverAddr)
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

	return &pb.ChangeAssetsReply{
		ChangeAssets: &pb.ChangeAssets{
			Gold: value.(*pb.FinalAssets).Gold,
		},
	}, nil
}
