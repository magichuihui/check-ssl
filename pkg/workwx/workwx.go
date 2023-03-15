package workwx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/magichuihui/check-ssl/config"
)

var (
	ErrUnsupportedMessage = errors.New("尚不支持的消息类型")
)

type WorkWX struct {
	endpoint string
	botKey   string
	Client   *http.Client
}

func NewWorkWX(c *config.Config) (wx *WorkWX) {
	wx = &WorkWX{
		endpoint: "https://qyapi.weixin.qq.com",
		Client:   &http.Client{Timeout: 5 * time.Second},
		botKey:   c.WorkWX.BotKey,
	}
	return
}

func (wx *WorkWX) UploadTempFileMedia(media *Media) (resp respMediaUpload, err error) {

	url := wx.endpoint + "/cgi-bin/webhook/upload_media?key=" + wx.botKey + "&type=file"

	err = wx.executeQyapiMediaUpload(url, media, &resp)

	if err != nil {
		return respMediaUpload{}, err
	}
	return resp, err
}

func (wx *WorkWX) SendFileMessage(mediaID string) {

}

func (wx *WorkWX) executeQyapiMediaUpload(
	path string,
	media *Media,
	respObj interface{},
) error {
	buf := bytes.Buffer{}
	mw := multipart.NewWriter(&buf)

	err := media.writeTo(mw)
	if err != nil {
		return err
	}

	err = mw.Close()
	if err != nil {
		return err
	}

	resp, err := wx.Client.Post(path, mw.FormDataContentType(), &buf)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(respObj)
	if err != nil {
		return err
	}

	return nil
}

// 发送消息，允许的参数类型：Text、Markdown、Image、News
func (wx *WorkWX) Send(msg interface{}) error {
	msgBytes, err := marshalMessage(msg)
	if err != nil {
		return err
	}
	url := wx.endpoint + "/cgi-bin/webhook/send?key=" + wx.botKey

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wx.Client.Do(req)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	var respSendMessage respCommon
	err = json.Unmarshal(body, &respSendMessage)
	if err != nil {
		return err
	}
	if respSendMessage.ErrCode != 0 && respSendMessage.ErrMsg != "" {
		return errors.New(string(body))
	}
	return nil
}

// 防止 < > 等 HTML 字符在 json.marshal 时被 escape
func marshal(msg interface{}) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(buf)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "")
	err := jsonEncoder.Encode(msg)
	if err != nil {
		return nil, nil
	}
	return buf.Bytes(), nil
}

// 将消息包装成企信接口要求的格式，返回 json bytes
func marshalMessage(msg interface{}) ([]byte, error) {
	if text, ok := msg.(Text); ok {
		textMsg := TextMessage{
			Message: Message{MsgType: "text"},
			Text:    text,
		}
		return marshal(textMsg)
	}
	if textMsg, ok := msg.(TextMessage); ok {
		textMsg.MsgType = "text"
		return marshal(textMsg)
	}
	if file, ok := msg.(File); ok {
		fileMsg := FileMessage{
			Message: Message{MsgType: "file"},
			File:    file,
		}
		return marshal(fileMsg)
	}
	if fileMsg, ok := msg.(FileMessage); ok {
		fileMsg.MsgType = "file"
		return marshal(fileMsg)
	}

	return nil, ErrUnsupportedMessage
}
