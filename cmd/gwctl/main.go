package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/gauravkghildiyal/gwctl/pkg/cmd"
	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"github.com/spf13/cobra"
	cobraflag "github.com/spf13/pflag"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	cobraflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = path.Join(os.Getenv("HOME"), ".kube/config")
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to get restConfig from BuildConfigFromFlags: %v", err))
	}

	client, err := client.New(restConfig, client.Options{})
	if err != nil {
		panic(fmt.Sprintf("Error initializing Kubernetes client: %v", err))
	}
	gatewayv1alpha2.AddToScheme(client.Scheme())
	gatewayv1beta1.AddToScheme(client.Scheme())

	dc := dynamic.NewForConfigOrDie(restConfig)

	policyManager := policymanager.New(dc)
	if err := policyManager.Init(context.Background()); err != nil {
		panic(err)
	}

	params := &types.Params{
		Client:          client,
		DC:              dc,
		DiscoveryClient: discovery.NewDiscoveryClientForConfigOrDie(restConfig),
		PolicyManager:   policyManager,
		Out:             os.Stdout,
	}

	rootCmd := &cobra.Command{
		Use: "gwctl",
	}
	rootCmd.AddCommand(cmd.NewGetCommand(params))
	rootCmd.AddCommand(cmd.NewDescribeCommand(params))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
