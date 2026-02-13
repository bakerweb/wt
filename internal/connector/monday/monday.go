// Package monday provides a placeholder connector for Monday.com.
package monday

import (
	"context"
	"fmt"

	"github.com/bakerweb/wt/internal/connector"
)

// Client is a placeholder for the Monday.com connector.
type Client struct{}

func New() *Client                                                   { return &Client{} }
func (c *Client) Name() string                                      { return "monday" }
func (c *Client) GetTicket(ctx context.Context, key string) (*connector.Ticket, error) {
	return nil, fmt.Errorf("monday.com connector is not yet implemented")
}
func (c *Client) ListAssigned(ctx context.Context) ([]connector.Ticket, error) {
	return nil, fmt.Errorf("monday.com connector is not yet implemented")
}
func (c *Client) TransitionTicket(ctx context.Context, key, status string) error {
	return fmt.Errorf("monday.com connector is not yet implemented")
}
func (c *Client) Validate(ctx context.Context) error {
	return fmt.Errorf("monday.com connector is not yet implemented")
}
