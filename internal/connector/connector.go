package connector

import "context"

// Ticket represents a task/issue from an external system.
type Ticket struct {
	Key         string
	Summary     string
	Description string
	Status      string
	Assignee    string
	URL         string
	Labels      []string
}

// Connector defines the interface that all task management integrations must implement.
type Connector interface {
	// Name returns the connector's identifier (e.g. "jira", "monday", "clickup").
	Name() string

	// GetTicket fetches a single ticket by its key/ID.
	GetTicket(ctx context.Context, key string) (*Ticket, error)

	// ListAssigned fetches tickets assigned to the current user.
	ListAssigned(ctx context.Context) ([]Ticket, error)

	// TransitionTicket moves a ticket to a new status.
	TransitionTicket(ctx context.Context, key, status string) error

	// Validate checks that the connector is properly configured.
	Validate(ctx context.Context) error
}

// Registry holds all registered connectors.
type Registry struct {
	connectors map[string]Connector
}

// NewRegistry creates a new connector registry.
func NewRegistry() *Registry {
	return &Registry{
		connectors: make(map[string]Connector),
	}
}

// Register adds a connector to the registry.
func (r *Registry) Register(c Connector) {
	r.connectors[c.Name()] = c
}

// Get retrieves a connector by name.
func (r *Registry) Get(name string) (Connector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

// List returns the names of all registered connectors.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.connectors))
	for name := range r.connectors {
		names = append(names, name)
	}
	return names
}
