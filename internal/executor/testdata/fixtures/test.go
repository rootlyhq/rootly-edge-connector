package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("Go: %s\n", os.Getenv("REC_PARAM_MESSAGE"))
}
