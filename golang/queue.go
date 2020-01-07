package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger"
	"github.com/dyninc/qstring"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
	"github.com/golang/snappy"
)

type Search struct {
	Keywords []string `json:"keywords"`
}

type DockerResults struct {
	Results    []DockerImage `json:"results"`
	Query      string        `json:"query"`
	LastPage   int           `json:"num_pages"`
	NumResults int           `json:"num_results"`
	PageSize   int           `json:"page_size"`
}

type DockerImage struct {
	Description string
	IsOfficial  bool   `json:"is_official"`
	IsTrusted   bool   `json:"is_trusted"`
	Name        string `json:"name"`
	StarCount   int    `json:"star_count"`
	Dockerfile  string `json:"dockerfile"`
}

type DockerFile struct {
	Contents string `json:"contents"`
}

// https://hub.docker.com/v2/repositories/aaronshaf/dynamodb-admin/
type DockerRepo struct {
	Affiliation     interface{}           `json:"affiliation"`
	CanEdit         bool                  `json:"can_edit"`
	Description     string                `json:"description"`
	FullDescription string                `json:"full_description"`
	HasStarred      bool                  `json:"has_starred"`
	IsAutomated     bool                  `json:"is_automated"`
	IsMigrated      bool                  `json:"is_migrated"`
	IsPrivate       bool                  `json:"is_private"`
	LastUpdated     string                `json:"last_updated"`
	Name            string                `json:"name"`
	Namespace       string                `json:"namespace"`
	Permissions     DockerRepoPermissions `json:"permissions"`
	PullCount       int                   `json:"pull_count"`
	RepositoryType  string                `json:"repository_type"`
	StarCount       int                   `json:"star_count"`
	Status          int                   `json:"status"`
	User            string                `json:"user"`
}

type DockerRepoPermissions struct {
	Admin bool `json:"admin"`
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

// https://hub.docker.com/v2/users/aaronshaf/
type DockerUser struct {
	Company     string `json:"company"`
	DateJoined  string `json:"date_joined"`
	FullName    string `json:"full_name"`
	GravatarURL string `json:"gravatar_url"`
	ID          string `json:"id"`
	Location    string `json:"location"`
	ProfileURL  string `json:"profile_url"`
	Type        string `json:"type"`
	Username    string `json:"username"`
}

// https://hub.docker.com/api/build/v1/source/?image=byrnedo%2Falpine-curl
type DockerBuild struct {
	Meta    DockerBuildMeta     `json:"meta"`
	Objects []DockerBuildObject `json:"objects"`
}

type DockerBuildMeta struct {
	Limit      int         `json:"limit"`
	Next       interface{} `json:"next"`
	Offset     int         `json:"offset"`
	Previous   interface{} `json:"previous"`
	TotalCount int         `json:"total_count"`
}

type DockerBuildObject struct {
	Autotests     string   `json:"autotests"`
	BuildInFarm   bool     `json:"build_in_farm"`
	BuildSettings []string `json:"build_settings"`
	Channel       string   `json:"channel"`
	Deploykey     string   `json:"deploykey"`
	Image         string   `json:"image"`
	Owner         string   `json:"owner"`
	Provider      string   `json:"provider"`
	RepoLinks     bool     `json:"repo_links"`
	Repository    string   `json:"repository"`
	ResourceURI   string   `json:"resource_uri"`
	State         string   `json:"state"`
	UUID          string   `json:"uuid"`
}

func searchDockerHub(filePath string) {

	count := 1
	// Open our jsonFile
	// filePath := "search-terms.json"
	jsonFile, err := os.Open(filePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened file: ", filePath)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var search Search
	err = json.Unmarshal(byteValue, &search)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v\n", err)
	}

	// Instantiate default collector
	c := colly.NewCollector(
		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted
		colly.CacheDir("./data/cache"),
	)

	if torProxy {
		rp, err := proxy.RoundRobinProxySwitcher("socks5://127.0.0.1:9050")
		if err != nil {
			log.Fatal(err)
		}
		c.SetProxyFunc(rp)
	}

	// create a request queue with 2 consumer threads
	q, _ := queue.New(
		64, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 1500000}, // Use default queue storage
	)

	visited := 1
	skipped := 1
	c.OnRequest(func(r *colly.Request) {
		// log.Println("visiting", r.URL)
		visited++
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnResponse(func(r *colly.Response) {
		if torProxy {
			log.Printf("Proxy Address: %s\n", r.Request.ProxyURL)
		}
		// fmt.Println("r.Ctx.Get(\"url\")", r.Ctx.Get("url"))
		currentPage := 1

		// userInfo
		// if strings.HasPrefix(r.Request.URL.String(), "https://hub.docker.com/v2/users") {
		// }

		// vcsInfo
		// if strings.HasPrefix(r.Request.URL.String(), "https://hub.docker.com/api/build/v1/source/?image=") {
		// }

		if strings.HasPrefix(r.Request.URL.String(), "https://hub.docker.com/v2/repositories") {
			var dockerfile DockerFile
			err := json.Unmarshal(r.Body, &dockerfile)
			if err != nil {
				log.Fatalln("error json", err)
			}
			image := strings.Replace(r.Request.URL.String(), "https://hub.docker.com/v2/repositories/", "", -1)
			image = strings.Replace(image, "dockerfile/", "", -1)
			if dockerfile.Contents != "" {
				err = store.Update(func(txn *badger.Txn) error {
					percentageLoss := count * 100 / skipped
					log.Println("indexing [", count, " / ", skipped, " / ", visited, " / ", percentageLoss, "%] dockerfile to key:", image+"/dockerfile-content")
					// log.Println("dockerfile: \n", dockerfile.Contents)
					err := txn.Set([]byte(image+"/dockerfile-content"), []byte(dockerfile.Contents))
					if err == nil {
						count++
					}
					return err
				})
				if err != nil {
					log.Fatalln("error badger", err)
				}

				// user info
				// https://hub.docker.com/v2/users/aaronshaf/
				repoInfo := strings.Split(image, "/")
				users := fmt.Sprintf("https://hub.docker.com/v2/users/%s/", repoInfo[0])
				// log.Println("enqueue docker userinfo", users)
				q.AddURL(users)

				// github info
				// https://hub.docker.com/api/build/v1/source/?image=byrnedo%2Falpine-curl
				vcsInfo := fmt.Sprintf("https://hub.docker.com/api/build/v1/source/?image=%s/%s", repoInfo[0], repoInfo[1])
				// log.Println("enqueue docker user vcsInfo", vcsInfo)
				q.AddURL(vcsInfo)

				// docker info
				// https://hub.docker.com/v2/repositories/aaronshaf/dynamodb-admin/
				repositories := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/", repoInfo[0], repoInfo[1])
				// log.Println("enqueue repositories", repositories)
				q.AddURL(repositories)

			} else {
				skipped++
			}
			return
		}

		var res DockerResults
		err := json.Unmarshal(r.Body, &res)

		lastPage := res.LastPage
		for _, result := range res.Results {
			dockerfile := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/dockerfile/", result.Name)
			// log.Println("enqueue dockerfile info", dockerfile)
			q.AddURL(dockerfile)
			err := store.Update(func(txn *badger.Txn) error {
				// log.Println("indexing dockerimage", result.Name)
				cnt, err := compress([]byte(result.Description))
				if err != nil {
					return err
				}
				err = txn.Set([]byte(result.Name), cnt)
				return err
			})
			if err != nil {
				log.Fatalln("error badger", err)
			}

		}

		for currentPage <= lastPage {
			// var res DockerResults
			// err := json.Unmarshal(r.Body, &res)
			if nil == err {
				lastPage = res.LastPage
				// fmt.Println(fmt.Sprintf("%s&page=%v", r.Request.URL, currentPage))
				if !strings.Contains(r.Request.URL.String(), "page=") {
					url := fmt.Sprintf("%s&page=%v", r.Request.URL, currentPage)
					urlSanitized, err := sanitizeQuery(url)
					if err != nil {
						log.Fatalln("error urlSanitized", err)
					}
					q.AddURL(urlSanitized)
				}
				// c.Images = append(c.Images, res.Results...)
				// c.Results = append(c.Results, res.Results...)
			} else {
				log.Println("error: ", string(r.Body))
				log.Fatalln("error json", err)
			}
			currentPage++
		}

		// log.Printf("%s\n", bytes.Replace(r.Body, []byte("\n"), nil, -1))
	})

	for _, keyword := range search.Keywords {
		// Add URLs to the queue
		q.AddURL(fmt.Sprintf("https://index.docker.io/v1/search?q=%s&n=1000", keyword))
	}
	// Consume URLs
	q.Run(c)

}

func compress(data []byte) ([]byte, error) {
	return snappy.Encode([]byte{}, data), nil
}

func decompress(data []byte) ([]byte, error) {
	return snappy.Decode([]byte{}, data)
}

// Query is the http request query struct.
type Query struct {
	Q    string
	N    int
	Page int
}

func sanitizeQuery(href string) (string, error) {
	query := &Query{}
	u, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	err = qstring.Unmarshal(u.Query(), query)
	if err != nil {
		return "", err
	}
	q, err := qstring.MarshalString(query)
	href = fmt.Sprintf("https://index.docker.io/v1/search?%s", q)
	return href, nil
}
