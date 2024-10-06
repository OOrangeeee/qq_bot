package model

type AppConfig struct {
	Hmac      hmac           `json:"hmac"`
	DataBase  dataBaseConfig `json:"data-base"`
	Redis     redisConfig    `json:"redis"`
	Github    githubConfig   `json:"github"`
	Character character      `json:"character"`
	Llm       llm            `json:"llm"`
	QQ        qqConfig       `json:"qq"`
	Gaode     gaode          `json:"gaode"`
}

type hmac struct {
	Key string `json:"key"`
}

type dataBaseConfig struct {
	DevDsn string `json:"dev-dsn"`
	ProDsn string `json:"pro-dsn"`
}

type redisConfig struct {
	DevDsn          string `json:"dev-dsn"`
	ProDsn          string `json:"pro-dsn"`
	NumOfWorker     int    `json:"num-of-worker"`
	TaskChannelSize int    `json:"task-channel-size"`
}

type githubConfig struct {
	Token  string `json:"token"`
	ApiUrl string `json:"api-url"`
}

type character struct {
	Describe string `json:"describe"`
}

type llm struct {
	Secret     string `json:"secret"`
	VipQQ      string `json:"vipqq"`
	VipMessage string `json:"vip-message"`
	Version    string `json:"version"`
}

type qqConfig struct {
	BotUrl   string `json:"bot-url"`
	BotToken string `json:"bot-token"`
	BotQQ    string `json:"bot-qq"`
}

type gaode struct {
	Key         string `json:"key"`
	DiLiCodeUrl string `json:"di-li-code-url"`
}
