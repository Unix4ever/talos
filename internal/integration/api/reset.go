// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// +build integration_api

package api

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/talos-systems/talos/internal/integration/base"
)

type ResetSuite struct {
	base.APISuite

	ctx       context.Context
	ctxCancel context.CancelFunc
}

// SuiteName ...
func (suite *ResetSuite) SuiteName() string {
	return "api.ResetSuite"
}

// SetupTest ...
func (suite *ResetSuite) SetupTest() {
	if testing.Short() {
		suite.T().Skip("skipping in short mode")
	}

	// make sure we abort at some point in time, but give enough room for Resets
	suite.ctx, suite.ctxCancel = context.WithTimeout(context.Background(), 30*time.Minute)
}

// TearDownTest ...
func (suite *ResetSuite) TearDownTest() {
	suite.ctxCancel()
}

// TestResetNodeByNode Resets cluster node by node, waiting for health between Resets.
func (suite *ResetSuite) TestResetNodeByNode() {
	if !suite.Capabilities().SupportsReboot {
		suite.T().Skip("cluster doesn't support reboot (and reset)")
	}

	if suite.Cluster == nil {
		suite.T().Skip("without full cluster state reset test is not reliable (can't wait for cluster readiness in between resets)")
	}

	nodes := suite.DiscoverNodes()
	suite.Require().NotEmpty(nodes)

	sort.Strings(nodes)

	for i, node := range nodes {
		if i == 0 {
			// first node should be init node, due to bug with etcd cluster build for init node
			// and Reset(), skip resetting first node
			suite.T().Log("Skipping init node", node, "due to known issue with etcd")
			continue
		}

		suite.T().Log("Resetting node", node)

		// uptime should go down after Reset, as it reboots the node
		suite.AssertRebooted(suite.ctx, node, func(nodeCtx context.Context) error {
			// force reboot after reset, as this is the only mode we can test
			return suite.Client.Reset(nodeCtx, true, true)
		}, 10*time.Minute)

		// TODO: there is no good way to assert that node was reset and disk contents were really wiped
	}
}

func init() {
	allSuites = append(allSuites, new(ResetSuite))
}
