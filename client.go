package easyclient

import (
	`errors`
	`fmt`
	`io/ioutil`
	`net/http`
	`os`
	`path/filepath`
	`regexp`
	`time`
	
	`github.com/abc1236762/go-cloudflare-scraper`
	`github.com/juju/persistent-cookiejar`
)

type Client scraper.Client

func New(needCookieJar bool, cookieFilename, userAgent string) (client *Client, err error) {
	if cookieFilename == "" {
		cookieFilename = ".cookie"
	}
	
	var jar http.CookieJar
	if needCookieJar {
		if jar, err = cookiejar.New(&cookiejar.Options{
			Filename: cookieFilename,
		}); err != nil {
			return
		}
	}
	
	return (*Client)(scraper.NewClient(jar, userAgent)), nil
}

func (c *Client) SaveCookie() error {
	if c.Jar != nil {
		return c.Jar.(*cookiejar.Jar).Save()
	}
	return nil
}

func (c *Client) getWithBody(url string) (resp *http.Response, body []byte, err error) {
	if resp, err = (*scraper.Client)(c).Get(url); err != nil {
		return
	} else if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("expected status %s", resp.Status)
	}
	
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}
	return resp, body, resp.Body.Close()
}

func (c *Client) GetWithBody(url string, retryCnt uint64,
	isRetryWhenBlank, isRetryWhenEOFNotHtml bool) (resp *http.Response, body []byte, err error) {
	for i := uint64(0); i <= retryCnt; i++ {
		if resp, body, err = c.getWithBody(url); err != nil {
			continue
		}
		
		if isRetryWhenBlank && len(body) == 0 {
			err = errors.New("body returned nothing")
		} else if isRetryWhenEOFNotHtml &&
			!regexp.MustCompile(`</\s*html>`).Match(body) {
			err = errors.New("eof of body is not matched pattern `</\\s*html>`")
		} else {
			break
		}
	}
	return
}

func (c *Client) GetWithBodyStr(url string, retryCnt uint64,
	isRetryWhenBlank, isRetryWhenEOFNotHtml bool) (resp *http.Response, bodyStr string, err error) {
	var body []byte
	if resp, body, err = c.GetWithBody(url, retryCnt,
		isRetryWhenBlank, isRetryWhenEOFNotHtml); err != nil {
		return
	}
	return resp, string(body), nil
}

func (c *Client) Download(url, path string, retryCnt uint64, needRetryWhenBlank bool) (err error) {
	var body []byte
	var resp *http.Response
	
	if resp, body, err = c.GetWithBody(
		url, retryCnt, needRetryWhenBlank, false); err != nil {
		return
	}
	
	if err = os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return
	}
	if err = ioutil.WriteFile(path, body, 0666); err != nil {
		return
	}
	
	var fileTime time.Time
	if fileTime, err = time.Parse(time.RFC1123,
		resp.Header.Get("Last-Modified")); err != nil {
		return
	}
	return os.Chtimes(path, fileTime, fileTime)
}
