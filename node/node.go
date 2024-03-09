package node

type Node struct {
	Id           int32
	Address      string
	RoutingTable map[int32]string
}
