package main

import (
	"fmt"
	"log"

	"github.com/jinzhu/configor"
	"github.com/magichuihui/check-ssl/config"
	"github.com/magichuihui/check-ssl/pkg/aliyun"
	"github.com/magichuihui/check-ssl/pkg/ssl"
	"github.com/magichuihui/check-ssl/pkg/workwx"
)

var c = &config.Config{}

func init() {
	configor.Load(&c, "config.yaml")
	fmt.Println(c)
}

func main() {
	ali, err := aliyun.NewClient(c)
	if err != nil {
		panic(err.Error())
	}

	// 从阿里云接口获取域名
	ali.CreateDomainRecords(c.AliDNS.DomainName, "A")
	ali.CreateDomainRecords(c.AliDNS.DomainName, "CNAME")

	var checker ssl.SSLChecker
	checker.AppendHosts(ali.Records)
	errMessage := checker.ProcessHosts()

	fmt.Println(errMessage)

	// 是否发送到企业微信
	if errMessage != "" && c.WorkWX.Enabled {
		buf := []byte(errMessage)
		m, _ := workwx.NewMediaFromBuffer("tls_errors_when_connecting.txt", buf)

		wx := workwx.NewWorkWX(c)
		resp, _ := wx.UploadTempFileMedia(m)

		file := workwx.FileMessage{
			Message: workwx.Message{MsgType: "file"},
			File:    workwx.File{MediaID: resp.MediaID},
		}
		err := wx.Send(file)

		if err != nil {
			log.Fatal(err)
		}
	}
}
