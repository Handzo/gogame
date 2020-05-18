package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	apipb "github.com/Handzo/gogame/apigateway/proto"
	"github.com/Handzo/gogame/common/log"
	"github.com/go-redis/redis"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/websocket"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type proxyServer struct {
	mux    *http.ServeMux
	api    apipb.ApiGatewayServiceClient
	redis  *redis.Client
	logger log.Factory
	tracer opentracing.Tracer
}

func NewProxyServer(api apipb.ApiGatewayServiceClient, redis *redis.Client, logger log.Factory, tracer opentracing.Tracer) *proxyServer {
	proxy := &proxyServer{
		mux:    http.NewServeMux(),
		api:    api,
		redis:  redis,
		logger: logger,
		tracer: tracer,
	}

	proxy.mux.HandleFunc("/", proxy.connect)
	return proxy
}

func (s *proxyServer) connect(w http.ResponseWriter, r *http.Request) {
	// TODO: set span for socket connection?
	logger := s.logger.Bg()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warn("socket upgrade error", log.Error(err))
		return
	}

	remote := conn.RemoteAddr().String()

	// pubsub
	pubsub := s.redis.Subscribe(remote)

	_, err = pubsub.Receive()
	if err != nil {
		logger.Error(err)
		return
	}

	defer pubsub.Close()

	ctx := metadata.AppendToOutgoingContext(r.Context(), "remote", remote)

	socket := newSocket(conn)
	defer socket.close()
	defer logger.Info("socket closed")

	pch := pubsub.Channel()
	go func() {
		for {
			msg, ok := <-pch
			if !ok {
				break
			}
			response(socket, msg.Payload)
		}
	}()

	for {
		if err := s.listen(ctx, socket); err != nil {
			logger.Warn(err)
			break
		}
	}
}

func (s *proxyServer) listen(ctx context.Context, wsocket *socket) error {
	reqData, err := wsocket.ReadMessage()
	if err != nil {
		return err
	}

	span, ctx, logger := s.logger.StartForWithTracer(ctx, s.tracer, "proxy")
	defer span.Finish()

	logger.Info(string(reqData))

	// TODO: return error?
	// wsocket.WriteMessage(reqData)

	req := &apipb.Request{}
	err = jsonpb.Unmarshal(bytes.NewReader(reqData), req)
	if err != nil {
		if r, err := responseWithError(wsocket, req, status.Error(3, "bad request")); err != nil {
			return err
		} else {
			logger.Warn(string(r))
		}
		return nil
	}

	logger.Info(req)

	res, err := s.api.Send(opentracing.ContextWithSpan(ctx, span), req)

	if err != nil {
		if r, err := responseWithError(wsocket, req, err); err != nil {
			return err
		} else {
			logger.Warn(string(r))
		}
		return nil
	}

	logger.Info(res)

	r, err := response(wsocket, res)
	if err != nil {
		return err
	}

	logger.Info(string(r))
	return nil
}

func (s *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type responseWrapper struct {
	*apipb.Request
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

func responseWithError(wsocket *socket, req *apipb.Request, err error) ([]byte, error) {
	res := &responseWrapper{
		Request: &apipb.Request{
			Key:  req.Key,
			Type: req.Type,
		},
	}

	if e, ok := status.FromError(err); ok {
		res.Code = uint32(e.Code())
		res.Message = e.Message()
	} else {
		res.Code = 13
		res.Message = "internal error"
	}

	return response(wsocket, res)
}

func response(wsocket *socket, res interface{}) ([]byte, error) {
	resdata, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	wsocket.WriteMessage(resdata)

	return resdata, nil
}
