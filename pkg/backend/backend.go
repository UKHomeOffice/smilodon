package backend

// Backend TODO
type Backend interface {
	Run()
	// VolumeAttacher
	// NetworkAttacher
}

// TODO
// // VolumeAttacher can attach / detach a volume to a node.
// type VolumeAttacher interface {
// 	VolumeAttach()
// 	VolumeWaitForAttach()
// 	VolumeDetach()
// }

// // NetworkAttacher can attach / detach a network to a node.
// type NetworkAttacher interface {
// 	NetworkAttach()
// 	NetworkWaitForAttach()
// 	NetworkDetach()
// }
