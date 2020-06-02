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

	proxy.mux.HandleFunc("/", proxy.handleFunc)
	return proxy
}

func (s *proxyServer) handleFunc(w http.ResponseWriter, r *http.Request) {
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

	ctx := context.WithValue(r.Context(), "remote", remote)
	ctx = metadata.AppendToOutgoingContext(ctx, "remote", remote)

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
			socket.WriteMessage([]byte(msg.Payload))
		}
	}()

	if err := s.connect(ctx); err != nil {
		logger.Warn(err)
		return
	}

	defer s.disconnect(ctx)

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

	req := &apipb.Request{}
	err = jsonpb.Unmarshal(bytes.NewReader(reqData), req)
	if err != nil {
		if r, err := responseWithError(wsocket, req, status.Error(3, "bad request")); err != nil {
			return err
		} else {
			s.logger.Bg().Warn(string(r))
		}
		return nil
	}

	span, ctx, logger := s.logger.StartForWithTracer(ctx, s.tracer, req.Type)
	span.SetTag("remote", ctx.Value("remote"))
	defer span.Finish()

	logger.Info(string(reqData))
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

func (s *proxyServer) connect(ctx context.Context) error {
	span, ctx, _ := s.logger.StartForWithTracer(ctx, s.tracer, "Connect")
	span.SetTag("remote", ctx.Value("remote"))
	defer span.Finish()

	// send connection
	_, err := s.api.Connect(ctx, &apipb.Request{})

	return err
}

func (s *proxyServer) disconnect(ctx context.Context) error {
	span, ctx, _ := s.logger.StartForWithTracer(ctx, s.tracer, "Disconnect")
	span.SetTag("remote", ctx.Value("remote"))
	defer span.Finish()

	// send connection
	_, err := s.api.Disconnect(ctx, &apipb.Request{})

	return err
}

func (s *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

type responseWrapper struct {
	*apipb.Response
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

func responseWithError(wsocket *socket, req *apipb.Request, err error) ([]byte, error) {
	res := &responseWrapper{
		Response: &apipb.Response{
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
