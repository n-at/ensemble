package main

import (
	"ensemble/repository"
	"ensemble/storage"
	"ensemble/web"
	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"time"
)

var (
	webConfig        web.Configuration
	storageConfig    storage.Configuration
	repositoryConfig repository.Configuration
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

	webConfig = web.Configuration{
		DebugTemplates: false,
		Listen:         ":3000",
	}
	if err := viper.UnmarshalKey("web", &webConfig); err != nil {
		log.Fatalf("unable to read web configuration: %s", err)
	}

	storageConfig = storage.Configuration{}
	if err := viper.UnmarshalKey("db", &storageConfig); err != nil {
		log.Fatalf("unable to read db configuration: %s", err)
	}
	log.Infof("db: %v", storageConfig)

	path := viper.GetString("path")
	if len(path) == 0 {
		log.Fatalf("path not defined")
	}

	repositoryConfig = repository.Configuration{
		Path: path,
	}
}

func main() {
	store, err := storage.New(storageConfig)
	if err != nil {
		log.Fatalf("unable to create storage: %s", err)
	}
	defer store.Close()

	if err := store.UserEnsureAdminExists(); err != nil {
		log.Fatalf("unable to create admin: %s", err)
	}

	manager := repository.New(repositoryConfig, store)
	scheduleProjectsUpdate(manager)

	server := web.New(webConfig, store, manager)
	log.Fatal(server.Start(webConfig.Listen))
}

///////////////////////////////////////////////////////////////////////////////

func scheduleProjectsUpdate(m *repository.Manager) {
	cron := viper.GetString("update")
	if len(cron) == 0 {
		cron = "0 * * * *"
	}

	scheduler := gocron.NewScheduler(time.Now().Location())
	_, err := scheduler.Cron(cron).Do(func() {
		m.UpdateAll()
	})
	if err != nil {
		log.Fatalf("unable to start projecs update schedule: %s", err)
	}
}
