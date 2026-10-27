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

package amifamily

import (
	"fmt"
	"strings"

	"github.com/awslabs/operatorpkg/serrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
)

// HugePagesFromNodeClaim returns the hugepages that Karpenter chose for the NodeClaim, from its flexible capacity
// annotation. The result maps a page size in kibibytes to a number of pages. It returns nil when the NodeClaim has
// no flexible capacity. It returns an error for a flexible resource that is not hugepages, because this provider
// can only allocate hugepages.
func HugePagesFromNodeClaim(nodeClaim *karpv1.NodeClaim) (map[int64]int64, error) {
	flexible, err := cloudprovider.FlexibleCapacityFromNodeClaim(nodeClaim)
	if err != nil {
		return nil, err
	}
	if len(flexible) == 0 {
		return nil, nil
	}
	hugePages := map[int64]int64{}
	for name, quantity := range flexible {
		size, ok := strings.CutPrefix(string(name), corev1.ResourceHugePagesPrefix)
		if !ok {
			return nil, serrors.Wrap(fmt.Errorf("flexible resource is not supported, only hugepages are"), "resource", name)
		}
		pageSize, err := resource.ParseQuantity(size)
		if err != nil {
			return nil, serrors.Wrap(fmt.Errorf("parsing hugepage size, %w", err), "resource", name)
		}
		// The kernel names each hugepage pool by its size in kibibytes.
		if pageSize.Value() < 1024 || pageSize.Value()%1024 != 0 {
			return nil, serrors.Wrap(fmt.Errorf("hugepage size is not a whole number of kibibytes"), "resource", name)
		}
		if quantity.Value() <= 0 || quantity.Value()%pageSize.Value() != 0 {
			return nil, serrors.Wrap(fmt.Errorf("hugepages amount is not a positive whole number of pages"), "resource", name, "quantity", quantity.String())
		}
		hugePages[pageSize.Value()/1024] = quantity.Value() / pageSize.Value()
	}
	return hugePages, nil
}
