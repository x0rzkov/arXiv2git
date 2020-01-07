package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
)

const service = "https://gimmeproxy.com/api/getProxy"

// Proxy represents the core of the library.
type Proxy struct {
	success int
	service string
	failure []error
	entries []ProxySettings
}

// NewProxy returns an instance of Proxy.
func NewProxy(url string) *Proxy {
	p := new(Proxy)
	p.service = url
	p.failure = make([]error, 0)
	p.entries = make([]ProxySettings, 0)
	return p
}

// Execute returns a list with N proxy settings.
func (p *Proxy) Execute(n int) error {
	fails := make(chan error, n)
	queue := make(chan ProxySettings, n)

	for i := 0; i < n; i++ {
		go p.Fetch(fails, queue)
	}

	var fail error
	var item ProxySettings

	for i := 0; i < n; i++ {
		fail = <-fails
		item = <-queue
		p.failure = append(p.failure, fail)
		p.entries = append(p.entries, item)
		if item.Curl != "" {
			p.success++
		}
	}

	var msg string

	for _, err := range p.failure {
		if err != nil {
			msg += "\xe2\x80\xa2\x20" + err.Error() + "\n"
		}
	}

	if msg == "" {
		return nil
	}

	return errors.New(msg)
}

// Fetch queries a web API service to get one proxy.
func (p *Proxy) Fetch(fails chan error, queue chan ProxySettings) {
	client := &http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest("GET", p.service, nil)

	if err != nil {
		fails <- err
		queue <- ProxySettings{}
		return
	}

	req.Header.Set("Host", "gimmeproxy.com")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/11.1.2 Safari/605.1.15")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-us")

	resp, err := client.Do(req)

	if err != nil {
		fails <- err
		queue <- ProxySettings{}
		return
	}

	defer resp.Body.Close()

	var v ProxySettings

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		fails <- err
		queue <- ProxySettings{}
		return
	}

	if v.StatusCode == 429 {
		fails <- errors.New(v.StatusMessage)
		queue <- ProxySettings{}
		return
	}

	fails <- nil
	queue <- v
}

// ToSlice export the list of proxies to a slice
func (p *Proxy) ToSlice() (proxies []string) {
	for _, item := range p.entries {
		if item.Curl == "" {
			continue
		}
		proxies = append(proxies, item.Curl)
	}
	return
}

// Export writes the list of proxies into W in JSON format.
func (p *Proxy) Export(w io.Writer) {
	if err := json.NewEncoder(w).Encode(p.entries); err != nil {
		log.Println("json.decode", err)
	}
}

// Print writes the list of proxies into W in Tabular format.
func (p *Proxy) Print(w io.Writer) {
	var entry []string

	data := [][]string{}

	for _, item := range p.entries {
		if item.Curl == "" {
			continue
		}

		entry = []string{}

		t := time.Unix(item.TsChecked, 0)

		entry = append(entry, item.Country)
		entry = append(entry, item.Curl)
		entry = append(entry, fmt.Sprintf("%.2f", item.Speed))
		entry = append(entry, time.Since(t).String())

		if item.Get {
			entry = append(entry, "\x1b[48;5;010m░░\x1b[0m")
		} else {
			entry = append(entry, "\x1b[48;5;009m░░\x1b[0m")
		}

		if item.Post {
			entry = append(entry, "\x1b[48;5;010m░░\x1b[0m")
		} else {
			entry = append(entry, "\x1b[48;5;009m░░\x1b[0m")
		}

		if item.Cookies {
			entry = append(entry, "\x1b[48;5;010m░░\x1b[0m")
		} else {
			entry = append(entry, "\x1b[48;5;009m░░\x1b[0m")
		}

		if item.Referer {
			entry = append(entry, "\x1b[48;5;010m░░\x1b[0m")
		} else {
			entry = append(entry, "\x1b[48;5;009m░░\x1b[0m")
		}

		if item.UserAgent {
			entry = append(entry, "\x1b[48;5;010m░░\x1b[0m")
		} else {
			entry = append(entry, "\x1b[48;5;009m░░\x1b[0m")
		}

		if item.AnonymityLevel == 1 {
			entry = append(entry, "\x1b[48;5;010m░░\x1b[0m")
		} else {
			entry = append(entry, "\x1b[48;5;009m░░\x1b[0m")
		}

		data = append(data, entry)
	}

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{
		"Country",
		"cURL",
		"Speed",
		"Uptime",
		"G", //            get: bool - supports GET requests
		"P", //           post: bool - supports POST requests
		"C", //        cookies: bool - supports cookies
		"R", //        referer: bool - supports 'referer' header
		"U", //      userAgent: bool - supports 'user-agent' header
		"A", // anonymityLevel:  int - 1:anonymous, 0:notanonymous
	})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()

	if len(data) > 0 {
		fmt.Fprintln(w, "G - supports GET requests")
		fmt.Fprintln(w, "P - supports POST requests")
		fmt.Fprintln(w, "C - supports cookies")
		fmt.Fprintln(w, "R - supports 'referer' header")
		fmt.Fprintln(w, "U - supports 'user-agent' header")
		fmt.Fprintln(w, "A - 1:anonymous, 0:notanonymous")
	}
}

// Sort re-orders the list of proxies by a column.
func (p *Proxy) Sort(column string) {
	for idx, item := range p.entries {
		switch column {
		case "port":
			p.entries[idx].Filter = item.Port
		case "speed":
			p.entries[idx].Filter = fmt.Sprintf("%.2f", item.Speed*100)
		case "country":
			p.entries[idx].Filter = item.Country
		case "protocol":
			p.entries[idx].Filter = item.Protocol
		case "uptime":
			p.entries[idx].Filter = fmt.Sprintf("%d", item.TsChecked)
		}
	}

	sort.Sort(byFilter(p.entries))
}

type byFilter []ProxySettings

func (s byFilter) Len() int {
	return len(s)
}

func (s byFilter) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byFilter) Less(i int, j int) bool {
	return s[i].Filter < s[j].Filter
}

// ProxySettings represents the configuration for the proxy.
type ProxySettings struct {
	Protocol       string          `json:"protocol"`
	IP             string          `json:"ip"`
	Type           string          `json:"type"`
	Port           string          `json:"port"`
	Curl           string          `json:"curl"`
	IPPort         string          `json:"ipPort"`
	Filter         string          `json:"_filter"`
	Country        string          `json:"country"`
	StatusMessage  string          `json:"status_message"`
	AnonymityLevel int             `json:"anonymityLevel"`
	StatusCode     int             `json:"status_code"`
	TsChecked      int64           `json:"tsChecked"`
	Speed          float64         `json:"speed"`
	Get            bool            `json:"get"`
	Post           bool            `json:"post"`
	Cookies        bool            `json:"cookies"`
	Referer        bool            `json:"referer"`
	UserAgent      bool            `json:"user-agent"`
	SupportsHTTPS  bool            `json:"supportsHttps"`
	Websites       map[string]bool `json:"websites"`
}

var htmlTableRowWithIPAndPort = regexp.MustCompile(`<td>(\d+\.\d+\d\.\d+\.\d+)</td><td>(\d+)`)

func getProxy() (*url.URL, error) {
	urls, err := parseFreeProxyList()
	if err != nil {
		return nil, err
	}

	ctx, cancelProxyChecks := context.WithCancel(context.Background())
	availableProxy := make(chan *url.URL)

	var proxyChecks sync.WaitGroup
	for _, u := range urls {
		proxyChecks.Add(1)

		go func(u *url.URL) {
			defer proxyChecks.Done()

			if isProxyAvailable(ctx, u) {
				select {
				case availableProxy <- u:
				default:
				}
			}
		}(u)
	}

	select {
	case <-allChecksFinished(&proxyChecks):
		return nil, fmt.Errorf("all proxies unavailable")
	case proxy := <-availableProxy:
		cancelProxyChecks()
		proxyChecks.Wait()
		close(availableProxy)
		return proxy, nil
	}
}

func parseFreeProxyList() ([]*url.URL, error) {
	resp, err := http.Get("https://free-proxy-list.net")
	if err != nil {
		return nil, fmt.Errorf("cannot get free proxy list: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rows := htmlTableRowWithIPAndPort.FindAllStringSubmatch(string(body), -1)
	urls := make([]*url.URL, len(rows))
	for i, row := range rows {
		ip := row[1]
		port := row[2]
		urls[i], _ = url.Parse(fmt.Sprintf("http://%s:%s", ip, port))
	}

	return urls, nil
}

func isProxyAvailable(ctx context.Context, proxy *url.URL) bool {
	withProxy := &http.Transport{Proxy: http.ProxyURL(proxy)}
	client := &http.Client{Transport: withProxy}

	req, _ := http.NewRequest("HEAD", "https://www.google.com", nil)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func allChecksFinished(checks *sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		checks.Wait()
		close(done)
	}()

	return done
}
