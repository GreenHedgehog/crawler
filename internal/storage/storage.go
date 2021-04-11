package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"crawler/internal/logger"
	"crawler/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Interface interface {
	Insert(ctx context.Context, list []*models.Product) error
	List(ctx context.Context, params *models.SearchParams, pageToken string, pageSize uint32) (list []*models.Product, token string, err error)
}

type mongodb struct {
	client *mongo.Client
}

func New(uri string) (Interface, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("init clinet failed: %w", err)
	}
	if err = client.Connect(context.TODO()); err != nil {
		return nil, fmt.Errorf("conntect failed: %w", err)
	}
	return &mongodb{client: client}, nil
}

func (m *mongodb) Insert(ctx context.Context, list []*models.Product) error {
	writes := make([]mongo.WriteModel, 0, len(list))
	for i := range list {
		model := mongo.NewUpdateOneModel()
		model.SetUpsert(true)
		model.SetFilter(bson.M{"name": list[i].Name})
		model.SetUpdate(bson.M{
			"$set": bson.M{
				"name":                  list[i].GetName(),
				"price":                 list[i].GetPrice(),
				"price_changes_counter": list[i].GetPriceChangesCounter(),
				"last_update_ts":        list[i].GetLastUpdateTs(),
			},
		})
		writes = append(writes, model)
	}

	_, err := m.collection().BulkWrite(context.TODO(), writes)
	if err != nil {
		return fmt.Errorf("bulk write failed: %w", err)
	}
	return nil
}

func (m *mongodb) List(ctx context.Context, params *models.SearchParams, pageToken string, pageSize uint32) (list []*models.Product, token string, err error) {
	var filter = bson.D{}
	if pageToken != "" {
		objectID, sortFieldLastValue, err := parsePageToken(pageToken, params.GetOrderBy())
		if err != nil {
			return nil, "", err
		}
		if params.GetOrderBy() == models.SearchByDefault {
			filter = bson.D{{
				"_id", bson.D{{"$gt", objectID}},
			}}
		} else {
			sortField := getFieldName(params.GetOrderBy())
			orderName := getOrderName(params.GetOrderMethod())
			filter = bson.D{{
				"$or", bson.A{
					bson.D{{sortField, bson.D{{orderName, sortFieldLastValue}}}},
					bson.M{
						sortField: bson.D{bson.E{"$eq", sortFieldLastValue}},
						"_id":     bson.D{bson.E{orderName, objectID}},
					},
				},
			}}
		}
	}
	data, _ := json.Marshal(filter)
	logger.G().Sugar().Infow("debug", "filter", string(data))

	var find = options.Find()
	if params.GetOrderBy() != models.SearchByDefault {
		sortField := getFieldName(params.GetOrderBy())
		order := getOrder(params.GetOrderMethod())
		find.SetSort(bson.D{{sortField, order}})
	}
	find.SetLimit(int64(pageSize))

	cursor, err := m.collection().Find(context.TODO(), filter, find)
	if err != nil {
		return nil, "", fmt.Errorf("find failed: %w", err)
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &list)
	if err != nil {
		return nil, "", fmt.Errorf("decode failed: %w", err)
	}
	if err = cursor.Err(); err != nil {
		return nil, "", fmt.Errorf("cursor error: %w", err)
	}

	if len(list) == 0 {
		return list, "", nil
	}
	return list, formPageToken(params.GetOrderBy(), list[len(list)-1]), nil
}

func (m *mongodb) collection() *mongo.Collection {
	return m.client.Database("main").Collection("products")
}

func getFieldName(field models.SearchField) string {
	switch field {
	case models.SearchByPrice:
		return "price"
	case models.SearchByPriceChange:
		return "price_changes_counter"
	case models.SearchByLastUpdate:
		return "last_update_ts"
	case models.SearchByName:
		return "name"
	}
	return ""
}

func getFieldValue(field models.SearchField, product *models.Product) string {
	switch field {
	case models.SearchByPrice:
		return strconv.FormatUint(uint64(product.Price), 10)
	case models.SearchByPriceChange:
		return strconv.FormatUint(uint64(product.PriceChangesCounter), 10)
	case models.SearchByLastUpdate:
		return strconv.FormatUint(product.LastUpdateTs, 10)
	case models.SearchByName:
		return product.Name
	}
	return ""
}

func getOrder(order models.SearchOrder) int8 {
	if order == models.SearchDescending {
		return -1
	}
	return 1
}

func getOrderName(order models.SearchOrder) string {
	if order == models.SearchDescending {
		return "$lt"
	}
	return "$gt"
}

func formPageToken(field models.SearchField, product *models.Product) string {
	if field == models.SearchByDefault {
		return product.Id
	}
	return product.Id + "_" + getFieldValue(field, product)
}

func parsePageToken(pageToken string, field models.SearchField) (objectID primitive.ObjectID, sortFieldValue interface{}, err error) {
	if field == models.SearchByDefault {
		objectID, err := primitive.ObjectIDFromHex(pageToken)
		if err != nil {
			return [12]byte{}, nil, fmt.Errorf("parse objectId failed: %w", err)
		}
		return objectID, nil, nil
	}

	tokens := strings.Split(pageToken, "_")
	if len(tokens) != 2 {
		return [12]byte{}, nil, fmt.Errorf("invalid token format")
	}

	objectID, err = primitive.ObjectIDFromHex(tokens[0])
	if err != nil {
		return [12]byte{}, nil, fmt.Errorf("parse objectId failed: %w", err)
	}

	if field == models.SearchByName {
		return objectID, tokens[1], nil
	}

	value, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return [12]byte{}, nil, fmt.Errorf("invalid token format")
	}
	return objectID, value, nil
}
