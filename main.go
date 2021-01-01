package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"path"
	"topigo/globals"
	topigo "topigo/server"
)
import "topigo/config"

// protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative topigo.proto

func main() {

	configEnvPrefix := *flag.String("p", "", "Specifies the prefix for configuration environment variables.")
	configFilePath := *flag.String("c", "config.yml", "Specifies the file path for the config file.")
	flag.BoolVar(&globals.VerboseLogging, "v", globals.VerboseLogging, "Verbose logging.")
	flag.Parse()

	config := config.MakeConfig(configFilePath, configEnvPrefix)
	ensureStorageFoldersAreCreated(config)

	target := fmt.Sprintf("%v:%v", config.Server.Host, config.Server.Port)
	log.Printf("starting server on %v\n", target)

	listener, err := net.Listen("tcp", target)
	if err != nil {
		log.Fatalf("could not start server on %v: %v", target, err)
	}

	server := grpc.NewServer()
	topigoServer := topigo.MakeTopicoServer(config)
	server.RegisterService(&topigo.Topigo_ServiceDesc, &topigoServer)
	if globals.VerboseLogging {
		log.Println("waiting for connections ...")
	}
	server.Serve(listener)
}

func ensureStorageFoldersAreCreated(config config.Config) {
	err := os.MkdirAll(path.Join(config.Storage.Directory, "messages"), os.ModePerm)
	if err != nil {
		log.Fatalf("could not create storage folder 'messages' in %v: %v", config.Storage.Directory, err)
	}
}
