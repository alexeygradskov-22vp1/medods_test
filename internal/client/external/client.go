package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ExternalServiceClient struct {
	client  *http.Client
	baseUrl string
}

func NewClient(conf *ExternalClientConfig) *ExternalServiceClient {
	return &ExternalServiceClient{client: conf.client, baseUrl: conf.BaseUrl}
}

func (c *ExternalServiceClient) SendWebhook(path string, payload map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	url := fmt.Sprintf("%s%s", c.baseUrl, path)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
