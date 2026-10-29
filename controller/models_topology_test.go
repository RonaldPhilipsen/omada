package controller

import "testing"

func TestTopology(t *testing.T) {
	topo := NewTopology(loadFixture(t, "topology.json"))
	roots := topo.Roots()
	if len(roots) != 1 {
		t.Fatalf("roots: %d", len(roots))
	}
	gw := roots[0]
	if gw.Type() != "gateway" || gw.MAC() != "AA:BB:CC:DD:EE:01" || gw.Name() != "Main Gateway" {
		t.Error("gateway node")
	}
	if gw.Model() == nil || *gw.Model() != "EAC2401" {
		t.Errorf("model: %v", gw.Model())
	}
	if gw.IPAddress() == nil || *gw.IPAddress() != "192.168.1.1" {
		t.Errorf("ip: %v", gw.IPAddress())
	}
	if gw.Status() != DeviceStatusConnected || gw.StatusCategory() != DeviceStatusCategoryConnected {
		t.Error("status")
	}
	if gw.Disconnected() || gw.ClientCount() != 12 {
		t.Error("counters")
	}

	// Gateway root has no uplink info.
	if gw.UpstreamPort() != nil || gw.OwnUplinkPort() != nil {
		t.Error("root has no uplink")
	}
	if gw.UpstreamLinkSpeed() != LinkSpeedUnknown || gw.UpstreamDuplex() != LinkDuplexUnknown {
		t.Error("root uplink defaults")
	}

	wans := gw.WANPorts()
	if len(wans) != 2 {
		t.Fatalf("wan ports: %d", len(wans))
	}
	wan := wans[0]
	if wan.Port() != "1" || wan.Name() != "WAN" {
		t.Errorf("wan: %v %v", wan.Port(), wan.Name())
	}
	if wan.LinkStatus() != LinkStatusLinkUp || wan.LinkSpeed() != LinkSpeedSpeed1G || wan.Duplex() != LinkDuplexFull {
		t.Error("wan link")
	}
	if !wan.WANConnected() || !wan.OnlineDetection() {
		t.Error("wan connectivity")
	}
	wan2 := wans[1]
	if wan2.Name() != "2" {
		t.Errorf("wan name falls back to port: %q", wan2.Name())
	}
	if wan2.LinkStatus() != LinkStatusUnknown || wan2.LinkSpeed() != LinkSpeedUnknown || wan2.Duplex() != LinkDuplexUnknown {
		t.Error("wan2 enum defaults")
	}
	if wan2.WANConnected() || wan2.OnlineDetection() {
		t.Error("wan2 must be offline")
	}

	children := gw.Successors()
	if len(children) != 2 {
		t.Fatalf("children: %d", len(children))
	}
	sw := children[0]
	if sw.Type() != "switch" || sw.ClientCount() != 8 {
		t.Error("switch child")
	}
	if sw.UpstreamPort() == nil || *sw.UpstreamPort() != "1" {
		t.Errorf("upstream port: %v", sw.UpstreamPort())
	}
	if sw.OwnUplinkPort() == nil || *sw.OwnUplinkPort() != "8" {
		t.Errorf("own uplink port: %v", sw.OwnUplinkPort())
	}
	if sw.UpstreamLinkSpeed() != LinkSpeedSpeed1G || sw.UpstreamDuplex() != LinkDuplexFull {
		t.Error("uplink link")
	}

	leaf := children[1]
	if leaf.Type() != "ap" || !leaf.Disconnected() {
		t.Error("ap child")
	}
	if leaf.Model() != nil || leaf.IPAddress() == nil {
		t.Error("ap model/ip")
	}
	if leaf.Status() != DeviceStatusDisconnected {
		t.Errorf("ap status: %v", leaf.Status())
	}
	if leaf.UpstreamPort() == nil || *leaf.UpstreamPort() != "3" {
		t.Errorf("ap upstream port: %v", leaf.UpstreamPort())
	}
	if leaf.OwnUplinkPort() != nil {
		t.Error("ap has no own uplink port")
	}
	if leaf.UpstreamLinkSpeed() != LinkSpeedSpeedAuto || leaf.UpstreamDuplex() != LinkDuplexAuto {
		t.Error("ap uplink defaults")
	}
	if len(leaf.Successors()) != 0 {
		t.Error("ap leaf has no successors")
	}

	grandchild := sw.Successors()
	if len(grandchild) != 1 || grandchild[0].Type() != "omada controller" {
		t.Fatalf("grandchild: %+v", grandchild)
	}
}
