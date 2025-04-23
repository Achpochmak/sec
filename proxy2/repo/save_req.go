package repo

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const kRequests = "requests"

type MongoRequestSaver struct {
	requests *mongo.Collection
}

func NewMongoRequestSaver(conn *mongo.Client) RequestSaver {
	return &MongoRequestSaver{
		requests: conn.Database(DatabaseName).Collection(kRequests),
	}
}

func (s *MongoRequestSaver) Save(req *http.Request) (string, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}

	req.Body = io.NopCloser(bytes.NewReader(body))

	value := bson.M{
		"method":     req.Method,
		"scheme":     req.URL.Scheme,
		"host":       req.Host,
		"path":       req.URL.Path,
		"get_params": parseURLQuery(req.URL),
		"headers":    parseHTTPHeaders(req.Header),
		"cookies":    parseHTTPCookies(req.Cookies()),
	}

	postParams, err := parsePostParameters(req)
	if err != nil {
		return "", err
	}
	if len(postParams) != 0 {
		value["post_params"] = postParams
		req.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		if req.Body != nil {
			value["body"] = string(body)
		}
	}

	res, err := s.requests.InsertOne(context.Background(), value)
	if err != nil {
		return "", err
	}

	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (s *MongoRequestSaver) GetEncoded(id string) (*http.Request, error) {
	value, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	return createHTTPRequest(value)
}

func (s *MongoRequestSaver) Get(id string) (*RequestData, error) {
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	res := s.requests.FindOne(context.Background(), bson.D{{Key: "_id", Value: objectId}})
	value := &RequestData{}

	err = res.Decode(value)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (s *MongoRequestSaver) List(limit int64) ([]*RequestData, error) {
	ctx := context.Background()

	opts := options.Find().SetLimit(limit).SetSort(bson.D{{Key: "_id", Value: -1}})
	cursor, err := s.requests.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}

	res := make([]*RequestData, 0, limit/2)

	err = cursor.All(ctx, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
