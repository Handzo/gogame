package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/Handzo/gogame/apigateway/code"
	apipb "github.com/Handzo/gogame/apigateway/proto"
)

type MsgInfo struct {
	reqType    reflect.Type
	reqHandler grpcHandler
}

type grpcHandler func(context.Context, interface{}) (interface{}, error)

type GRPCRouter struct {
	routes map[string]*MsgInfo
}

func NewRouter() *GRPCRouter {
	return &GRPCRouter{
		routes: make(map[string]*MsgInfo),
	}
}

func (this *GRPCRouter) Register(reqTypeStr string, payloadType interface{}, reqHandler grpcHandler) {
	_, ok := this.routes[reqTypeStr]
	if ok {
		panic(fmt.Sprintf("Type %s already registered", reqTypeStr))
	}

	reqType := reflect.TypeOf(payloadType)
	if reqType == nil || reqType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Payload type must be a pointer for type: %s", reqTypeStr))
	}

	info := &MsgInfo{
		reqType:    reqType,
		reqHandler: reqHandler,
	}

	this.routes[reqTypeStr] = info
}

func (this *GRPCRouter) Route(ctx context.Context, req *apipb.Request) (*apipb.Response, error) {
	i, ok := this.routes[req.Type]
	if !ok {
		return nil, code.InvalidEventType
	}

	reqPayload, err := this.unmarshal(i.reqType, req.Payload)
	if err != nil {
		return nil, code.InvalidRequestPayloadError
	}

	response, err := i.reqHandler(ctx, reqPayload)
	if err != nil {
		return nil, err
	}

	resPayload, err := this.marshal(response)
	if err != nil {
		return nil, code.InvalidResponsePayloadError
	}

	res := &apipb.Response{
		Key:     req.Key,
		Type:    req.Type,
		Payload: resPayload,
	}

	return res, nil
}

func (this *GRPCRouter) unmarshal(reqType reflect.Type, data []byte) (interface{}, error) {
	payload := reflect.New(reqType.Elem()).Interface()
	return payload, json.Unmarshal(data, payload)
}

func (this *GRPCRouter) marshal(payload interface{}) ([]byte, error) {
	return json.Marshal(payload)
}
