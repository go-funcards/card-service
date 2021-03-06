package card

import "context"

type Storage interface {
	Save(ctx context.Context, model Card) error
	SaveMany(ctx context.Context, models []Card) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, filter Filter, index uint64, size uint32) ([]Card, error)
	Count(ctx context.Context, filter Filter) (uint64, error)
}
