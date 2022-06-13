package main

import (
	"errors"
	"log"
	"os"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Url      string `yaml:"url" env:"URL" env-required:"true"`
	Login    string `yaml:"login" env:"LOGIN" env-required:"true"`
	Passwd   string `yaml:"passwd" env:"PASSWD" env-required:"true"`
	Insecure bool   `yaml:"insecure" env:"INSECURE" env-default:"false"`
}

var instance *Config
var once sync.Once

func GetConfig() *Config {
	once.Do(func() {
		log.Println("read exporter configuration")
		instance = &Config{}
		help, _ := cleanenv.GetDescription(instance, nil)

		if err := cleanenv.ReadConfig("config.yaml", instance); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if err := cleanenv.ReadEnv(instance); err != nil {
					log.Println(help)
					log.Fatal(err)
				}
			} else {
				log.Println(help)
				log.Fatal(err)
			}
		}
	})
	return instance
}
