package main

import (
	"ensemble/storage"
	"ensemble/web"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

var (
	webConfiguration     web.Configuration
	storageConfiguration storage.Configuration
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	viper.SetConfigName("application")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("unable to read config file: %s", err)
	}

	webConfiguration = web.Configuration{
		DebugTemplates: false,
		Listen:         ":3000",
	}
	if err := viper.UnmarshalKey("web", &webConfiguration); err != nil {
		log.Fatalf("unable to read web configuration: %s", err)
	}

	storageConfiguration = storage.Configuration{}
	if err := viper.UnmarshalKey("db", &storageConfiguration); err != nil {
		log.Fatalf("unable to read db configuration: %s", err)
	}
	log.Infof("db: %v", storageConfiguration)
}

func main() {
	store, err := storage.New(storageConfiguration)
	if err != nil {
		log.Fatalf("unable to create storage: %s", err)
	}
	defer store.Close()

	server := web.New(webConfiguration, store)
	log.Fatal(server.Start(webConfiguration.Listen))
}
