package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"google.golang.org/grpc"

	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

func main() {
	addr := flag.String("addr", ":8400", "service address")
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewLogClient(conn)
	ctx := context.Background()
	res, err := client.GetServers(ctx, &pb.GetServersRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("servers:")
	for _, server := range res.Servers {
		fmt.Printf("\t- %v\n", server)
	}
}
