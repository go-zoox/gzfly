package room

import (
	"github.com/go-zoox/gzfly/user"
)

// RoomsManager managers all rooms
type RoomsManager interface {
	// CreateRoom creates a room by admin
	CreateRoom(admin *user.User, id, secret string) error

	// DissolveRoom dissolves a room by admin
	DissolveRoom(admin *user.User, id string) error

	// SetupOwner setups the owner for specify room by admin
	SetupOwner(admin *user.User, id string, owner *user.User) error

	// Invite invites user to all rooms
	Invite(user *user.User) error
	// InviteTo invites user to specific rooms
	InviteTo(user *user.User, id ...string) error

	// Kickout kickouts user from all rooms
	Kickout(user *user.User) error
	// KickoutFrom kickouts user from specific rooms
	KickoutFrom(user *user.User, id ...string) error

	// User

	// Join is user join to specific room with secret
	Join(user *user.User, id, secret string) error
	// Leave is user levave specific room
	Leave(user *user.User, id string, code int, message string) error

	// LeaveByOfflineTimeout is user leave all rooms by timeout back online
	LeaveByBackOnlineTimeout(user *user.User) error
}

// Room is a room for users
type Room interface {
	// Room

	// Invite invites user to current room
	Invite(user *user.User) error
	// Kickout kickouts user from current room
	Kickout(user *user.User) error

	// User

	// Join is user join to current room with secret
	Join(user *user.User, secret string) error
	// Leave is user levave current room
	Leave(user *user.User, code int, message string) error

	// Network

	// Offline is user websocket connection offline by network poor
	Offline(user *user.User) error
	// Online is user websocket connection back online
	Online(user *user.User) error
}
