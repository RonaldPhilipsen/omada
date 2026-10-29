package controller

// This file ports the topology model classes from upstream
// tplink_omada_client topology.py. The topology endpoint reports the
// physical wiring between managed devices and their nearest recognized
// downstream nodes, including which port on the upstream device each node
// is plugged into.

// Topology is the site's full physical network topology tree, rooted at its
// gateway(s).
type Topology struct {
	data map[string]any
}

// NewTopology builds a Topology from a raw API object.
func NewTopology(data map[string]any) Topology { return Topology{data: data} }

// Roots are the root nodes of the topology tree (typically the site's
// gateway(s)).
func (t Topology) Roots() []TopologyNode {
	items := mapList(t.data, "topologyArray")
	out := make([]TopologyNode, 0, len(items))
	for _, n := range items {
		m, ok := n.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewTopologyNode(m))
	}
	return out
}

// TopologyNode is a single node (device, or a directly-identifiable client)
// in the topology tree.
type TopologyNode struct {
	data map[string]any
}

// NewTopologyNode builds a TopologyNode from a raw API object.
func NewTopologyNode(data map[string]any) TopologyNode {
	return TopologyNode{data: data}
}

// Type is the type of node. Known values include "gateway", "switch", "ap",
// and "omada controller".
func (n TopologyNode) Type() string { return mapStr(n.data, "type") }

// MAC is the MAC address of the node.
func (n TopologyNode) MAC() string { return mapStr(n.data, "mac") }

// Name is the display name of the node, falling back to the MAC.
func (n TopologyNode) Name() string {
	if name := mapStr(n.data, "name"); name != "" {
		return name
	}
	return n.MAC()
}

// Model is the model of the node, if it's a managed Omada device.
func (n TopologyNode) Model() *string { return mapPtrStr(n.data, "model") }

// IPAddress is the IP address of the node, if known.
func (n TopologyNode) IPAddress() *string { return mapPtrStr(n.data, "ip") }

// Status is the status of the node, if it's a managed Omada device.
func (n TopologyNode) Status() DeviceStatus {
	if v := mapPtrInt(n.data, "status"); v != nil {
		return DeviceStatus(*v)
	}
	return DeviceStatusUnknown
}

// StatusCategory is the high-level status of the node.
func (n TopologyNode) StatusCategory() DeviceStatusCategory {
	if v := mapPtrInt(n.data, "statusCategory"); v != nil {
		return DeviceStatusCategory(*v)
	}
	return DeviceStatusCategoryUnknown
}

// Disconnected reports whether the node is currently disconnected.
func (n TopologyNode) Disconnected() bool {
	return mapBool(n.data, "disconnected")
}

// ClientCount is the number of clients attached to this node.
func (n TopologyNode) ClientCount() int { return mapInt(n.data, "clientCount") }

func (n TopologyNode) wiredUpInfo() map[string]any {
	m, _ := mapMap(n.data, "wiredUpInfo")
	return m
}

// UpstreamPort is the port number, on the parent node, that this node is
// plugged into.
func (n TopologyNode) UpstreamPort() *string {
	up, ok := mapMap(n.wiredUpInfo(), "upLinkPort")
	if !ok {
		return nil
	}
	return mapPtrStr(up, "port")
}

// OwnUplinkPort is the port number, on this node's own hardware, used for
// its uplink (switches only).
func (n TopologyNode) OwnUplinkPort() *string {
	up, ok := mapMap(n.wiredUpInfo(), "port")
	if !ok {
		return nil
	}
	return mapPtrStr(up, "port")
}

// UpstreamLinkSpeed is the negotiated link speed between this node and its
// parent.
func (n TopologyNode) UpstreamLinkSpeed() LinkSpeed {
	if v := mapPtrInt(n.wiredUpInfo(), "linkSpeed"); v != nil {
		return LinkSpeed(*v)
	}
	return LinkSpeedUnknown
}

// UpstreamDuplex is the negotiated duplex mode between this node and its
// parent.
func (n TopologyNode) UpstreamDuplex() LinkDuplex {
	if v := mapPtrInt(n.wiredUpInfo(), "duplex"); v != nil {
		return LinkDuplex(*v)
	}
	return LinkDuplexUnknown
}

// Successors are the nodes plugged into this node.
func (n TopologyNode) Successors() []TopologyNode {
	items := mapList(n.data, "successors")
	out := make([]TopologyNode, 0, len(items))
	for _, s := range items {
		m, ok := s.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewTopologyNode(m))
	}
	return out
}

// WANPorts are the WAN port health summaries, for gateway nodes.
func (n TopologyNode) WANPorts() []TopologyWanPort {
	items := mapList(n.data, "wanPorts")
	out := make([]TopologyWanPort, 0, len(items))
	for _, p := range items {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, NewTopologyWanPort(m))
	}
	return out
}

// TopologyWanPort is a summary WAN port health entry, as reported on a
// gateway node in the topology tree.
type TopologyWanPort struct {
	data map[string]any
}

// NewTopologyWanPort builds a TopologyWanPort from a raw API object.
func NewTopologyWanPort(data map[string]any) TopologyWanPort {
	return TopologyWanPort{data: data}
}

// Port is the port number.
func (w TopologyWanPort) Port() string { return mapStr(w.data, "port") }

// Name is the port name, falling back to the port number.
func (w TopologyWanPort) Name() string {
	if name := mapStr(w.data, "name"); name != "" {
		return name
	}
	return w.Port()
}

// LinkStatus is the low-level connectivity status of the link.
func (w TopologyWanPort) LinkStatus() LinkStatus {
	if v := mapPtrInt(w.data, "status"); v != nil {
		return LinkStatus(*v)
	}
	return LinkStatusUnknown
}

// LinkSpeed is the established link speed of the port.
func (w TopologyWanPort) LinkSpeed() LinkSpeed {
	if v := mapPtrInt(w.data, "linkSpeed"); v != nil {
		return LinkSpeed(*v)
	}
	return LinkSpeedUnknown
}

// Duplex is the actual duplex mode of the port.
func (w TopologyWanPort) Duplex() LinkDuplex {
	if v := mapPtrInt(w.data, "duplex"); v != nil {
		return LinkDuplex(*v)
	}
	return LinkDuplexUnknown
}

// WANConnected reports whether the port is connected to the internet/WAN.
func (w TopologyWanPort) WANConnected() bool {
	return mapInt(w.data, "internetState") != 0
}

// OnlineDetection reports whether regular internet ping tests are working.
func (w TopologyWanPort) OnlineDetection() bool {
	return mapInt(w.data, "onlineDetection") != 0
}
