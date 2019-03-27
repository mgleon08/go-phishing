package db

import (
	"log"

	// 把 LedisDB import 進來
	"github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
)

var db *ledis.DB

func Connect() {
	// 建立一個設定檔，把資料的儲存位置設定到 ./db-data
	cfg := config.NewConfigDefault()
	cfg.DataDir = "./db_data"

	// 要求建立連線
	l, _ := ledis.Open(cfg)
	_db, err := l.Select(0)

	if err != nil {
		panic(err)
	}

	// 成功建立連線，把 db 存到全域變數，之後其他地方會用到
	db = _db
	log.Println("Connect to db successfully")
}
