// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package access

import (
	"github.com/talos-systems/talos/internal/app/machined/pkg/runtime"
	"github.com/talos-systems/talos/internal/pkg/cluster"
	"github.com/talos-systems/talos/internal/pkg/provision"
)

// Adapter provides cluster access via provision.Cluster.
type Adapter struct {
	cluster.ConfigClientProvider
	cluster.KubernetesClient
	cluster.APICrashDumper
	cluster.Info
}

type infoWrapper struct {
	clusterInfo provision.ClusterInfo
}

func (wrapper *infoWrapper) Nodes() []string {
	nodes := make([]string, len(wrapper.clusterInfo.Nodes))

	for i := range nodes {
		nodes[i] = wrapper.clusterInfo.Nodes[i].PrivateIP.String()
	}

	return nodes
}

func (wrapper *infoWrapper) NodesByType(t runtime.MachineType) []string {
	var nodes []string

	for _, node := range wrapper.clusterInfo.Nodes {
		if node.Type == t {
			nodes = append(nodes, node.PrivateIP.String())
		}
	}

	return nodes
}

// NewAdapter returns ClusterAccess object from Cluster.
func NewAdapter(clusterInfo provision.Cluster, opts ...provision.Option) *Adapter {
	options := provision.DefaultOptions()

	for _, opt := range opts {
		if err := opt(&options); err != nil {
			panic(err)
		}
	}

	configProvider := cluster.ConfigClientProvider{
		DefaultClient: options.TalosClient,
		TalosConfig:   options.TalosConfig,
	}

	return &Adapter{
		ConfigClientProvider: configProvider,
		KubernetesClient: cluster.KubernetesClient{
			ClientProvider: &configProvider,
			ForceEndpoint:  options.ForceEndpoint,
		},
		APICrashDumper: cluster.APICrashDumper{
			ClientProvider: &configProvider,
			Info:           &infoWrapper{clusterInfo: clusterInfo.Info()},
		},
		Info: &infoWrapper{clusterInfo: clusterInfo.Info()},
	}
}
