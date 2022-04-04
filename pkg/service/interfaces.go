package service

import (
	"context"
	"time"

	"github.com/underpin-korea/protocol/livekit"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// encapsulates CRUD operations for room settings
//counterfeiter:generate . ObjectStore
type ObjectStore interface {
	ServiceStore
	EgressStore

	// enable locking on a specific room to prevent race
	// returns a (lock uuid, error)
	LockRoom(ctx context.Context, name livekit.RoomName, duration time.Duration) (string, error)
	UnlockRoom(ctx context.Context, name livekit.RoomName, uid string) error

	StoreRoom(ctx context.Context, room *livekit.Room) error
	DeleteRoom(ctx context.Context, name livekit.RoomName) error

	StoreParticipant(ctx context.Context, roomName livekit.RoomName, participant *livekit.ParticipantInfo) error
	DeleteParticipant(ctx context.Context, roomName livekit.RoomName, identity livekit.ParticipantIdentity) error
}

//counterfeiter:generate . ServiceStore
type ServiceStore interface {
	LoadRoom(ctx context.Context, name livekit.RoomName) (*livekit.Room, error)
	// ListRooms returns currently active rooms. if names is not nil, it'll filter and return
	// only rooms that match
	ListRooms(ctx context.Context, names []livekit.RoomName) ([]*livekit.Room, error)

	LoadParticipant(ctx context.Context, roomName livekit.RoomName, identity livekit.ParticipantIdentity) (*livekit.ParticipantInfo, error)
	ListParticipants(ctx context.Context, roomName livekit.RoomName) ([]*livekit.ParticipantInfo, error)
}

//counterfeiter:generate . EgressStore
type EgressStore interface {
	LoadRoom(ctx context.Context, name livekit.RoomName) (*livekit.Room, error)

	StoreEgress(ctx context.Context, info *livekit.EgressInfo) error
	LoadEgress(ctx context.Context, egressID string) (*livekit.EgressInfo, error)
	ListEgress(ctx context.Context, roomID livekit.RoomID) ([]*livekit.EgressInfo, error)
	UpdateEgress(ctx context.Context, info *livekit.EgressInfo) error
	DeleteEgress(ctx context.Context, info *livekit.EgressInfo) error
}

//counterfeiter:generate . RoomAllocator
type RoomAllocator interface {
	CreateRoom(ctx context.Context, req *livekit.CreateRoomRequest) (*livekit.Room, error)
}
