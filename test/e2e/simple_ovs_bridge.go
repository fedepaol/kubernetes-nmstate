package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Simple OVS bridge", func() {
	Context("when desiredState is configured", func() {
		Context("with an ovs bridge up", func() {
			BeforeEach(func() {
				updateDesiredState(ovsBrUp(bridge1))
			})
			AfterEach(func() {
				updateDesiredState(ovsBrAbsent(bridge1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the ovs bridge at currentState", func() {
				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(SatisfyAll(
						HaveKeyWithValue("name", bridge1),
						HaveKeyWithValue("type", "ovs-bridge"),
						HaveKeyWithValue("state", "up"),
					)))
				}
			})
		})
	})
})