package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"livepoll/models"
	"livepoll/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PollService struct {
	polls   *repositories.PollRepository
	results string
}

func NewPollService(polls *repositories.PollRepository) *PollService {
	return &PollService{polls: polls}
}

func (s *PollService) CreatePoll(ctx context.Context, creatorID primitive.ObjectID, question string, options []string, allowMultiple bool, expiresAt *time.Time) (*models.Poll, error) {
	question = strings.TrimSpace(question)
	if question == "" || len(question) > 200 {
		return nil, errors.New("question is required and must be under 200 characters")
	}
	if len(options) < 2 || len(options) > 10 {
		return nil, errors.New("poll must have between 2 and 10 options")
	}

	seen := map[string]bool{}
	clean := make([]models.PollOption, 0, len(options))
	for _, opt := range options {
		text := strings.TrimSpace(opt)
		if text == "" {
			return nil, errors.New("all options must be filled")
		}
		if seen[strings.ToLower(text)] {
			return nil, errors.New("duplicate options are not allowed")
		}
		seen[strings.ToLower(text)] = true
		clean = append(clean, models.PollOption{ID: primitive.NewObjectID().Hex(), Text: text})
	}

	if expiresAt != nil && expiresAt.Before(time.Now().Add(-time.Minute)) {
		return nil, errors.New("expiration date must be in the future")
	}

	poll := &models.Poll{
		CreatorID:         creatorID,
		Question:          question,
		Options:           clean,
		Status:            "active",
		AllowMultipleVotes: allowMultiple,
		ExpiresAt:         expiresAt,
		PublicID:          primitive.NewObjectID().Hex(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := s.polls.CreatePoll(ctx, poll); err != nil {
		return nil, err
	}
	return poll, nil
}

func (s *PollService) ValidateVote(ctx context.Context, poll *models.Poll, optionID string, voterIdentifier string) error {
	if poll == nil {
		return errors.New("poll not found")
	}
	if poll.Status != "active" {
		return errors.New("poll is not active")
	}
	if poll.ExpiresAt != nil && time.Now().After(*poll.ExpiresAt) {
		return errors.New("poll has expired")
	}
	if voterIdentifier == "" {
		return errors.New("voter identifier is required")
	}
	if !poll.AllowMultipleVotes {
		_, err := s.polls.FindVoteByPollAndVoter(ctx, poll.ID, voterIdentifier)
		if err == nil {
			return errors.New("duplicate vote not allowed")
		}
	}
	found := false
	for _, option := range poll.Options {
		if option.ID == optionID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("option does not belong to the poll")
	}
	return nil
}

func (s *PollService) SaveVote(ctx context.Context, poll *models.Poll, optionID string, voterID *primitive.ObjectID, voterIdentifier string) (*models.Vote, error) {
	if err := s.ValidateVote(ctx, poll, optionID, voterIdentifier); err != nil {
		return nil, err
	}
	vote := &models.Vote{
		PollID:          poll.ID,
		OptionID:        optionID,
		VoterID:         voterID,
		VoterIdentifier: voterIdentifier,
		CreatedAt:       time.Now(),
	}
	if err := s.polls.SaveVote(ctx, vote); err != nil {
		return nil, err
	}
	return vote, nil
}

func (s *PollService) CalculateResults(ctx context.Context, poll *models.Poll) (*models.PollResult, error) {
	if poll == nil {
		return nil, errors.New("poll not found")
	}
	votes, err := s.polls.GetVotesForPoll(ctx, poll.ID)
	if err != nil {
		return nil, err
	}
	counts := map[string]int64{}
	for _, vote := range votes {
		counts[vote.OptionID]++
	}

	results := make([]models.ResultItem, 0, len(poll.Options))
	totalVotes := int64(0)
	for _, option := range poll.Options {
		count := counts[option.ID]
		totalVotes += count
	}

	for _, option := range poll.Options {
		count := counts[option.ID]
		percent := 0.0
		if totalVotes > 0 {
			percent = float64(count) / float64(totalVotes) * 100
		}
		results = append(results, models.ResultItem{
			OptionID:   option.ID,
			OptionText: option.Text,
			Votes:      count,
			Percentage: math.Round(percent*100) / 100,
			TotalVotes: totalVotes,
		})
	}

	return &models.PollResult{
		PollID:     poll.PublicID,
		TotalVotes: totalVotes,
		Results:    results,
	}, nil
}

func (s *PollService) PublishPollResults(ctx context.Context, redisClient any, pollID string, result *models.PollResult) error {
	payload, err := json.Marshal(map[string]any{
		"type":     "poll_results_updated",
		"pollId":   pollID,
		"results":  result.Results,
		"totalVotes": result.TotalVotes,
	})
	if err != nil {
		return err
	}
	_ = redisClient
	_ = ctx
	_ = payload
	return nil
}

func (s *PollService) BuildPublicPollResponse(poll *models.Poll) map[string]any {
	result, err := s.CalculateResults(context.Background(), poll)
	if err != nil {
		result = &models.PollResult{PollID: poll.PublicID, TotalVotes: 0, Results: []models.ResultItem{}}
	}
	return map[string]any{
		"id":                poll.ID.Hex(),
		"publicId":          poll.PublicID,
		"question":          poll.Question,
		"options":           poll.Options,
		"status":            poll.Status,
		"allowMultipleVotes": poll.AllowMultipleVotes,
		"expiresAt":         poll.ExpiresAt,
		"createdAt":         poll.CreatedAt,
		"updatedAt":         poll.UpdatedAt,
		"results":           result.Results,
		"totalVotes":        result.TotalVotes,
	}
}

func (s *PollService) MakeShareURL(baseURL, publicID string) string {
	return fmt.Sprintf("%s/poll/%s", strings.TrimRight(baseURL, "/"), publicID)
}
