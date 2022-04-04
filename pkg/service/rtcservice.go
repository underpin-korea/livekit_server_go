package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/sebest/xff"

	"github.com/gorilla/websocket"
	"github.com/ua-parser/uap-go/uaparser"
	"github.com/underpin-korea/protocol/livekit"
	"github.com/underpin-korea/protocol/logger"

	"github.com/underpin-korea/livekit_server_go/pkg/config"
	"github.com/underpin-korea/livekit_server_go/pkg/routing"
	"github.com/underpin-korea/livekit_server_go/pkg/routing/selector"
	"github.com/underpin-korea/livekit_server_go/pkg/rtc"
	"github.com/underpin-korea/livekit_server_go/pkg/rtc/types"
	"github.com/underpin-korea/livekit_server_go/pkg/telemetry/prometheus"
)

type RTCService struct {
	router        routing.MessageRouter
	roomAllocator RoomAllocator
	store         ServiceStore
	upgrader      websocket.Upgrader
	currentNode   routing.LocalNode
	config        *config.Config
	isDev         bool
	limits        config.LimitConfig
	parser        *uaparser.Parser
}

func NewRTCService(
	conf *config.Config,
	ra RoomAllocator,
	store ServiceStore,
	router routing.MessageRouter,
	currentNode routing.LocalNode,
) *RTCService {
	s := &RTCService{
		router:        router,
		roomAllocator: ra,
		store:         store,
		upgrader:      websocket.Upgrader{},
		currentNode:   currentNode,
		config:        conf,
		isDev:         conf.Development,
		limits:        conf.Limit,
		parser:        uaparser.NewFromSaved(),
	}

	// allow connections from any origin, since script may be hosted anywhere
	// security is enforced by access tokens
	s.upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	return s
}

func (s *RTCService) Validate(w http.ResponseWriter, r *http.Request) {
	_, _, code, err := s.validate(r)
	if err != nil {
		handleError(w, code, err.Error())
		return
	}
	_, _ = w.Write([]byte("success"))
}

func (s *RTCService) validate(r *http.Request) (livekit.RoomName, routing.ParticipantInit, int, error) {
	claims := GetGrants(r.Context())
	// require a claim
	if claims == nil || claims.Video == nil {
		return "", routing.ParticipantInit{}, http.StatusUnauthorized, rtc.ErrPermissionDenied
	}

	onlyName, err := EnsureJoinPermission(r.Context())
	if err != nil {
		return "", routing.ParticipantInit{}, http.StatusUnauthorized, err
	}

	roomName := livekit.RoomName(r.FormValue("room"))
	reconnectParam := r.FormValue("reconnect")
	autoSubParam := r.FormValue("auto_subscribe")
	publishParam := r.FormValue("publish")

	if onlyName != "" {
		roomName = onlyName
	}

	// this is new connection for existing participant -  with publish only permissions
	if publishParam != "" {
		// Make sure grant has CanPublish set,
		if !claims.Video.GetCanPublish() {
			return "", routing.ParticipantInit{}, http.StatusUnauthorized, rtc.ErrPermissionDenied
		}
		// Make sure by default subscribe is off
		claims.Video.SetCanSubscribe(false)
		claims.Identity += "#" + publishParam
	}

	if router, ok := s.router.(routing.Router); ok {
		if foundNode, err := router.GetNodeForRoom(r.Context(), roomName); err == nil {
			if selector.LimitsReached(s.limits, foundNode.Stats) {
				return "", routing.ParticipantInit{}, http.StatusServiceUnavailable, rtc.ErrLimitExceeded
			}
		}
	}

	pi := routing.ParticipantInit{
		Reconnect:     boolValue(reconnectParam),
		Identity:      livekit.ParticipantIdentity(claims.Identity),
		Name:          livekit.ParticipantName(claims.Name),
		AutoSubscribe: true,
		Client:        s.ParseClientInfo(r),
		Grants:        claims,
	}

	if autoSubParam != "" {
		pi.AutoSubscribe = boolValue(autoSubParam)
	}

	return roomName, pi, http.StatusOK, nil
}

func (s *RTCService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// reject non websocket requests
	if !websocket.IsWebSocketUpgrade(r) {
		prometheus.ServiceOperationCounter.WithLabelValues("signal_ws", "error", "reject").Add(1)
		w.WriteHeader(404)
		return
	}

	roomName, pi, code, err := s.validate(r)
	if err != nil {
		handleError(w, code, err.Error())
		return
	}

	// when autocreate is disabled, we'll check to ensure it's already created
	if !s.config.Room.AutoCreate {
		_, err := s.store.LoadRoom(context.Background(), roomName)
		if err == ErrRoomNotFound {
			handleError(w, 404, err.Error())
			return
		} else if err != nil {
			handleError(w, 500, err.Error())
			return
		}
	}

	// create room if it doesn't exist, also assigns an RTC node for the room
	rm, err := s.roomAllocator.CreateRoom(r.Context(), &livekit.CreateRoomRequest{Name: string(roomName)})
	if err != nil {
		prometheus.ServiceOperationCounter.WithLabelValues("signal_ws", "error", "create_room").Add(1)
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// this needs to be started first *before* using router functions on this node
	connId, reqSink, resSource, err := s.router.StartParticipantSignal(r.Context(), roomName, pi)
	if err != nil {
		prometheus.ServiceOperationCounter.WithLabelValues("signal_ws", "error", "start_signal").Add(1)
		handleError(w, http.StatusInternalServerError, "could not start session: "+err.Error())
		return
	}

	pLogger := rtc.LoggerWithParticipant(
		rtc.LoggerWithRoom(logger.Logger(logger.GetLogger()), roomName, ""),
		pi.Identity, "",
	)
	done := make(chan struct{})
	// function exits when websocket terminates, it'll close the event reading off of response sink as well
	defer func() {
		pLogger.Infow("server closing WS connection", "connID", connId)
		reqSink.Close()
		close(done)
	}()

	// upgrade only once the basics are good to go
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		prometheus.ServiceOperationCounter.WithLabelValues("signal_ws", "error", "upgrade").Add(1)
		pLogger.Warnw("could not upgrade to WS", err)
		handleError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sigConn := NewWSSignalConnection(conn)
	if types.ProtocolVersion(pi.Client.Protocol).SupportsProtobuf() {
		sigConn.useJSON = false
	}

	prometheus.ServiceOperationCounter.WithLabelValues("signal_ws", "success", "").Add(1)
	pLogger.Infow("new client WS connected",
		"connID", connId,
		"roomID", rm.Sid,
	)

	// handle responses
	go func() {
		defer func() {
			// when the source is terminated, this means Participant.Close had been called and RTC connection is done
			// we would terminate the signal connection as well
			_ = conn.Close()
		}()
		defer rtc.Recover()
		for {
			select {
			case <-done:
				return
			case msg := <-resSource.ReadChan():
				if msg == nil {
					pLogger.Infow("source closed connection",
						"connID", connId)
					return
				}
				res, ok := msg.(*livekit.SignalResponse)
				if !ok {
					pLogger.Errorw("unexpected message type", nil,
						"type", fmt.Sprintf("%T", msg),
						"connID", connId)
					continue
				}

				if err = sigConn.WriteResponse(res); err != nil {
					pLogger.Warnw("error writing to websocket", err)
					return
				}
			}
		}
	}()

	// handle incoming requests from websocket
	for {
		req, err := sigConn.ReadRequest()
		// normal closure
		if err != nil {
			if err == io.EOF || strings.HasSuffix(err.Error(), "use of closed network connection") ||
				websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				return
			} else {
				pLogger.Errorw("error reading from websocket", err)
				return
			}
		}
		if err := reqSink.WriteMessage(req); err != nil {
			pLogger.Warnw("error writing to request sink", err,
				"connID", connId)
		}
	}
}

func (s *RTCService) ParseClientInfo(r *http.Request) *livekit.ClientInfo {
	values := r.Form
	ci := &livekit.ClientInfo{}
	if pv, err := strconv.Atoi(values.Get("protocol")); err == nil {
		ci.Protocol = int32(pv)
	}
	sdkString := values.Get("sdk")
	switch sdkString {
	case "js":
		ci.Sdk = livekit.ClientInfo_JS
	case "ios", "swift":
		ci.Sdk = livekit.ClientInfo_SWIFT
	case "android":
		ci.Sdk = livekit.ClientInfo_ANDROID
	case "flutter":
		ci.Sdk = livekit.ClientInfo_FLUTTER
	case "go":
		ci.Sdk = livekit.ClientInfo_GO
	case "unity":
		ci.Sdk = livekit.ClientInfo_UNITY
	}

	ci.Version = values.Get("version")
	ci.Os = values.Get("os")
	ci.OsVersion = values.Get("os_version")
	ci.Browser = values.Get("browser")
	ci.BrowserVersion = values.Get("browser_version")
	ci.DeviceModel = values.Get("device_model")
	// get real address (forwarded http header) - check Cloudfare headers first, fall back to X-Forwaded-For
	ci.Address = r.Header.Get("CF-Connecting-IP")
	if len(ci.Address) == 0 {
		ci.Address = xff.GetRemoteAddr(r)
	}

	// attempt to parse types for SDKs that support browser as a platform
	if ci.Sdk == livekit.ClientInfo_JS ||
		ci.Sdk == livekit.ClientInfo_FLUTTER ||
		ci.Sdk == livekit.ClientInfo_UNITY {
		client := s.parser.Parse(r.UserAgent())
		if ci.Browser == "" {
			ci.Browser = client.UserAgent.Family
			ci.BrowserVersion = client.UserAgent.ToVersionString()
		}
		if ci.Os == "" {
			ci.Os = client.Os.Family
			ci.OsVersion = client.Os.ToVersionString()
		}
		if ci.DeviceModel == "" {
			model := client.Device.Family
			if model != "" && client.Device.Model != "" && model != client.Device.Model {
				model += " " + client.Device.Model
			}

			ci.DeviceModel = model
		}
	}

	return ci
}
