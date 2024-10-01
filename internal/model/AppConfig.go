package model

type AppConfig struct {
	Hmac hmac `json:"hmac"`
}

type hmac struct {
	Key string `json:"key"`
}
