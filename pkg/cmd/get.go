package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/httproutes"
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
		Use:   "get {policies|policycrds|httproutes}",
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
		list, err := policies.List(context.TODO(), clients, ns)
		if err != nil {
			panic(err)
		}
		policies.Print(list)
	case "policycrds":
		list, err := policies.ListCRDs(context.TODO(), clients)
		if err != nil {
			panic(err)
		}
		policies.PrintCRDs(list)
	case "httproute", "httproutes":
		list, err := httproutes.List(context.TODO(), clients, ns)
		if err != nil {
			panic(err)
		}
		httproutes.Print(list)
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized RESOURCE_TYPE\n")
	}
}
