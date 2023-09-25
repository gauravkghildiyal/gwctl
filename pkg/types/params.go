package types

import (
	"bytes"
	"context"
	"io"
	"testing"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gauravkghildiyal/gwctl/pkg/common"
	"github.com/gauravkghildiyal/gwctl/pkg/policymanager"
)

type Params struct {
	Client          client.Client
	DC              dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
	PolicyManager   *policymanager.PolicyManager
	Out             io.Writer
}

func MustParamsForTest(t *testing.T, fakeClients *common.FakeClients) *Params {
	policyManager := policymanager.New(fakeClients.DC)
	if err := policyManager.Init(context.Background()); err != nil {
		t.Fatalf("failed to initialize PolicyManager: %v", err)
	}
	return &Params{
		Client:          fakeClients.Client,
		DC:              fakeClients.DC,
		DiscoveryClient: fakeClients.DiscoveryClient,
		PolicyManager:   policyManager,
		Out:             &bytes.Buffer{},
	}
}
