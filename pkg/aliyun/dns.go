package aliyun

import (
	"fmt"

	alidns20150109 "github.com/alibabacloud-go/alidns-20150109/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/magichuihui/check-ssl/config"
)

var (
	PAGE_SIZE int64 = 100
)

type AliClient struct {
	*alidns20150109.Client
	Records []string `json:"Records"`
}

/**
 * 使用AK&SK初始化账号Client
 * @param accessKeyId
 * @param accessKeySecret
 * @return Client
 * @throws Exception
 */
func NewClient(c *config.Config) (aliClient *AliClient, err error) {
	config := &openapi.Config{
		// 您的AccessKey ID
		AccessKeyId: tea.String(c.AliDNS.AccessKey),
		// 您的AccessKey Secret
		AccessKeySecret: tea.String(c.AliDNS.AccessSecret),
	}
	// 访问的域名
	config.Endpoint = tea.String("alidns.cn-shanghai.aliyuncs.com")
	client, err := alidns20150109.NewClient(config)

	aliClient = &AliClient{}
	aliClient.Client = client
	return aliClient, err
}

func (ali *AliClient) CreateDomainRecords(domain string, typeKeyword string) {
	var pageNumber int64 = 1

	for {

		describeDomainRecordsRequest := &alidns20150109.DescribeDomainRecordsRequest{
			DomainName:  tea.String(domain),
			TypeKeyWord: tea.String(typeKeyword),
			PageSize:    tea.Int64(PAGE_SIZE),
			PageNumber:  tea.Int64(pageNumber),
		}

		// 复制代码运行请自行打印 API 的返回值
		resp, _err := ali.Client.DescribeDomainRecords(describeDomainRecordsRequest)

		if _err != nil {
			fmt.Println(_err.Error())
		}

		for _, record := range resp.Body.DomainRecords.Record {
			ali.Records = append(ali.Records, *record.RR+"."+*record.DomainName+":443")
		}

		// 遍历完全部dns记录之后跳出
		if pageNumber*PAGE_SIZE >= *resp.Body.TotalCount {
			break
		}

		// 获取下一页的dns记录
		pageNumber += 1
	}
}

func (ali *AliClient) Output() {
	fmt.Println(len(ali.Records))
}
