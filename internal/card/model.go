package card

import (
	"github.com/go-funcards/card-service/proto/v1"
	"github.com/go-funcards/slice"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Attachment struct {
	AttachmentID string `json:"attachment_id" bson:"attachment_id,omitempty"`
	Metadata     string `json:"metadata" bson:"metadata,omitempty"`
	Delete       bool   `json:"-" bson:"-"`
}

type Card struct {
	CardID      string       `json:"card_id" bson:"_id,omitempty"`
	OwnerID     string       `json:"owner_id" bson:"owner_id,omitempty"`
	BoardID     string       `json:"board_id" bson:"board_id,omitempty"`
	CategoryID  string       `json:"category_id" bson:"category_id,omitempty"`
	Name        string       `json:"name" bson:"name,omitempty"`
	Type        string       `json:"type" bson:"type,omitempty"`
	Content     string       `json:"content" bson:"content,omitempty"`
	Position    int32        `json:"position" bson:"position,omitempty"`
	CreatedAt   time.Time    `json:"created_at" bson:"created_at,omitempty"`
	Tags        []string     `json:"tags" bson:"tags,omitempty"`
	Attachments []Attachment `json:"attachments" bson:"attachments,omitempty"`
}

type Filter struct {
	Types       []string `json:"types,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CardIDs     []string `json:"card_ids,omitempty"`
	OwnerIDs    []string `json:"owner_ids,omitempty"`
	BoardIDs    []string `json:"board_ids,omitempty"`
	CategoryIDs []string `json:"category_ids,omitempty"`
}

func (a Attachment) toProto() *v1.CardsResponse_Card_Attachment {
	return &v1.CardsResponse_Card_Attachment{
		AttachmentId: a.AttachmentID,
		Metadata:     a.Metadata,
	}
}

func (c Card) toProto() *v1.CardsResponse_Card {
	return &v1.CardsResponse_Card{
		CardId:     c.CardID,
		OwnerId:    c.OwnerID,
		BoardId:    c.BoardID,
		CategoryId: c.CategoryID,
		Name:       c.Name,
		Type:       v1.CardType(v1.CardType_value[c.Type]),
		Content:    c.Content,
		Position:   c.Position,
		CreatedAt:  timestamppb.New(c.CreatedAt),
		Tags:       c.Tags,
		Attachments: slice.Map(c.Attachments, func(a Attachment) *v1.CardsResponse_Card_Attachment {
			return a.toProto()
		}),
	}
}

func CreateCard(in *v1.CreateCardRequest) Card {
	return Card{
		CardID:     in.GetCardId(),
		OwnerID:    in.GetOwnerId(),
		BoardID:    in.GetBoardId(),
		CategoryID: in.GetCategoryId(),
		Name:       in.GetName(),
		Type:       in.GetType().String(),
		Content:    in.GetContent(),
		Position:   in.GetPosition(),
		CreatedAt:  time.Now().UTC(),
		Tags:       in.GetTags(),
		Attachments: slice.Map(in.GetAttachments(), func(item *v1.CreateCardRequest_Att) Attachment {
			return Attachment{AttachmentID: item.GetAttachmentId(), Metadata: item.GetMetadata()}
		}),
	}
}

func UpdateCard(in *v1.UpdateCardRequest) Card {
	return Card{
		CardID:     in.GetCardId(),
		BoardID:    in.GetBoardId(),
		CategoryID: in.GetCategoryId(),
		Name:       in.GetName(),
		Content:    in.GetContent(),
		Position:   in.GetPosition(),
		Tags:       in.GetTags(),
		Attachments: slice.Map(in.GetAttachments(), func(item *v1.UpdateCardRequest_Att) Attachment {
			return Attachment{
				AttachmentID: item.GetAttachmentId(),
				Metadata:     item.GetMetadata(),
				Delete:       item.GetDelete(),
			}
		}),
	}
}

func CreateFilter(in *v1.CardsRequest) Filter {
	return Filter{
		Types: slice.Map(in.GetTypes(), func(item v1.CardType) string {
			return item.String()
		}),
		Tags:        in.GetTags(),
		CardIDs:     in.GetCardIds(),
		OwnerIDs:    in.GetOwnerIds(),
		BoardIDs:    in.GetBoardIds(),
		CategoryIDs: in.GetCategoryIds(),
	}
}
