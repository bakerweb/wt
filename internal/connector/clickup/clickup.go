// Package clickup provides a placeholder connector for ClickUp.
package clickup

import (
	"context"
	"fmt"

	"github.com/bakerweb/wt/internal/connector"
)

// Client is a placeholder for the ClickUp connector.
type Client struct{}

func New() *Client                                                    { return &Client{} }
func (c *Client) Name() string                                       { return "clickup" }
func (c *Client) GetTicket(ctx context.Context, key string) (*connector.Ticket, error) {
	return nil, fmt.Errorf("clickup connector is not yet implemented")
}
func (c *Client) ListAssigned(ctx context.Context) ([]connector.Ticket, error) {
	return nil, fmt.Errorf("clickup connector is not yet implemented")
}
func (c *Client) TransitionTicket(ctx context.Context, key, status string) error {
	return fmt.Errorf("clickup connector is not yet implemented")
}
func (c *Client) Validate(ctx context.Context) error {
	return fmt.Errorf("clickup connector is not yet implemented")
}
