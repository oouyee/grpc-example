package main

import (
	"context"
	"fmt"
	"grpc-example/pb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	fmt.Println(res)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res1, err1 := client.ChangeAssets(ctx, &pb.ChangeAssetsArgs{
		UID:  101,
		Gold: 5000,
	})
	fmt.Println(res1, err1)

	res2, err2 := client.FetchAssets(context.Background(), &pb.FetchAssetsArgs{
		UID: 101,
	})
	fmt.Println(res2, err2)

	time.Sleep(5 * time.Second)

	res3, err3 := client.FetchAssets(context.Background(), &pb.FetchAssetsArgs{
		UID: 101,
	})
	fmt.Println(res3, err3)

}
