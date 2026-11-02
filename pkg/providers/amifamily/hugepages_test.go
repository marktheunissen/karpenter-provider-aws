/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package amifamily_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"

	"github.com/aws/karpenter-provider-aws/pkg/providers/amifamily"
)

var _ = Describe("HugePagesFromNodeClaim", func() {
	nodeClaimWith := func(annotation string) *karpv1.NodeClaim {
		return &karpv1.NodeClaim{ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Annotations: map[string]string{karpv1.FlexibleCapacityAnnotationKey: annotation},
		}}
	}

	It("should return nil when the NodeClaim has no annotation", func() {
		hugePages, err := amifamily.HugePagesFromNodeClaim(&karpv1.NodeClaim{})
		Expect(err).ToNot(HaveOccurred())
		Expect(hugePages).To(BeNil())
	})
	It("should return nil when the annotation is empty", func() {
		hugePages, err := amifamily.HugePagesFromNodeClaim(nodeClaimWith("{}"))
		Expect(err).ToNot(HaveOccurred())
		Expect(hugePages).To(BeNil())
	})
	It("should convert each hugepages amount to a page count, keyed by page size in kibibytes", func() {
		hugePages, err := amifamily.HugePagesFromNodeClaim(nodeClaimWith(`{"hugepages-2Mi":"8602Mi","hugepages-1Gi":"2Gi"}`))
		Expect(err).ToNot(HaveOccurred())
		Expect(hugePages).To(Equal(map[int64]int64{2048: 4301, 1048576: 2}))
	})
	DescribeTable("should reject annotations that it cannot allocate",
		func(annotation string) {
			_, err := amifamily.HugePagesFromNodeClaim(nodeClaimWith(annotation))
			Expect(err).To(HaveOccurred())
		},
		Entry("invalid JSON", `{"hugepages-2Mi":`),
		Entry("a resource that is not hugepages", `{"example.com/device":"2"}`),
		Entry("an invalid page size", `{"hugepages-abc":"2Mi"}`),
		Entry("a page size smaller than a kibibyte", `{"hugepages-512":"1Ki"}`),
		Entry("an amount that is not a whole number of pages", `{"hugepages-2Mi":"3Mi"}`),
		Entry("a zero amount", `{"hugepages-2Mi":"0"}`),
	)
})
