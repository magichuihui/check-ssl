package config

type Config struct {
	AliDNS struct {
		AccessKey    string `required:"true"`
		AccessSecret string `required:"true"`
		DomainName   string `required:"true"`
	}

	WorkWX struct {
		Enabled bool `default:"false"`
		BotKey  string
	}
}
