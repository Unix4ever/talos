// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// +build integration_api

package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/talos-systems/talos/internal/integration/base"
	"github.com/talos-systems/talos/pkg/client"
	"github.com/talos-systems/talos/pkg/retry"
)

type RebootSuite struct {
	base.APISuite

	ctx       context.Context
	ctxCancel context.CancelFunc
}

// SuiteName ...
func (suite *RebootSuite) SuiteName() string {
	return "api.RebootSuite"
}

// SetupTest ...
func (suite *RebootSuite) SetupTest() {
	if testing.Short() {
		suite.T().Skip("skipping in short mode")
	}

	// make sure we abort at some point in time, but give enough room for reboots
	suite.ctx, suite.ctxCancel = context.WithTimeout(context.Background(), 30*time.Minute)
}

// TearDownTest ...
func (suite *RebootSuite) TearDownTest() {
	suite.ctxCancel()
}

// TestRebootNodeByNode reboots cluster node by node, waiting for health between reboots.
func (suite *RebootSuite) TestRebootNodeByNode() {
	if !suite.Capabilities().SupportsReboot {
		suite.T().Skip("cluster doesn't support reboots")
	}

	nodes := suite.DiscoverNodes()
	suite.Require().NotEmpty(nodes)

	for _, node := range nodes {
		suite.T().Log("rebooting node", node)

		suite.AssertRebooted(suite.ctx, node, func(nodeCtx context.Context) error {
			return suite.Client.Reboot(nodeCtx)
		}, 10*time.Minute)
	}
}

// TestRebootAllNodes reboots all cluster nodes at the same time.
func (suite *RebootSuite) TestRebootAllNodes() {
	if !suite.Capabilities().SupportsReboot {
		suite.T().Skip("cluster doesn't support reboots")
	}

	// offset to account for uptime measuremenet inaccuracy
	const offset = 2 * time.Second

	nodes := suite.DiscoverNodes()
	suite.Require().NotEmpty(nodes)

	errCh := make(chan error, len(nodes))

	var initialUptime sync.Map

	for _, node := range nodes {
		go func(node string) {
			errCh <- func() error {
				nodeCtx := client.WithNodes(suite.ctx, node)

				// read uptime before reboot
				uptimeBefore, err := suite.ReadUptime(nodeCtx)
				if err != nil {
					return fmt.Errorf("error reading initial uptime (node %q): %w", node, err)
				}

				initialUptime.Store(node, uptimeBefore)
				return nil
			}()
		}(node)
	}

	for range nodes {
		suite.Require().NoError(<-errCh)
	}

	rebootTimestamp := time.Now()

	allNodesCtx := client.WithNodes(suite.ctx, nodes...)

	suite.Require().NoError(suite.Client.Reboot(allNodesCtx))

	for _, node := range nodes {
		go func(node string) {
			errCh <- func() error {
				uptimeBeforeInterface, ok := initialUptime.Load(node)
				if !ok {
					return fmt.Errorf("uptime record not found for %q", node)
				}

				uptimeBefore := uptimeBeforeInterface.(time.Duration) //nolint: errcheck

				nodeCtx := client.WithNodes(suite.ctx, node)

				return retry.Constant(10 * time.Minute).Retry(func() error {
					requestCtx, requestCtxCancel := context.WithTimeout(nodeCtx, 5*time.Second)
					defer requestCtxCancel()

					elapsed := time.Since(rebootTimestamp) - offset

					uptimeAfter, err := suite.ReadUptime(requestCtx)
					if err != nil {
						// API might be unresponsive during reboot
						return retry.ExpectedError(fmt.Errorf("error reading uptime for node %q: %w", node, err))
					}

					// uptime of the node before it actually reboots still goes up linearly
					// so we can safely add elapsed time here
					if uptimeAfter >= uptimeBefore+elapsed {
						// uptime should go down after reboot
						return retry.ExpectedError(fmt.Errorf("uptime didn't go down for node %q: before %s + %s, after %s", node, uptimeBefore, elapsed, uptimeAfter))
					}

					return nil
				})
			}()
		}(node)
	}

	for range nodes {
		suite.Assert().NoError(<-errCh)
	}

	if suite.Cluster != nil {
		// without cluster state we can't do deep checks, but basic reboot test still works
		// NB: using `ctx` here to have client talking to init node by default
		suite.AssertClusterHealthy(suite.ctx)
	}
}

func init() {
	allSuites = append(allSuites, new(RebootSuite))
}
