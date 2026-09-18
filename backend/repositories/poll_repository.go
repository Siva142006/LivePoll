package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"livepoll/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PollRepository struct {
	collection *mongo.Collection
	votes      *mongo.Collection
}

func NewPollRepository(db *mongo.Database) *PollRepository {
	return &PollRepository{
		collection: db.Collection("polls"),
		votes:      db.Collection("votes"),
	}
}

func (r *PollRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "creatorId", Value: 1}}},
		{Keys: bson.D{{Key: "publicId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})
	if err != nil {
		return err
	}
	_, err = r.votes.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "pollId", Value: 1}}},
		{Keys: bson.D{{Key: "pollId", Value: 1}, {Key: "voterIdentifier", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "pollId", Value: 1}, {Key: "optionId", Value: 1}}},
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *PollRepository) CreatePoll(ctx context.Context, poll *models.Poll) error {
	if poll.ID.IsZero() {
		poll.ID = primitive.NewObjectID()
	}
	if poll.PublicID == "" {
		poll.PublicID = primitive.NewObjectID().Hex()
	}
	poll.CreatedAt = time.Now()
	poll.UpdatedAt = poll.CreatedAt
	if poll.Status == "" {
		poll.Status = "active"
	}
	_, err := r.collection.InsertOne(ctx, poll)
	return err
}

func (r *PollRepository) GetPollByID(ctx context.Context, id primitive.ObjectID) (*models.Poll, error) {
	var poll models.Poll
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&poll)
	if err != nil {
		return nil, err
	}
	return &poll, nil
}

func (r *PollRepository) GetPollByPublicID(ctx context.Context, publicID string) (*models.Poll, error) {
	var poll models.Poll
	err := r.collection.FindOne(ctx, bson.M{"publicId": publicID}).Decode(&poll)
	if err != nil {
		return nil, err
	}
	return &poll, nil
}

func (r *PollRepository) ListByCreator(ctx context.Context, creatorID primitive.ObjectID) ([]models.Poll, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"creatorId": creatorID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err = cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

func (r *PollRepository) ListAll(ctx context.Context) ([]models.Poll, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err = cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

func (r *PollRepository) DeletePoll(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *PollRepository) ClosePoll(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": "closed", "updatedAt": time.Now()}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("poll not found")
	}
	return nil
}

func (r *PollRepository) SaveVote(ctx context.Context, vote *models.Vote) error {
	vote.CreatedAt = time.Now()
	_, err := r.votes.InsertOne(ctx, vote)
	return err
}

func (r *PollRepository) CountVotesByPoll(ctx context.Context, pollID primitive.ObjectID) (int64, error) {
	return r.votes.CountDocuments(ctx, bson.M{"pollId": pollID})
}

func (r *PollRepository) FindVoteByPollAndVoter(ctx context.Context, pollID primitive.ObjectID, voterIdentifier string) (*models.Vote, error) {
	var vote models.Vote
	err := r.votes.FindOne(ctx, bson.M{"pollId": pollID, "voterIdentifier": voterIdentifier}).Decode(&vote)
	if err != nil {
		return nil, err
	}
	return &vote, nil
}

func (r *PollRepository) CountVotesByOption(ctx context.Context, pollID primitive.ObjectID, optionID string) (int64, error) {
	return r.votes.CountDocuments(ctx, bson.M{"pollId": pollID, "optionId": optionID})
}

func (r *PollRepository) GetVotesForPoll(ctx context.Context, pollID primitive.ObjectID) ([]models.Vote, error) {
	cursor, err := r.votes.Find(ctx, bson.M{"pollId": pollID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var votes []models.Vote
	if err = cursor.All(ctx, &votes); err != nil {
		return nil, err
	}
	return votes, nil
}

func (r *PollRepository) ValidatePollOwnership(ctx context.Context, pollID primitive.ObjectID, userID primitive.ObjectID) error {
	poll, err := r.GetPollByID(ctx, pollID)
	if err != nil {
		return err
	}
	if poll.CreatorID != userID {
		return fmt.Errorf("forbidden")
	}
	return nil
}
