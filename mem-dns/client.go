package main

import (
	"context"
	"fmt"
	"grpc-example/pb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/resolver/dns"
)

func Run() {
	fmt.Println("Discovering services...")

	target := "discovery:///assets-service"

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewAssetsClient(conn)

	res, err := client.InitAssets(context.Background(), &pb.InitAssetsArgs{
		UID:  101,
		Gold: 1000,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("InitAssets response:", res)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res1, err1 := client.ChangeAssets(ctx, &pb.ChangeAssetsArgs{
		UID:  101,
		Gold: 5000,
	})
	if err1 != nil {
		fmt.Println("ChangeAssets error:", err1)
	} else {
		fmt.Println("ChangeAssets response:", res1)
	}

	res2, err2 := client.FetchAssets(context.Background(), &pb.FetchAssetsArgs{
		UID: 101,
	})
	if err2 != nil {
		panic(err2)
	}
	fmt.Println("FetchAssets response:", res2)
}
