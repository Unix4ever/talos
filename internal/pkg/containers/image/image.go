// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package image

import (
	"context"
	"fmt"
	"time"

	"github.com/talos-systems/talos/internal/app/machined/pkg/runtime"
	"github.com/talos-systems/talos/pkg/retry"

	"github.com/containerd/containerd"
)

// Pull is a convenience function that wraps the containerd image pull func with
// retry functionality.
func Pull(ctx context.Context, reg runtime.Registries, client *containerd.Client, ref string) (img containerd.Image, err error) {
	resolver := NewResolver(reg)

	err = retry.Exponential(1*time.Minute, retry.WithUnits(1*time.Second)).Retry(func() error {
		if img, err = client.Pull(ctx, ref, containerd.WithPullUnpack, containerd.WithResolver(resolver)); err != nil {
			return retry.ExpectedError(fmt.Errorf("failed to pull image %q: %w", ref, err))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return img, nil
}
