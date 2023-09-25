package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/gauravkghildiyal/gwctl/pkg/resources/backends"
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

func NewDescribeCommand(params *types.Params) *cobra.Command {
	flags := &describeFlags{}

	cmd := &cobra.Command{
		Use:   "describe {policies|httproutes|gateways|gatewayclasses|backends} RESOURCE_NAME",
		Short: "Show details of a specific resource or group of resources",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			runDescribe(args, params, flags)
		},
	}
	cmd.Flags().StringVarP(&flags.namespace, "namespace", "n", "default", "")
	cmd.Flags().BoolVarP(&flags.allNamespaces, "all-namespaces", "A", false, "If present, list requested resources from all namespaces.")

	return cmd
}

func runDescribe(args []string, params *types.Params, flags *describeFlags) {
	kind := args[0]
	ns := flags.namespace
	if flags.allNamespaces {
		ns = ""
	}

	switch kind {
	case "policy", "policies":
		policyList := params.PolicyManager.GetPolicies()
		policies.PrintDescribeView(params, policyList)
	case "httproute", "httproutes":
		var httpRoutes []gatewayv1beta1.HTTPRoute
		if len(args) == 1 {
			var err error
			httpRoutes, err = httproutes.List(context.TODO(), params, ns)
			if err != nil {
				panic(err)
			}
		} else {
			httpRoute, err := httproutes.Get(context.TODO(), params, ns, args[1])
			if err != nil {
				panic(err)
			}
			httpRoutes = []gatewayv1beta1.HTTPRoute{httpRoute}
		}
		httproutes.PrintDescribeView(context.TODO(), params, httpRoutes)
	case "gateway", "gateways":
		var gws []gatewayv1beta1.Gateway
		if len(args) == 1 {
			var err error
			gws, err = gateways.List(context.TODO(), params, ns)
			if err != nil {
				panic(err)
			}
		} else {
			gw, err := gateways.Get(context.TODO(), params, ns, args[1])
			if err != nil {
				panic(err)
			}
			gws = []gatewayv1beta1.Gateway{gw}
		}
		gateways.PrintDescribeView(context.TODO(), params, gws)
	case "gatewayclass", "gatewayclasses":
		var gwClasses []gatewayv1beta1.GatewayClass
		if len(args) == 1 {
			var err error
			gwClasses, err = gatewayclasses.List(context.TODO(), params)
			if err != nil {
				panic(err)
			}
		} else {
			gwc, err := gatewayclasses.Get(context.TODO(), params, args[1])
			if err != nil {
				panic(err)
			}
			gwClasses = []gatewayv1beta1.GatewayClass{gwc}
		}
		gatewayclasses.PrintDescribeView(context.TODO(), params, gwClasses)
	case "backend", "backends":
		var backendsList []unstructured.Unstructured

		resourceType := "service"
		var resourceName string
		if len(args) > 1 {
			resourceName = args[1]
			a, b, ok := strings.Cut(args[1], "/")
			if ok {
				resourceType, resourceName = a, b
			}
		}

		if resourceName == "" {
			var err error
			backendsList, err = backends.List(context.TODO(), params, resourceType, ns)
			if err != nil {
				panic(err)
			}
		} else {
			backend, err := backends.Get(context.TODO(), params, resourceType, ns, resourceName)
			if err != nil {
				panic(err)
			}
			backendsList = []unstructured.Unstructured{backend}
		}
		backends.PrintDescribeView(context.TODO(), params, backendsList)
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized RESOURCE_TYPE\n")
	}
}
