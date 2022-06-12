package main

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Url          string `yaml:"url" env:"URL" env-required:"true"`
	Login        string `yaml:"login" env:"LOGIN" env-required:"true"`
	Passwd       string `yaml:"passwd" env:"PASSWD" env-required:"true"`
	VerifySSL    *bool  `yaml:"verifySSL" env:"VERIFYSSL" env-required:"true"`
}

var instance *Config
var once sync.Once

func GetConfig() *Config {
	once.Do(func() {
		log.Println("read exporter configuration")
		instance = &Config{}
		if err := cleanenv.ReadConfig("config.yaml", instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			log.Println(help)
			log.Fatal(err)
		}
	})
	return instance
}