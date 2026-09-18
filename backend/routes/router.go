package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"livepoll/config"
	"livepoll/middleware"
	"livepoll/repositories"
	"livepoll/services"
	"livepoll/utils"
	"livepoll/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupRouter(cfg *config.Config, db *mongo.Database, redisClient *redis.Client, hub *websocket.Hub) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	userRepo := repositories.NewUserRepository(db)
	pollRepo := repositories.NewPollRepository(db)
	_ = userRepo.EnsureIndexes(context.Background())
	_ = pollRepo.EnsureIndexes(context.Background())

	userService := services.NewUserService(userRepo, cfg)
	pollService := services.NewPollService(pollRepo)

	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "ok", gin.H{"status": "ok", "database": "connected", "redis": "connected"})
	})

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/signup", func(c *gin.Context) {
				var payload struct {
					Name     string `json:"name"`
					Email    string `json:"email"`
					Password string `json:"password"`
				}
				if err := c.ShouldBindJSON(&payload); err != nil {
					utils.Error(c, http.StatusBadRequest, "Invalid signup payload", "invalid json")
					return
				}
				user, err := userService.Signup(c.Request.Context(), payload.Name, payload.Email, payload.Password)
				if err != nil {
					utils.Error(c, http.StatusBadRequest, "Signup failed", err.Error())
					return
				}
				utils.Success(c, http.StatusCreated, "User created successfully", gin.H{"user": gin.H{"id": user.ID.Hex(), "name": user.Name, "email": user.Email}})
			})

			auth.POST("/login", func(c *gin.Context) {
				var payload struct {
					Email    string `json:"email"`
					Password string `json:"password"`
				}
				if err := c.ShouldBindJSON(&payload); err != nil {
					utils.Error(c, http.StatusBadRequest, "Invalid login payload", "invalid json")
					return
				}
				user, token, err := userService.Login(c.Request.Context(), payload.Email, payload.Password, cfg)
				if err != nil {
					utils.Error(c, http.StatusUnauthorized, "Login failed", err.Error())
					return
				}
				utils.Success(c, http.StatusOK, "Login successful", gin.H{"token": token, "user": gin.H{"id": user.ID.Hex(), "name": user.Name, "email": user.Email}})
			})

			auth.GET("/me", middleware.AuthRequired(cfg), func(c *gin.Context) {
				userIDRaw, _ := c.Get("userID")
				userID, ok := userIDRaw.(string)
				if !ok {
					utils.Error(c, http.StatusUnauthorized, "User not found", "invalid user context")
					return
				}
				objID, err := primitive.ObjectIDFromHex(userID)
				if err != nil {
					utils.Error(c, http.StatusUnauthorized, "User not found", "invalid user id")
					return
				}
				user, err := userRepo.FindByID(c.Request.Context(), objID)
				if err != nil {
					utils.Error(c, http.StatusUnauthorized, "User not found", "user missing")
					return
				}
				utils.Success(c, http.StatusOK, "Authenticated user", gin.H{"id": user.ID.Hex(), "name": user.Name, "email": user.Email})
			})
		}

		polls := api.Group("/polls")
		{
			polls.GET("/public/:publicId", func(c *gin.Context) {
				publicID := c.Param("publicId")
				poll, err := pollRepo.GetPollByPublicID(c.Request.Context(), publicID)
				if err != nil {
					utils.Error(c, http.StatusNotFound, "Poll not found", "poll missing")
					return
				}
				utils.Success(c, http.StatusOK, "Poll retrieved", gin.H{"poll": pollService.BuildPublicPollResponse(poll)})
			})

			polls.POST("/public/:publicId/vote", func(c *gin.Context) {
				publicID := c.Param("publicId")
				poll, err := pollRepo.GetPollByPublicID(c.Request.Context(), publicID)
				if err != nil {
					utils.Error(c, http.StatusNotFound, "Poll not found", "poll missing")
					return
				}

				var payload struct {
					OptionID        string   `json:"optionId"`
					OptionIDs       []string `json:"optionIds"`
					VoterIdentifier string   `json:"voterIdentifier"`
				}
				if err := c.ShouldBindJSON(&payload); err != nil {
					utils.Error(c, http.StatusBadRequest, "Invalid vote payload", "invalid json")
					return
				}
				if payload.VoterIdentifier == "" {
					utils.Error(c, http.StatusBadRequest, "Vote failed", "voter identifier is required")
					return
				}

				selectedIDs := payload.OptionIDs
				if len(selectedIDs) == 0 && payload.OptionID != "" {
					selectedIDs = []string{payload.OptionID}
				}
				if len(selectedIDs) == 0 {
					utils.Error(c, http.StatusBadRequest, "Vote failed", "an option is required")
					return
				}

				if poll.AllowMultipleVotes {
					for _, optionID := range selectedIDs {
						if _, err := pollService.SaveVote(c.Request.Context(), poll, optionID, nil, payload.VoterIdentifier); err != nil {
							utils.Error(c, http.StatusUnprocessableEntity, "Vote failed", err.Error())
							return
						}
					}
				} else {
					if _, err := pollService.SaveVote(c.Request.Context(), poll, selectedIDs[0], nil, payload.VoterIdentifier); err != nil {
						utils.Error(c, http.StatusUnprocessableEntity, "Vote failed", err.Error())
						return
					}
				}

				updatedPoll, err := pollRepo.GetPollByID(c.Request.Context(), poll.ID)
				if err != nil {
					updatedPoll = poll
				}
				utils.Success(c, http.StatusOK, "Vote recorded", gin.H{"poll": pollService.BuildPublicPollResponse(updatedPoll)})
			})
		}

		polls.Use(middleware.AuthRequired(cfg))
		{
			polls.POST("", func(c *gin.Context) {
				var payload struct {
					Question          string    `json:"question"`
					Options           []string  `json:"options"`
					AllowMultipleVotes bool      `json:"allowMultipleVotes"`
					ExpiresAt         *time.Time `json:"expiresAt"`
				}
				if err := c.ShouldBindJSON(&payload); err != nil {
					utils.Error(c, http.StatusBadRequest, "Invalid poll payload", "invalid json")
					return
				}
				userIDRaw, _ := c.Get("userID")
				userIDStr, ok := userIDRaw.(string)
				if !ok {
					utils.Error(c, http.StatusUnauthorized, "User not found", "invalid user context")
					return
				}
				userID, err := primitive.ObjectIDFromHex(userIDStr)
				if err != nil {
					utils.Error(c, http.StatusUnauthorized, "User not found", "invalid user id")
					return
				}
				poll, err := pollService.CreatePoll(c.Request.Context(), userID, payload.Question, payload.Options, payload.AllowMultipleVotes, payload.ExpiresAt)
				if err != nil {
					utils.Error(c, http.StatusUnprocessableEntity, "Poll creation failed", err.Error())
					return
				}
				utils.Success(c, http.StatusCreated, "Poll created successfully", gin.H{"poll": poll})
			})

			polls.GET("", func(c *gin.Context) {
				userIDRaw, _ := c.Get("userID")
				userIDStr, ok := userIDRaw.(string)
				if !ok {
					utils.Error(c, http.StatusUnauthorized, "User not found", "invalid user context")
					return
				}
				userID, err := primitive.ObjectIDFromHex(userIDStr)
				if err != nil {
					utils.Error(c, http.StatusUnauthorized, "User not found", "invalid user id")
					return
				}
				pollsList, err := pollRepo.ListByCreator(c.Request.Context(), userID)
				if err != nil {
					utils.Error(c, http.StatusInternalServerError, "Unable to list polls", err.Error())
					return
				}
				utils.Success(c, http.StatusOK, "Polls retrieved", gin.H{"polls": pollsList})
			})

			polls.GET(":id", func(c *gin.Context) {
				id := c.Param("id")
				objID, err := primitive.ObjectIDFromHex(id)
				if err != nil {
					utils.Error(c, http.StatusBadRequest, "Invalid poll id", "invalid mongo id")
					return
				}
				poll, err := pollRepo.GetPollByID(c.Request.Context(), objID)
				if err != nil {
					utils.Error(c, http.StatusNotFound, "Poll not found", "poll missing")
					return
				}
				utils.Success(c, http.StatusOK, "Poll retrieved", gin.H{"poll": poll})
			})
		}
	}

	r.GET("/ws/polls/:pollId", func(c *gin.Context) {
		pollID := c.Param("pollId")
		websocket.HandleWebSocket(c, hub, pollID)
	})

	return r
}

func StartRedisSubscriber(ctx context.Context, client *redis.Client, hub *websocket.Hub) error {
	if client == nil || hub == nil {
		return fmt.Errorf("redis client and hub are required")
	}
	go func() {
		pubsub := client.Subscribe(ctx, "poll:*:results")
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			hub.Broadcast(msg.Channel, msg.Payload)
		}
	}()
	return nil
}
