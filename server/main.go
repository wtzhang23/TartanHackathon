package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	language "cloud.google.com/go/language/apiv1"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
)

const MaxRecordsPerLabel = 1000
const SalienceThreshold = .8
const SentimentFetchTimeout = 5
const RecommendationTimeout = 5
const dip = 0.1

type API = func(string, float32, float32, chan<- *url.URL)

func createFirestoreClient(ctx context.Context) *firestore.Client {
	projectID := "TartanHackathon"

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}

func main() {
	ctx := context.Background()
	firestoreClient := createFirestoreClient(ctx)
	languageHandlerChn := createLanguageHandler(ctx)
	defer firestoreClient.Close()
	defer close(languageHandlerChn)

	apis := []API{NewsRecommender(languageHandlerChn)}

	// get list of recommendations
	http.HandleFunc("/recommend", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recommend" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		// handle updating content and getting unique id
		case "POST":
			query := r.URL.Query()
			requestJSON, errJSON := IOToJson(r.Body)
			nRecommendations, errRecommendations := strconv.ParseInt(query.Get("num"), 0, 64)
			texts, textsOk := requestJSON["texts"].([]interface{})

			// check for bugs
			if errJSON != nil {
				errMsg := "Unable to read request body as JSON"
				http.Error(w, errMsg, 400)
				log.Println(errMsg)
				return
			} else if errRecommendations != nil {
				errMsg := fmt.Sprintf("Unable to parse the number of recommendations needed: %s", query.Get("num"))
				http.Error(w, errMsg, 400)
				log.Println(errMsg)
				return
			} else if !textsOk {
				errMsg := fmt.Sprintf("JSON not properly constructed. Missing `texts` field that should contain a list: %s", requestJSON)
				http.Error(w, errMsg, 400)
				log.Println(errMsg)
				return
			}

			nTexts := len(texts)
			resChn := make(chan []EntityResult)

			// send all texts for language processing
			for _, text := range texts {
				textAsMap, textOk := text.(map[string]interface{})
				if !textOk {
					errMsg := fmt.Sprintf("JSON not properly constructed. Each element in the `texts` field should be a JSON object: %s", requestJSON)
					http.Error(w, errMsg, 400)
					log.Println(errMsg)
					return
				}

				body, bodyOk := textAsMap["body"].(string)
				if !bodyOk {
					errMsg := fmt.Sprintf("JSON not properly constructed. Missing `body` field per text snippet: %s", textAsMap)
					http.Error(w, errMsg, 400)
					log.Println(errMsg)
					return
				}
				languageHandlerChn <- TextQuery{body, resChn} // request sent for language processing
			}
			// retrieve new sentiment information
			toRecommendWith := make(map[string][]float32)
			fetchTimeout := time.After(SentimentFetchTimeout * time.Second)
		EntitySentimentLoop:
			for i := 0; i < nTexts; i++ {
				select {
				case <-fetchTimeout:
					log.Println("Fetching results of sentiment analysis timed out")
					break EntitySentimentLoop
				case results := <-resChn:
					for _, res := range results {
						toRecommendWith[res.Label] = append(toRecommendWith[res.Label], res.Sentiment)
					}
				}
			}

			type SentLabelPair struct {
				label string
				mean  float32
			}

			// calculate means
			meanChn := make(chan SentLabelPair)
			nLabels := len(toRecommendWith)
			defer close(meanChn)
			for label, sentiments := range toRecommendWith {
				go func(l string, s []float32) {
					var sum float32
					for _, sentiment := range s {
						sum += sentiment
					}
					meanChn <- SentLabelPair{l, sum / float32(len(s))}
				}(label, sentiments)
			}
			means := make([]SentLabelPair, 0, nLabels)
			for i := 0; i < nLabels; i++ {
				means = append(means, <-meanChn)
			}
			rand.Shuffle(nLabels, func(i, j int) { means[i], means[j] = means[j], means[i] })

			// recommend enough
			recommendations := make(chan []*url.URL)
			defer close(recommendations)
			for i := int64(0); i < nRecommendations && i < int64(nLabels); i++ {
				go func(label string, mean float32) {
					recommendations <- recommend(label, mean, apis)
				}(means[i].label, means[i].mean)
			}
			asList := make([]string, 0)
			for i := int64(0); i < nRecommendations && i < int64(nLabels); i++ {
				for _, recommendation := range <-recommendations {
					asList = append(asList, recommendation.String())
				}
			}

			// encode recommendations
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string][]string{"urls": asList})
		default:
			http.Error(w, "Unsupported request method.", 405)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

type TextQuery struct {
	Text string
	Res  chan<- []EntityResult
}

type EntityResult struct {
	Sentiment float32
	Label     string
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

func createLanguageHandler(ctx context.Context) chan<- TextQuery {
	queryChn := make(chan TextQuery)
	client, err := language.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	} else {
		go func() {
			for query := range queryChn {
				go func() {
					sendTimeout := time.After(SentimentFetchTimeout * time.Second)

					// get sentiments of all entities
					sentimentResult, err := client.AnalyzeEntitySentiment(ctx,
						&languagepb.AnalyzeEntitySentimentRequest{
							Document: &languagepb.Document{
								Source: &languagepb.Document_Content{
									Content: query.Text,
								},
								Type: languagepb.Document_PLAIN_TEXT,
							},
							EncodingType: languagepb.EncodingType_UTF8,
						})
					res := make([]EntityResult, 0)
					if err == nil {
						// string together entities and their sentiments
						for _, entity := range sentimentResult.GetEntities() {
							salience := entity.GetSalience()
							sentiment := entity.GetSentiment().Score
							name := entity.GetName()
							log.Println(name)
							if salience >= SalienceThreshold {
								res = append(res, EntityResult{sentiment, name})
							}
						}
						log.Println("Finished sentiment analysis")
					} else {
						log.Printf("Failed to perform sentiment analysis: %s", err.Error())
					}

					select {
					case query.Res <- res:
					case <-sendTimeout:
						log.Println("Timed out sending sentiment analysis results")
					}
				}()
			}
		}()
	}
	return queryChn
}

func recommend(label string, mean float32, apis []API) []*url.URL {
	urls := make(chan *url.URL)
	defer close(urls)

	for _, api := range apis {
		go func(api API) {
			api(label, mean, dip, urls)
		}(api)
	}

	asSlice := make([]*url.URL, 0)
	recommendTimeout := time.After(RecommendationTimeout * time.Second)
RecommendLoop:
	for {
		select {
		case url, ok := <-urls:
			if !ok {
				break RecommendLoop
			}
			asSlice = append(asSlice, url)
		case <-recommendTimeout:
			break RecommendLoop
		}
	}
	return asSlice
}
