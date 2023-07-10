package cmd

import (
	"context"
	"fmt"
	"os"

	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/gatewayclasses"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/gateways"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/httproutes"
	"github.com/gauravkghildiyal/gwctl/pkg/resources/policies"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"github.com/spf13/cobra"
)

type describeFlags struct {
	namespace     string
	allNamespaces bool
}

func NewDescribeCommand(clients *types.Clients) *cobra.Command {
	flags := &describeFlags{}

	cmd := &cobra.Command{
		Use:   "describe {policies|httproutes|gateways|gatewayclasses} RESOURCE_NAME",
		Short: "Show details of a specific resource or group of resources",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			runDescribe(args, clients, flags)
		},
	}
	cmd.Flags().StringVarP(&flags.namespace, "namespace", "n", "default", "")
	cmd.Flags().BoolVarP(&flags.allNamespaces, "all-namespaces", "A", false, "If present, list requested resources from all namespaces.")

	return cmd
}

func runDescribe(args []string, clients *types.Clients, flags *describeFlags) {
	kind := args[0]
	ns := flags.namespace
	if flags.allNamespaces {
		ns = ""
	}

	switch kind {
	case "policy", "policies":
		policyList, err := policies.List(context.TODO(), clients, ns)
		if err != nil {
			panic(err)
		}
		policies.PrintDescribeView(clients, policyList)
	case "httproute", "httproutes":
		var httpRoutes []gatewayv1alpha2.HTTPRoute
		if len(args) == 1 {
			var err error
			httpRoutes, err = httproutes.List(context.TODO(), clients, ns)
			if err != nil {
				panic(err)
			}
		} else {
			httpRoute, err := httproutes.Get(context.TODO(), clients, ns, args[1])
			if err != nil {
				panic(err)
			}
			httpRoutes = []gatewayv1alpha2.HTTPRoute{httpRoute}
		}
		httproutes.PrintDescribeView(context.TODO(), clients, httpRoutes)
	case "gateway", "gateways":
		var gws []gatewayv1alpha2.Gateway
		if len(args) == 1 {
			var err error
			gws, err = gateways.List(context.TODO(), clients, ns)
			if err != nil {
				panic(err)
			}
		} else {
			gw, err := gateways.Get(context.TODO(), clients, ns, args[1])
			if err != nil {
				panic(err)
			}
			gws = []gatewayv1alpha2.Gateway{gw}
		}
		gateways.PrintDescribeView(context.TODO(), clients, gws)
	case "gatewayclass", "gatewayclasses":
		var gwClasses []gatewayv1alpha2.GatewayClass
		if len(args) == 1 {
			var err error
			gwClasses, err = gatewayclasses.List(context.TODO(), clients)
			if err != nil {
				panic(err)
			}
		} else {
			gwc, err := gatewayclasses.Get(context.TODO(), clients, args[1])
			if err != nil {
				panic(err)
			}
			gwClasses = []gatewayv1alpha2.GatewayClass{gwc}
		}
		gatewayclasses.PrintDescribeView(context.TODO(), clients, gwClasses)
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized RESOURCE_TYPE\n")
	}
}
