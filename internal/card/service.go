package card

import (
	"context"
	"github.com/go-funcards/card-service/proto/v1"
	"github.com/go-funcards/slice"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ v1.CardServer = (*cardService)(nil)

type cardService struct {
	v1.UnimplementedCardServer
	storage Storage
	log     *zap.Logger
}

func NewCardService(storage Storage, logger *zap.Logger) *cardService {
	return &cardService{
		storage: storage,
		log:     logger,
	}
}

func (s *cardService) CreateCard(ctx context.Context, in *v1.CreateCardRequest) (*emptypb.Empty, error) {
	if err := s.storage.Save(ctx, CreateCard(in)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *cardService) UpdateCard(ctx context.Context, in *v1.UpdateCardRequest) (*emptypb.Empty, error) {
	if err := s.storage.Save(ctx, UpdateCard(in)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *cardService) UpdateManyCards(ctx context.Context, in *v1.UpdateManyCardsRequest) (*emptypb.Empty, error) {
	if err := s.storage.SaveMany(ctx, slice.Map(in.GetCards(), func(item *v1.UpdateCardRequest) Card {
		return UpdateCard(item)
	})); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *cardService) DeleteCard(ctx context.Context, in *v1.DeleteCardRequest) (*emptypb.Empty, error) {
	if err := s.storage.Delete(ctx, in.GetCardId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *cardService) GetCards(ctx context.Context, in *v1.CardsRequest) (*v1.CardsResponse, error) {
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
			return item.toResponse()
		}),
		Total: total,
	}, nil
}
