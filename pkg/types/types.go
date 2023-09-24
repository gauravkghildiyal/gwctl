package types

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
)

type Params struct {
	Client          client.Client
	DC              dynamic.Interface
	PolicyManager   *policymanager.PolicyManager
	DiscoveryClient *discovery.DiscoveryClient
}
