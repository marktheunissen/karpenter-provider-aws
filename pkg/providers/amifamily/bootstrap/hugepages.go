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

package bootstrap

import (
	"fmt"
	"sort"
	"strings"

	"github.com/samber/lo"
)

// allocateHugePagesFunction is a shell function that allocates hugepages of one size. Its arguments are the page
// size in kibibytes and the number of pages. If the kernel allocates fewer pages than asked, the function frees
// cached memory, compacts memory and tries once more. It prints the result in both cases.
const allocateHugePagesFunction = `allocate_hugepages() {
  local file="/sys/kernel/mm/hugepages/hugepages-${1}kB/nr_hugepages"
  echo "$2" > "$file"
  if [ "$(cat "$file")" -lt "$2" ]; then
    sync
    echo 3 > /proc/sys/vm/drop_caches
    echo 1 > /proc/sys/vm/compact_memory
    echo "$2" > "$file"
  fi
  echo "hugepages-${1}kB: requested $2 pages, allocated $(cat "$file") pages"
}
`

// hugePagesSysctlFile holds the 2Mi page count, so that it is set again on every boot. The user data runs only on
// the first boot, but systemd-sysctl reads /etc/sysctl.d on every boot, before kubelet starts. The name sorts after
// other files, so its value wins. It is the same file that custom user data may write, so Karpenter's value replaces
// that one.
const hugePagesSysctlFile = "/etc/sysctl.d/zzzz-hugepages.conf"

// defaultHugePageSize is the default hugepage size in kibibytes on x86_64 and arm64 with 4 KiB base pages. The
// vm.nr_hugepages sysctl sets the page count for this size only.
const defaultHugePageSize = 2048

// hugePagesScript returns a shell script that allocates the hugepages in o.HugePages. It returns an empty string
// when there are no hugepages to allocate. The script must run before kubelet starts, because kubelet reads the
// hugepages count only when it starts.
//
// The script writes the 2Mi page count to hugePagesSysctlFile, so that the count stays after a reboot. Other page
// sizes have no sysctl, so their counts are lost when the node reboots.
func (o Options) hugePagesScript() string {
	if len(o.HugePages) == 0 {
		return ""
	}
	var script strings.Builder
	script.WriteString("#!/bin/bash -e\n")
	script.WriteString("# Allocate the hugepages that Karpenter chose for this node.\n")
	if pages, ok := o.HugePages[defaultHugePageSize]; ok {
		fmt.Fprintf(&script, "cat <<EOF >%s\nvm.nr_hugepages = %d\nEOF\n", hugePagesSysctlFile, pages)
	}
	script.WriteString(allocateHugePagesFunction)
	// Sort the page sizes so that the same input always gives the same script, and so the same launch template.
	sizes := lo.Keys(o.HugePages)
	sort.Slice(sizes, func(i, j int) bool { return sizes[i] < sizes[j] })
	for _, size := range sizes {
		fmt.Fprintf(&script, "allocate_hugepages %d %d\n", size, o.HugePages[size])
	}
	return script.String()
}
