package db

import (
	"context"
	"fmt"
	"github.com/go-funcards/card-service/internal/card"
	"github.com/go-funcards/mongodb"
	"github.com/go-funcards/slice"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"time"
)

var _ card.Storage = (*storage)(nil)

const (
	timeout    = 5 * time.Second
	collection = "cards"
)

type storage struct {
	c mongodb.Collection[card.Card]
}

func NewStorage(ctx context.Context, db *mongo.Database, logger *zap.Logger) (*storage, error) {
	s := &storage{c: mongodb.Collection[card.Card]{
		Inner: db.Collection(collection),
		Log:   logger,
	}}

	if err := s.indexes(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *storage) indexes(ctx context.Context) error {
	name, err := s.c.Inner.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{"owner_id", 1},
			{"board_id", 1},
			{"category_id", 1},
			{"type", 1},
			{"position", 1},
			{"created_at", 1},
			{"tags", 1},
		},
	})
	if err == nil {
		s.c.Log.Info("index created", zap.String("collection", collection), zap.String("name", name))
	}

	return err
}

func (s *storage) Save(ctx context.Context, model card.Card) error {
	return s.SaveMany(ctx, []card.Card{model})
}

func (s *storage) SaveMany(ctx context.Context, models []card.Card) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var write []mongo.WriteModel

	for _, model := range models {
		data, err := s.c.ToM(model)
		if err != nil {
			return err
		}

		delete(data, "_id")
		delete(data, "owner_id")
		delete(data, "type")
		delete(data, "created_at")
		delete(data, "attachments")

		if deleteAtts := slice.Map(model.Attachments, func(item card.Attachment) string {
			return item.AttachmentID
		}); len(deleteAtts) > 0 {
			s.c.Log.Info("delete attachments", zap.String("card_id", model.CardID), zap.Strings("attachments", deleteAtts))

			write = append(write, mongo.
				NewUpdateOneModel().
				SetFilter(bson.M{"_id": model.CardID}).
				SetUpdate(bson.M{
					"$pull": bson.M{
						"attachments": bson.M{
							"attachment_id": bson.M{
								"$in": deleteAtts,
							},
						},
					},
				}),
			)
		}

		addAtts := slice.Filter(model.Attachments, func(item card.Attachment) bool {
			return !item.Delete
		})

		write = append(write, mongo.
			NewUpdateOneModel().
			SetUpsert(true).
			SetFilter(bson.M{"_id": model.CardID}).
			SetUpdate(bson.M{
				"$set": data,
				"$setOnInsert": bson.M{
					"owner_id":   model.OwnerID,
					"type":       model.Type,
					"created_at": model.CreatedAt,
				},
				"$addToSet": bson.M{
					"attachments": bson.M{"$each": addAtts},
				},
			}),
		)
	}

	s.c.Log.Debug("bulk update")

	result, err := s.c.Inner.BulkWrite(ctx, write, options.BulkWrite())
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("bulk update: %s", mongodb.ErrMsgQuery), err)
	}

	s.c.Log.Info("document updated", zap.Any("result", result))

	return nil
}

func (s *storage) Delete(ctx context.Context, id string) error {
	return s.c.DeleteOne(ctx, bson.M{"_id": id})
}

func (s *storage) Find(ctx context.Context, filter card.Filter, index uint64, size uint32) ([]card.Card, error) {
	return s.c.Find(ctx, s.filter(filter), s.c.FindOptions(index, size).
		SetSort(bson.D{{"position", 1}, {"created_at", 1}}))
}

func (s *storage) Count(ctx context.Context, filter card.Filter) (uint64, error) {
	return s.c.CountDocuments(ctx, s.filter(filter))
}

func (s *storage) filter(filter card.Filter) bson.M {
	f := make(bson.M)
	if len(filter.CardIDs) > 0 {
		f["_id"] = bson.M{"$in": filter.CardIDs}
	}
	if len(filter.Types) > 0 {
		f["type"] = bson.M{"$in": filter.Types}
	}
	if len(filter.Tags) > 0 {
		f["tags"] = bson.M{"$in": filter.Tags}
	}
	if len(filter.OwnerIDs) > 0 {
		f["owner_id"] = bson.M{"$in": filter.OwnerIDs}
	}
	if len(filter.BoardIDs) > 0 {
		f["board_id"] = bson.M{"$in": filter.BoardIDs}
	}
	if len(filter.CategoryIDs) > 0 {
		f["category_id"] = bson.M{"$in": filter.CategoryIDs}
	}
	return f
}
