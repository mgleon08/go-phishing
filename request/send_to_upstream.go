package request

import (
	"io/ioutil"
	"net/http"
)

// 拿複製的 request 去請求真正的網站
func SendToUpstream(req *http.Request) ([]byte, http.Header, int) {
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
