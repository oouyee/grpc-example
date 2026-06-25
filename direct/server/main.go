package main

import (
	"context"
	"fmt"
	"grpc-example/pb"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	pb.RegisterAssetsServer(s, &Server{})

	err = s.Serve(lis)
	if err != nil {
		panic(err)
		return
	}
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

	time.Sleep(5 * time.Second)

	cache.Store(args.UID, &pb.FinalAssets{
		Gold: args.GetGold(),
	})

	return &pb.ChangeAssetsReply{
		ChangeAssets: &pb.ChangeAssets{
			Gold: args.GetGold(),
		},
	}, nil
}
