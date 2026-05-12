package notify

import "context"

type Medium string

const (
	MediumWhatsApp Medium = "whatsapp"
)

type Sender interface {
	Send(ctx context.Context, to string, text string) error
}
