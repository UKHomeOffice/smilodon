package main

import (
	"flag"
	"fmt"

	"github.com/UKHomeOffice/smilodon/pkg/backend"
	"github.com/UKHomeOffice/smilodon/pkg/backend/aws"
	"github.com/UKHomeOffice/smilodon/pkg/controller"
)

type cmdLineOpts struct {
	backend string
}

var (
	opts cmdLineOpts
)

func init() {
	flag.StringVar(&opts.backend, "backend", "", "Backend type")
}

func main() {
	flag.Parse()

	fmt.Printf("Chosen volume backend type: %q.\n", opts.backend)

	b := aws.New()
	c := controller.New(controller.Config{Backend: b})
	c.Run()
}
