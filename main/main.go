package main

import (
	"gcron"
	"os"
)

func main() {
	gcron.LoadConfig()
	//gcron.StartNewNode("localhost:9090", []string{"localhost:2379", "localhost:22379", "localhost:32379"})
	gcron.StartNewNode(os.Args[1], os.Args[1:])
}
