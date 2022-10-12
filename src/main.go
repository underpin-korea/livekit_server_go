package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/livekit/protocol/auth"
	livekit "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"

	//webHook
	"github.com/livekit/protocol/webhook"
)

var (
	host       = "http://localhost:7880"
	apiKey     = "devkey"
	apiSecret  = "secret"
	roomClient = lksdk.NewRoomServiceClient(host, apiKey, apiSecret)
)

func getJoinToken(apiKey, apiSecret, room, identity string) (string, error) {
	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     room,
	}
	at.AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(time.Hour)

	return at.ToJWT()
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/api/room", func(c *gin.Context) {

		roomList, _ := roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
		c.JSON(http.StatusOK, gin.H{
			"message": roomList,
		})
	})

	r.DELETE("/api/room", func(c *gin.Context) {

		room, _ := roomClient.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
			Room: "test4",
		})
		c.JSON(http.StatusOK, gin.H{
			"message": room,
		})
	})

	r.POST("/api/room", func(c *gin.Context) {
		room, _ := roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{

			Name: "test3",
		})

		at := auth.NewAccessToken(apiKey, apiSecret)
		grant := &auth.VideoGrant{
			RoomJoin: true,
			Room:     room.Name,
		}

		at.AddGrant(grant).
			SetIdentity("test1").
			SetValidFor(time.Hour)

		result, _ := at.ToJWT()

		c.JSON(http.StatusOK, gin.H{
			"message": result,
		})

	})

	r.POST("/api/webhook/room/create", func(c *gin.Context) {
		authProvider := auth.NewSimpleKeyProvider(
			apiKey, apiSecret,
			// event is a livekit.WebhookEvent{} object
		)
		event, _ := webhook.ReceiveWebhookEvent(c.Request, authProvider)

		// eventId := event.Id
		// eventCreatedAt := event.CreatedAt
		// eventRoom := event.Room
		// print("eventId?==> :{}", eventId)
		// print("eventCreatedAt?==> :{}", eventCreatedAt)
		// print("eventRoom?==> :{}", eventRoom)

		c.JSON(http.StatusOK, gin.H{
			"message": event,
		})

	})

	//////////////////////////////////////////////////////
	//목록
	r.GET("/api/room/participant", func(c *gin.Context) {

		participantList, _ := roomClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
			Room: "test2",
			//token인가
		})
		c.JSON(http.StatusOK, gin.H{
			"message": participantList,
		})
	})
	// 정보
	r.GET("/api/room/participant/{identity}", func(c *gin.Context) {
		participant, _ := roomClient.GetParticipant(context.Background(), &livekit.RoomParticipantIdentity{

			Room:     "test1",
			Identity: "identity",
		})
		c.JSON(http.StatusOK, gin.H{
			"message": participant,
		})

	})
	//remove
	//--------
	// r.DELETE("/api/room/participant/{identity}", func(c *gin.Context) {

	// 	participant, _ := roomClient.RemoveParticipant(context.Background(), &livekit.RemoveParticipant{
	// 		ParticipantId: "test1",

	// 	})

	// 	c.JSON(http.StatusOK, gin.H{
	// 		"message": participant,
	// 	})

	// })
	//--------
	// //mute
	// r.PATCH("/api/room/participant/{identity}", func(c *gin.Context) {
	// 	MutePublishedTrack, _ := roomClient.RemoveParticipant(context.Background(), &livekit.MuteRoomTrackRequest{

	// 		Room:     "test1",
	// 		Identity: "idnetity",
	// 		TrackSid: "dddddddds",
	// 		Muted1:   false,
	// 	})

	// 	c.JSON(http.StatusOK, gin.H{
	// 		"message": MutePublishedTrack,
	// 	})

	// })
	//update
	// r.PUT("/api/room/participant/{identity}", func(c *gin.Context) {
	// 	MutePublishedTrack, _ := roomClient.RemoveParticipant(context.Background(), &livekit.ParticipantUpdate{

	// 		[]*ParticipantInfo:=,

	// 	})

	// 	c.JSON(http.StatusOK, gin.H{
	// 		"message": MutePublishedTrack,
	// 	})

	// })

	//////////////////////////////////////////////////////

	// func ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// 	authProvider := auth.NewSimpleKeyProvider(
	// 		apiKey, apiSecret,
	// 	)
	// 	// event is a livekit.WebhookEvent{} object
	// 	event, err := webhook.ReceiveWebhookEvent(r, authProvider)
	// 	if err != nil {
	// 		// could not validate, handle error
	// 		return
	// 	}

	// consume WebhookEvent
	// }

	return r

}

func main() {
	r := setupRouter()
	print("hello World")
	r.Run(":8081")
}
