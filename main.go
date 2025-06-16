// In file: main.go
package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"terraform-provider-flashblade/internal/provider" // NOTE: Adjust if your module path is different
)

//go:generate tfplugindocs

func main() {
	// FIX: Change the address to the final, desired address.
	// The fully qualified format is "registry.terraform.io/<NAMESPACE>/<TYPE>"
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/purestorage/flashblade",
	})
	if err != nil {
		log.Fatal(err)
	}
}
