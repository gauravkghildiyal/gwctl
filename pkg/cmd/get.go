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

func NewGetCommand(params *types.Params) *cobra.Command {
	flags := &getFlags{}

	cmd := &cobra.Command{
		Use:   "get {policies|policycrds|httproutes}",
		Short: "Display one or many resources",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runGet(args, params, flags)
		},
	}
	cmd.Flags().StringVarP(&flags.namespace, "namespace", "n", "default", "")
	cmd.Flags().BoolVarP(&flags.allNamespaces, "all-namespaces", "A", false, "If present, list requested resources from all namespaces.")

	return cmd
}

func runGet(args []string, params *types.Params, flags *getFlags) {
	kind := args[0]
	ns := flags.namespace
	if flags.allNamespaces {
		ns = ""
	}

	switch kind {
	case "policy", "policies":
		list := params.PolicyManager.GetPolicies()
		policies.Print(params, list)
	case "policycrds":
		list := params.PolicyManager.GetCRDs()
		policies.PrintCRDs(params, list)
	case "httproute", "httproutes":
		list, err := httproutes.List(context.TODO(), params, ns)
		if err != nil {
			panic(err)
		}
		httproutes.Print(list)
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized RESOURCE_TYPE\n")
	}
}
