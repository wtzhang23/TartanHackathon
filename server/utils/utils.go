package utils

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"

	"cloud.google.com/go/firestore"
)

func CreateFirestoreClient(ctx context.Context) *firestore.Client {
	projectID := "TartanHackathon"

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}

func IOToJson(body io.ReadCloser) (map[string]interface{}, error) {
	var asMap map[string]interface{}
	asBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	} else {
		json.Unmarshal(asBytes, &asMap)
		return asMap, nil
	}
}
