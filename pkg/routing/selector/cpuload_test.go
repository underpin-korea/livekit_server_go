package selector_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/underpin-korea/protocol/livekit"

	"github.com/underpin-korea/livekit_server_go/pkg/routing/selector"
)

func TestCPULoadSelector_SelectNode(t *testing.T) {
	sel := selector.CPULoadSelector{CPULoadLimit: 0.8}

	var nodes []*livekit.Node
	_, err := sel.SelectNode(nodes)
	require.Error(t, err, "should error no available nodes")

	// Select a node with high load when no nodes with low load are available
	nodes = []*livekit.Node{nodeLoadHigh}
	if _, err := sel.SelectNode(nodes); err != nil {
		t.Error(err)
	}

	// Select a node with low load when available
	nodes = []*livekit.Node{nodeLoadLow, nodeLoadHigh}
	for i := 0; i < 5; i++ {
		node, err := sel.SelectNode(nodes)
		if err != nil {
			t.Error(err)
		}
		if node != nodeLoadLow {
			t.Error("selected the wrong node")
		}
	}
}
