package service

import (
	"GitHubBot/internal/log"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

// 请求体的结构
type RequestBody struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

// 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 响应体的结构
type ResponseBody struct {
	Choices []Choice `json:"choices"`
}

// 选择结果的结构
type Choice struct {
	Message MessageContent `json:"message"`
}

// 消息内容结构
type MessageContent struct {
	Content string `json:"content"`
}

// SendMessage 发送API请求并返回响应中的message
func SendMessage(apiKey string, messages []Message) (string, error) {
	// 构建请求体
	requestData := RequestBody{
		Model:     "glm-4-flash",
		Messages:  messages,
		MaxTokens: 4095,
	}

	// 序列化请求数据
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("error marshaling request data: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "https://open.bigmodel.cn/api/paas/v4/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// 添加鉴权头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("关闭响应体失败")
		}
	}(resp.Body)

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// 解析响应数据
	var response ResponseBody
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response data: %v", err)
	}

	// 检查响应中是否有有效的消息内容
	if len(response.Choices) > 0 && response.Choices[0].Message.Content != "" {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no valid message content found in the response")
}
