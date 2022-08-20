package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Maksat-luci/Telegram-Bot/pkg/client/imgur"
	"github.com/Maksat-luci/Telegram-Bot/pkg/logging"
)

// imgurService структура для работы с сервисом imgur
type imgurService struct {
	client imgur.Client
	logger *logging.Logger
}

// NewImgurService конструктор для нашей структуры
func NewImgurService(client imgur.Client, logger *logging.Logger) ImgurService {
	return &imgurService{
		client: client,
		logger: logger,
	}
}

// ImgurService интерфейс работы с сервисом
type ImgurService interface {
	ShareImage(ctx context.Context, image []byte) (string, error)
}

func (i *imgurService) ShareImage(ctx context.Context, image []byte) (string, error) {
	response, err := i.client.UploadImage(ctx, image)
	fmt.Println(response)

	if err != nil {
		return "", err
	}
	var responseData map[string]interface{}

	if err = json.NewDecoder(response.Body).Decode(&responseData); err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		i.logger.Error(responseData)
		return "", fmt.Errorf("failed to upload image")
	}

	return responseData["data"].(map[string]interface{})["link"].(string), nil
}