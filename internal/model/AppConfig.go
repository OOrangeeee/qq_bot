package model

type AppConfig struct {
	Hmac      hmac           `json:"hmac"`
	DataBase  dataBaseConfig `json:"data-base"`
	Redis     redisConfig    `json:"redis"`
	Github    githubConfig   `json:"github"`
	Character character      `json:"character"`
	Llm       llm            `json:"llm"`
}

type hmac struct {
	Key string `json:"key"`
}

type dataBaseConfig struct {
	DevDsn string `json:"dev-dsn"`
	ProDsn string `json:"pro-dsn"`
}

type redisConfig struct {
	DevDsn                       string  `json:"dev-dsn"`
	ProDsn                       string  `json:"pro-dsn"`
	NumOfWorker                  int     `json:"num-of-worker"`
	TaskChannelSize              int     `json:"task-channel-size"`
	BloomFilterCapacity          uint    `json:"bloom-filter-capacity"`
	BloomFilterFalsePositiveRate float64 `json:"bloom-filter-false-positive-rate"`
}

type githubConfig struct {
	Token string `json:"token"`
}

type character struct {
	Describe string `json:"describe"`
}

type llm struct {
	Secret     string `json:"secret"`
	VipQQ      string `json:"vipqq"`
	VipMessage string `json:"vip-message"`
}
