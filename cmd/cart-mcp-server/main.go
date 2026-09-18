// Command cart-mcp-server is a standalone Model Context Protocol server
// exposing a stateful shopping cart — add_item_to_cart and view_cart — over
// stdio. It has no dependency on google.golang.org/adk/v2 at all: matching
// Python's own cart_server.py, an MCP server is a plain, general-purpose
// program any MCP-compliant client can talk to, not something coupled to
// this course's own agent framework. cmd/shopping-agent is one such client.
package main

import (
	"context"
	"log"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// cart holds the server's own state — a mutex from the start, the same
// discipline this repo's own shared-state trackers have followed since
// module-25: a real MCP server can field concurrent tool calls from more
// than one client session.
type cart struct {
	mu    sync.Mutex
	items []string
}

// AddItemArgs is add_item_to_cart's input — its JSON Schema is inferred
// automatically by mcp.AddTool from this struct and its jsonschema tags, no
// hand-written inputSchema needed (unlike Python's own lab).
type AddItemArgs struct {
	Item string `json:"item" jsonschema:"the item to add to the cart"`
}

// AddItemResult is add_item_to_cart's output. Returning this (with a nil
// *mcp.CallToolResult) lets the framework auto-populate the response's
// Content with this value's own JSON text — confirmed in CallToolResult's
// own doc comment.
type AddItemResult struct {
	Status string   `json:"status"`
	Cart   []string `json:"cart"`
}

// ViewCartArgs is view_cart's input — empty, since the tool takes no
// arguments. mcp.AddTool still infers a valid (empty-object) JSON Schema
// for it.
type ViewCartArgs struct{}

// ViewCartResult is view_cart's output.
type ViewCartResult struct {
	Items []string `json:"items"`
}

// addItem appends args.Item to the cart and returns the cart's new
// contents — the direct equivalent of Python's CART.append(item).
func (c *cart) addItem(_ context.Context, _ *mcp.CallToolRequest, args AddItemArgs) (*mcp.CallToolResult, AddItemResult, error) {
	c.mu.Lock()
	c.items = append(c.items, args.Item)
	itemsCopy := append([]string(nil), c.items...)
	c.mu.Unlock()

	log.Printf("[Server]: added %q to the cart", args.Item)
	return nil, AddItemResult{Status: "success", Cart: itemsCopy}, nil
}

// view returns the cart's current contents.
func (c *cart) view(_ context.Context, _ *mcp.CallToolRequest, _ ViewCartArgs) (*mcp.CallToolResult, ViewCartResult, error) {
	c.mu.Lock()
	itemsCopy := append([]string(nil), c.items...)
	c.mu.Unlock()

	log.Printf("[Server]: client viewed the cart (%d item(s))", len(itemsCopy))
	return nil, ViewCartResult{Items: itemsCopy}, nil
}

func main() {
	log.SetFlags(0)
	log.Println("[Server]: Starting Shopping Cart MCP Server...")

	c := &cart{}
	server := mcp.NewServer(&mcp.Implementation{Name: "shopping_cart_mcp_server", Version: "0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_item_to_cart",
		Description: "Adds an item to the shopping cart.",
	}, c.addItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "view_cart",
		Description: "Returns the current contents of the shopping cart.",
	}, c.view)

	log.Println("[Server]: Waiting for a client to connect...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("[Server]: run failed: %v", err)
	}
}
