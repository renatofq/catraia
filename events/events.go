package events

const (
	ContainerCreated = "CREATED"
)

type ContainerEvent struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
}
