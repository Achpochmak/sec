package repo

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoResponseSaver struct {
	collection *mongo.Collection
}

func NewMongoResponseSaver(client *mongo.Client) ResponseSaver {
	return &MongoResponseSaver{
		collection: client.Database(DatabaseName).Collection(ResponsesCollection),
	}
}

func (s *MongoResponseSaver) Save(requestID string, resp *http.Response) (string, error) {
	requestObjectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	doc := bson.M{
		"code":       resp.StatusCode,
		"message":    strings.TrimSpace(resp.Status[strings.Index(resp.Status, " "):]),
		"headers":    convertToBSON(resp.Header),
		"request_id": requestObjectID,
		"body":       string(body),
		"timestamp":  primitive.NewDateTimeFromTime(time.Now()),
	}

	res, err := s.collection.InsertOne(context.Background(), doc)
	if err != nil {
		return "", err
	}

	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (s *MongoResponseSaver) Get(id string) (*ResponseData, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var result ResponseData
	err = s.collection.
		FindOne(context.Background(), bson.M{"_id": objectID}).
		Decode(&result)

	return &result, err
}

func (s *MongoResponseSaver) GetByRequest(requestID string) (*ResponseData, error) {
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, err
	}

	var result ResponseData
	err = s.collection.
		FindOne(context.Background(), bson.M{"request_id": objectID}).
		Decode(&result)

	return &result, err
}

func (s *MongoResponseSaver) List(limit int64) ([]*ResponseData, error) {
	opts := options.Find().
		SetLimit(limit).
		SetSort(bson.M{"_id": -1})

	ctx := context.Background()
	cursor, err := s.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*ResponseData
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
