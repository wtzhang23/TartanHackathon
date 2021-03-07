package recommend

import (
	"context"
	"log"
	"net/url"
	"time"

	language "cloud.google.com/go/language/apiv1"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
)

const SentimentFetchTimeout = 5
const NEntities = 10
const RecommendationTimeout = 5
const Dip = 0.1

type API = func(string, float32, float32, chan<- []*url.URL)

type TextQuery struct {
	Text string
	Res  chan<- []EntityResult
}

type EntityResult struct {
	Sentiment float32
	Label     string
}

func CreateLanguageHandler(ctx context.Context) chan<- TextQuery {
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
						for i, entity := range sentimentResult.GetEntities() {
							if i >= NEntities {
								break
							}
							sentiment := entity.GetSentiment().Score
							name := entity.GetName()
							log.Printf("%s: %f", name, sentiment)
							res = append(res, EntityResult{sentiment, name})
						}
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

func Recommend(label string, mean float32, apis []API) []*url.URL {
	log.Printf("recommending for label %s\n", label)
	urls := make(chan []*url.URL)
	defer close(urls)

	for _, api := range apis {
		go func(api API) {
			api(label, mean, Dip, urls)
		}(api)
	}

	recommendTimeout := time.After(RecommendationTimeout * time.Second)
RecommendLoop:
	select {
	case us, ok := <-urls:
		if !ok {
			break RecommendLoop
		}
		return us
	case <-recommendTimeout:
		break RecommendLoop
	}
	return nil
}
