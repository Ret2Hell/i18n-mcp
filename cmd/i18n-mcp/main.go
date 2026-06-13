package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/cli"
)

func main() {
	ctx := context.Background()
	if err := cli.Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
