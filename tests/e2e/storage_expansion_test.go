/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"fmt"
	"os"

	"github.com/cloudnative-pg/cloudnative-pg/tests"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils/run"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils/storage"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test case for validating storage expansion
// with different storage providers in different k8s environments
var _ = Describe("Verify storage", Label(tests.LabelStorage), func() {
	const (
		sampleFile  = fixturesDir + "/storage_expansion/cluster-storage-expansion.yaml.template"
		clusterName = "storage-expansion"
		level       = tests.Lowest
	)
	// Initializing a global namespace variable to be used in each test case
	var namespace, namespacePrefix string

	BeforeEach(func() {
		if testLevelEnv.Depth < int(level) {
			Skip("Test depth is lower than the amount requested for this test")
		}
	})

	// Gathering default storage class requires to check whether the value
	// of 'allowVolumeExpansion' is true or false
	defaultStorageClass := os.Getenv("E2E_DEFAULT_STORAGE_CLASS")

	Context("can be expanded", func() {
		BeforeEach(func() {
			// Initializing namespace variable to be used in test case
			namespacePrefix = "storage-expansion-true"
			// Extracting bool value of AllowVolumeExpansion
			allowExpansion, err := storage.GetStorageAllowExpansion(
				env.Ctx, env.Client,
				defaultStorageClass,
			)
			Expect(err).ToNot(HaveOccurred())
			if (allowExpansion == nil) || (*allowExpansion == false) {
				Skip(fmt.Sprintf("AllowedVolumeExpansion is false on %v", defaultStorageClass))
			}
		})

		It("expands PVCs via online resize", func() {
			var err error
			// Creating namespace
			namespace, err = env.CreateUniqueTestNamespace(env.Ctx, env.Client, namespacePrefix)
			Expect(err).ToNot(HaveOccurred())
			// Creating a cluster with three nodes
			AssertCreateCluster(namespace, clusterName, sampleFile, env)
			OnlineResizePVC(namespace, clusterName)
		})
	})

	Context("can not be expanded", func() {
		BeforeEach(func() {
			// Initializing namespace variable to be used in test case
			namespacePrefix = "storage-expansion-false"
			// Extracting bool value of AllowVolumeExpansion
			allowExpansion, err := storage.GetStorageAllowExpansion(
				env.Ctx, env.Client,
				defaultStorageClass,
			)
			Expect(err).ToNot(HaveOccurred())
			if (allowExpansion != nil) && (*allowExpansion == true) {
				Skip(fmt.Sprintf("AllowedVolumeExpansion is true on %v", defaultStorageClass))
			}
		})
		It("expands PVCs via offline resize", func() {
			var err error
			// Creating namespace
			namespace, err = env.CreateUniqueTestNamespace(env.Ctx, env.Client, namespacePrefix)
			Expect(err).ToNot(HaveOccurred())
			AssertCreateCluster(namespace, clusterName, sampleFile, env)
			By("update cluster for resizeInUseVolumes as false", func() {
				// Updating cluster with 'resizeInUseVolumes' sets to 'false' in storage.
				// Check if operator does not return error
				Eventually(func() error {
					_, _, err = run.Unchecked("kubectl patch cluster " + clusterName + " -n " + namespace +
						" -p '{\"spec\":{\"storage\":{\"resizeInUseVolumes\":false}}}' --type=merge")
					if err != nil {
						return err
					}
					return nil
				}, 60, 5).Should(Succeed())
			})
			OfflineResizePVC(namespace, clusterName, 600)
		})
	})
})
