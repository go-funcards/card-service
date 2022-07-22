package db

import (
	"context"
	"fmt"
	"github.com/go-funcards/card-service/internal/card"
	"github.com/go-funcards/mongodb"
	"github.com/go-funcards/slice"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var _ card.Storage = (*storage)(nil)

const (
	timeout    = 5 * time.Second
	collection = "cards"
)

type storage struct {
	c   *mongo.Collection
	log logrus.FieldLogger
}

func NewStorage(ctx context.Context, db *mongo.Database, log logrus.FieldLogger) *storage {
	s := &storage{
		c:   db.Collection(collection),
		log: log,
	}
	s.indexes(ctx)
	return s
}

func (s *storage) indexes(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, err := s.c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{"owner_id", 1},
			{"board_id", 1},
			{"category_id", 1},
			{"type", 1},
			{"position", 1},
			{"created_at", 1},
			{"tags", 1},
		},
		Options: options.Index().SetName("cards_index_01"),
	})
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"collection": collection,
			"error":      err,
		}).Fatal("index not created")
	}

	s.log.WithFields(logrus.Fields{
		"collection": collection,
		"name":       name,
	}).Info("index created")
}

func (s *storage) Save(ctx context.Context, model card.Card) error {
	return s.SaveMany(ctx, []card.Card{model})
}

func (s *storage) SaveMany(ctx context.Context, models []card.Card) error {
	var write []mongo.WriteModel
	for _, model := range models {
		data, err := mongodb.ToBson(model)
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
			s.log.WithFields(logrus.Fields{
				"card_id":     model.CardID,
				"attachments": deleteAtts,
			}).Info("delete attachments")

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

	s.log.Info("cards save")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := s.c.BulkWrite(ctx, write)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("cards save: %s", mongodb.ErrMsgQuery), err)
	}

	s.log.WithFields(logrus.Fields{"result": result}).Info("cards saved")

	return nil
}

func (s *storage) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	s.log.WithField("card_id", id).Debug("card delete")
	result, err := s.c.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf(mongodb.ErrMsgQuery, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf(mongodb.ErrMsgQuery, mongo.ErrNoDocuments)
	}
	s.log.WithField("card_id", id).Debug("card deleted")

	return nil
}

func (s *storage) Find(ctx context.Context, filter card.Filter, index uint64, size uint32) ([]card.Card, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	opts := mongodb.FindOptions(index, size).SetSort(bson.D{{"position", 1}, {"created_at", 1}})
	cur, err := s.c.Find(ctx, s.build(filter), opts)
	if err != nil {
		return nil, fmt.Errorf(mongodb.ErrMsgQuery, err)
	}
	return mongodb.DecodeAll[card.Card](ctx, cur)
}

func (s *storage) Count(ctx context.Context, filter card.Filter) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	total, err := s.c.CountDocuments(ctx, s.build(filter))
	if err != nil {
		return 0, fmt.Errorf(mongodb.ErrMsgQuery, err)
	}
	return uint64(total), nil
}

func (s *storage) build(filter card.Filter) any {
	f := make(mongodb.Filter, 0)
	if len(filter.CardIDs) > 0 {
		f = append(f, mongodb.In("_id", filter.CategoryIDs))
	}
	if len(filter.Types) > 0 {
		f = append(f, mongodb.In("type", filter.Types))
	}
	if len(filter.Tags) > 0 {
		f = append(f, mongodb.In("tags", filter.Tags))
	}
	if len(filter.OwnerIDs) > 0 {
		f = append(f, mongodb.In("owner_id", filter.OwnerIDs))
	}
	if len(filter.BoardIDs) > 0 {
		f = append(f, mongodb.In("board_id", filter.BoardIDs))
	}
	if len(filter.CategoryIDs) > 0 {
		f = append(f, mongodb.In("category_id", filter.CategoryIDs))
	}
	return f.Build()
}
