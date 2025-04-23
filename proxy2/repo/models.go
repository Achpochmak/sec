package repo

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	DatabaseName        = "http_proxy"
	RequestsCollection  = "requests"
	ResponsesCollection = "responses"
)

type RequestData struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Method     string             `bson:"method"`
	Scheme     string             `bson:"scheme"`
	Host       string             `bson:"host"`
	Path       string             `bson:"path"`
	GetParams  bson.M             `bson:"get_params"`
	Headers    bson.M             `bson:"headers"`
	Cookies    map[string]string  `bson:"cookies"`
	PostParams bson.M             `bson:"post_params,omitempty"`
	Body       string             `bson:"body,omitempty"`
	Timestamp  primitive.DateTime `bson:"timestamp"`
}

type ResponseData struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	RequestID primitive.ObjectID `bson:"request_id"`
	Code      int                `bson:"code"`
	Message   string             `bson:"message"`
	Headers   bson.M             `bson:"headers"`
	Body      string             `bson:"body"`
	Timestamp primitive.DateTime `bson:"timestamp"`
}
