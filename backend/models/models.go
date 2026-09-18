package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type PollOption struct {
	ID   string `bson:"id" json:"id"`
	Text string `bson:"text" json:"text"`
}

type Poll struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CreatorID         primitive.ObjectID `bson:"creatorId" json:"creatorId"`
	Question          string             `bson:"question" json:"question"`
	Options           []PollOption       `bson:"options" json:"options"`
	Status            string             `bson:"status" json:"status"`
	AllowMultipleVotes bool             `bson:"allowMultipleVotes" json:"allowMultipleVotes"`
	ExpiresAt         *time.Time         `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	PublicID          string             `bson:"publicId" json:"publicId"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Vote struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	PollID          primitive.ObjectID `bson:"pollId" json:"pollId"`
	OptionID        string             `bson:"optionId" json:"optionId"`
	VoterID         *primitive.ObjectID `bson:"voterId,omitempty" json:"voterId,omitempty"`
	VoterIdentifier string             `bson:"voterIdentifier" json:"voterIdentifier"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
}

type ResultItem struct {
	OptionID    string  `json:"optionId"`
	OptionText  string  `json:"optionText"`
	Votes       int64   `json:"votes"`
	Percentage  float64 `json:"percentage"`
	TotalVotes  int64   `json:"totalVotes,omitempty"`
}

type PollResult struct {
	PollID     string       `json:"pollId"`
	TotalVotes int64        `json:"totalVotes"`
	Results    []ResultItem `json:"results"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
