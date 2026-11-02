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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HugePages", func() {
	It("should return no script when there are no hugepages", func() {
		Expect(Options{}.hugePagesScript()).To(BeEmpty())
		Expect(Options{HugePages: map[int64]int64{}}.hugePagesScript()).To(BeEmpty())
	})
	It("should allocate each page size, smallest first", func() {
		script := Options{HugePages: map[int64]int64{1048576: 2, 2048: 4301}}.hugePagesScript()
		Expect(script).To(HavePrefix("#!/bin/bash -e\n"))
		Expect(script).To(ContainSubstring(allocateHugePagesFunction))
		Expect(script).To(HaveSuffix("allocate_hugepages 2048 4301\nallocate_hugepages 1048576 2\n"))
	})
	It("should write the 2Mi page count to sysctl.d, so it stays after a reboot", func() {
		script := Options{HugePages: map[int64]int64{1048576: 2, 2048: 4301}}.hugePagesScript()
		Expect(script).To(ContainSubstring("cat <<EOF >/etc/sysctl.d/zzzz-hugepages.conf\nvm.nr_hugepages = 4301\nEOF\n"))
	})
	It("should not write sysctl.d when there are no 2Mi pages", func() {
		script := Options{HugePages: map[int64]int64{1048576: 2}}.hugePagesScript()
		Expect(script).ToNot(ContainSubstring("sysctl.d"))
		Expect(script).To(HaveSuffix("allocate_hugepages 1048576 2\n"))
	})
	It("should give the same script for the same hugepages", func() {
		hugePages := map[int64]int64{2048: 1, 1048576: 1, 64: 1, 32768: 1}
		first := Options{HugePages: hugePages}.hugePagesScript()
		for range 10 {
			Expect(Options{HugePages: hugePages}.hugePagesScript()).To(Equal(first))
		}
	})
})
