package nodes

import (
	"bytes"
	"github.com/chromedp/cdproto/cdp"
	"strings"
)

func NodeText(node *cdp.Node) string {
	var buffer bytes.Buffer
	for _, n := range node.Children {
		switch n.NodeType {
		case cdp.NodeTypeText:
			buffer.WriteString(n.NodeValue)
		case cdp.NodeTypeElement:
			buffer.WriteString(NodeText(n))
		}
	}
	return strings.TrimSpace(buffer.String())
}
