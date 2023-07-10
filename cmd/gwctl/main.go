package main

import (
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/gauravkghildiyal/gwctl/pkg/cmd"
	"github.com/gauravkghildiyal/gwctl/pkg/types"
	"github.com/spf13/cobra"
)

func main() {
	// TODO: Replace proxying with proper authentication through kubeconfig.
	port := rand.Intn(30000) + 30000
	if err := exec.Command("kubectl", "proxy", "-p", fmt.Sprintf("%v", port)).Start(); err != nil {
		panic(err)
	}
	time.Sleep(100 * time.Millisecond)

	restConfig, err := clientcmd.BuildConfigFromFlags(fmt.Sprintf("http://localhost:%v", port), "")
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

	clients := &types.Clients{
		Client: client,
		DC:     dc,
	}

	rootCmd := &cobra.Command{
		Use: "gwctl",
	}
	rootCmd.AddCommand(cmd.NewGetCommand(clients))
	rootCmd.AddCommand(cmd.NewDescribeCommand(clients))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
