package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func bondAbsent(bondName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: absent
`, bondName))
}

func brAndBondAbsent(bridgeName string, bondName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: linux-bridge
    state: absent
  - name: %s
    type: bond
    state: absent
`, bridgeName, bondName))
}

func bondUp(bondName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - eth1
      options:
        miimon: '120'
`, bondName))
}

func brWithBondUp(bridgeName string, bondName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
  - name: %s
    type: bond
    state: up
    link-aggregation:
      mode: active-backup
      slaves:
        - eth1
      options:
        miimon: '120'
  - name: %s
    type: linux-bridge
    state: up
    bridge:
      options:
        stp:
          enabled: false
      port:
        - name: %s
`, bondName, bridgeName, bondName))
}

func bondUpWithEth1AndEth2(bondName string) nmstatev1alpha1.State {
	return nmstatev1alpha1.State(fmt.Sprintf(`interfaces:
- name: %s
  type: bond
  state: up
  ipv4:
    address:
    - ip: 10.10.10.10
      prefix-length: 24
    enabled: true
  link-aggregation:
    mode: balance-rr
    options:
      miimon: '140'
    slaves:
    - eth1
    - eth2
`, bondName))
}

var _ = Describe("NodeNetworkState", func() {
	var ()
	Context("when desiredState is configured", func() {
		Context("with a linux bridge up with no ports", func() {
			BeforeEach(func() {
				updateDesiredState(brUpNoPorts(bridge1))
			})
			AfterEach(func() {
				updateDesiredState(brAbsent(bridge1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the linux bridge at currentState with vlan_filtering 1", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					bridgeDescription(node, bridge1).Should(ContainSubstring("vlan_filtering 1"))
				}
			})
		})
		Context("with a linux bridge up", func() {
			BeforeEach(func() {
				updateDesiredState(brUp(bridge1))
			})
			AfterEach(func() {
				updateDesiredState(brAbsent(bridge1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the linux bridge at currentState", func() {
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
					vlansCardinality(node, bridge1).Should(Equal(0))
					getVLANFlagsEventually(node, "eth1", 1).Should(ConsistOf([]string{"PVID", "Egress Untagged"}))
					hasVlans(node, "eth1", 2, 4094).Should(Succeed())
					getVLANFlagsEventually(node, "eth2", 1).Should(ConsistOf([]string{"PVID", "Egress Untagged"}))
					hasVlans(node, "eth2", 2, 4094).Should(Succeed())
				}
			})
		})
		Context("with a active-backup miimon 100 bond interface up", func() {
			BeforeEach(func() {
				updateDesiredState(bondUp(bond1))
			})
			AfterEach(func() {
				updateDesiredState(bondAbsent(bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond interface at currentState", func() {
				var (
					expectedBond = interfaceByName(interfaces(bondUp(bond1)), bond1)
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(SatisfyAll(
						HaveKeyWithValue("name", expectedBond["name"]),
						HaveKeyWithValue("type", expectedBond["type"]),
						HaveKeyWithValue("state", expectedBond["state"]),
						HaveKeyWithValue("link-aggregation", expectedBond["link-aggregation"]),
					)))
				}
			})
		})
		Context("with the bond interface as linux bridge port", func() {
			BeforeEach(func() {
				updateDesiredState(brWithBondUp(bridge1, bond1))
			})
			AfterEach(func() {
				updateDesiredState(brAndBondAbsent(bridge1, bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond in the linux bridge as port at currentState", func() {
				var (
					expectedInterfaces = interfaces(brWithBondUp(bridge1, bond1))
					expectedBond       = interfaceByName(expectedInterfaces, bond1)
					expectedBridge     = interfaceByName(expectedInterfaces, bridge1)
				)
				for _, node := range nodes {
					interfacesForNode(node).Should(SatisfyAll(
						ContainElement(SatisfyAll(
							HaveKeyWithValue("name", expectedBond["name"]),
							HaveKeyWithValue("type", expectedBond["type"]),
							HaveKeyWithValue("state", expectedBond["state"]),
							HaveKeyWithValue("link-aggregation", expectedBond["link-aggregation"]),
						)),
						ContainElement(SatisfyAll(
							HaveKeyWithValue("name", expectedBridge["name"]),
							HaveKeyWithValue("type", expectedBridge["type"]),
							HaveKeyWithValue("state", expectedBridge["state"]),
							HaveKeyWithValue("bridge", HaveKeyWithValue("port",
								ContainElement(HaveKeyWithValue("name", bond1)))),
						))))

					hasVlans(node, bond1, 2, 4094).Should(Succeed())
					vlansCardinality(node, bridge1).Should(Equal(0))
					getVLANFlagsEventually(node, bond1, 1).Should(ConsistOf([]string{"PVID", "Egress Untagged"}))
					vlansCardinality(node, "eth1").Should(Equal(0))
					vlansCardinality(node, "eth2").Should(Equal(0))
				}
			})
		})
		Context("with bond interface that has 2 eths as slaves", func() {
			BeforeEach(func() {
				updateDesiredState(bondUpWithEth1AndEth2(bond1))
			})
			AfterEach(func() {
				updateDesiredState(bondAbsent(bond1))
				for _, node := range nodes {
					interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bond1))
				}
				resetDesiredStateForNodes()
			})
			It("should have the bond interface with 2 slaves at currentState", func() {
				var (
					expectedBond  = interfaceByName(interfaces(bondUpWithEth1AndEth2(bond1)), bond1)
					expectedSpecs = expectedBond["link-aggregation"].(map[string]interface{})
				)

				for _, node := range nodes {
					interfacesForNode(node).Should(ContainElement(SatisfyAll(
						HaveKeyWithValue("name", expectedBond["name"]),
						HaveKeyWithValue("type", expectedBond["type"]),
						HaveKeyWithValue("state", expectedBond["state"]),
						HaveKeyWithValue("link-aggregation", HaveKeyWithValue("mode", expectedSpecs["mode"])),
						HaveKeyWithValue("link-aggregation", HaveKeyWithValue("options", expectedSpecs["options"])),
						HaveKeyWithValue("link-aggregation", HaveKeyWithValue("slaves", ConsistOf([]string{"eth1", "eth2"}))),
					)))
				}
			})
		})
	})
})
