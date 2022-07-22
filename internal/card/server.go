package card

import (
	"context"
	"github.com/go-funcards/card-service/proto/v1"
	"github.com/go-funcards/slice"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ v1.CardServer = (*server)(nil)

type server struct {
	v1.UnimplementedCardServer
	storage Storage
}

func NewCardServer(storage Storage) *server {
	return &server{storage: storage}
}

func (s *server) CreateCard(ctx context.Context, in *v1.CreateCardRequest) (*emptypb.Empty, error) {
	err := s.storage.Save(ctx, CreateCard(in))

	return s.empty(err)
}

func (s *server) UpdateCard(ctx context.Context, in *v1.UpdateCardRequest) (*emptypb.Empty, error) {
	err := s.storage.Save(ctx, UpdateCard(in))

	return s.empty(err)
}

func (s *server) UpdateManyCards(ctx context.Context, in *v1.UpdateManyCardsRequest) (*emptypb.Empty, error) {
	err := s.storage.SaveMany(ctx, slice.Map(in.GetCards(), func(item *v1.UpdateCardRequest) Card {
		return UpdateCard(item)
	}))

	return s.empty(err)
}

func (s *server) DeleteCard(ctx context.Context, in *v1.DeleteCardRequest) (*emptypb.Empty, error) {
	err := s.storage.Delete(ctx, in.GetCardId())

	return s.empty(err)
}

func (s *server) GetCards(ctx context.Context, in *v1.CardsRequest) (*v1.CardsResponse, error) {
	filter := CreateFilter(in)

	data, err := s.storage.Find(ctx, filter, in.GetPageIndex(), in.GetPageSize())
	if err != nil {
		return nil, err
	}

	total := uint64(len(data))
	if len(in.GetCardIds()) == 0 && uint64(in.GetPageSize()) == total {
		if total, err = s.storage.Count(ctx, filter); err != nil {
			return nil, err
		}
	}

	return &v1.CardsResponse{
		Cards: slice.Map(data, func(item Card) *v1.CardsResponse_Card {
			return item.toProto()
		}),
		Total: total,
	}, nil
}

func (s *server) empty(err error) (*emptypb.Empty, error) {
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
