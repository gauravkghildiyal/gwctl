package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/policies"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"github.com/spf13/cobra"
)

type getFlags struct {
	namespace     string
	allNamespaces bool
}

func NewGetCommand(clients *types.Clients) *cobra.Command {
	flags := &getFlags{}

	cmd := &cobra.Command{
		Use:   "get {policies|policycrds}",
		Short: "Display one or many resources",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runGet(args, clients, flags)
		},
	}
	cmd.Flags().StringVarP(&flags.namespace, "namespace", "n", "default", "")
	cmd.Flags().BoolVarP(&flags.allNamespaces, "all-namespaces", "A", false, "If present, list requested resources from all namespaces.")

	return cmd
}

func runGet(args []string, clients *types.Clients, flags *getFlags) {
	kind := args[0]
	ns := flags.namespace
	if flags.allNamespaces {
		ns = ""
	}

	switch kind {
	case "policy", "policies":
		p, err := policies.List(context.TODO(), clients, ns)
		if err != nil {
			panic(err)
		}
		policies.Print(p)
	case "policycrds":
		p, err := policies.ListCRDs(context.TODO(), clients)
		if err != nil {
			panic(err)
		}
		policies.PrintCRDs(p)
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized RESOURCE_TYPE\n")
	}
}
