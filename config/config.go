package config

import (
	"github.com/getsentry/raven-go"
	"github.com/go-ini/ini"
	"log"
	"strings"
	"time"
)

var (
	Server = struct {
		Host string
	}{}
	ErrLog = struct {
		Open      int
		Type      string
		LocalPath string
		SentryUrl string
	}{}
	Redis = struct {
		Host        string
		Password    string
		MaxIdle     int
		MaxActive   int
		IdleTimeout time.Duration
		Db          int
		Timeout     int
	}{}
)

// 加载配置信息
func Load() {
	h, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("%v", err)
	}
	mapTo(h, "server", &Server)
	mapTo(h, "errLog", &ErrLog)
	mapTo(h, "redis", &Redis)

	if strings.ToUpper(ErrLog.Type) == "SENTRY" && ErrLog.Open == 1 {
		err := raven.SetDSN(ErrLog.SentryUrl)
		if err != nil {
			log.Fatalln("Sentry错误", err.Error())
		}
	}
}

func mapTo(h *ini.File, section string, v interface{}) {
	if err := h.Section(section).MapTo(v); err != nil {
		log.Fatal(err.Error())
	}
}
