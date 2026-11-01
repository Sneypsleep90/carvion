package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Sneypsleep90/carvion/internal/adapters/androiddemo"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func main() {
	adapter, err := androiddemo.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sdk.Serve(context.Background(), os.Stdin, os.Stdout, adapter); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
