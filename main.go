package main

import (
	"ensemble/privatekeys"
	"ensemble/repository"
	"ensemble/runner"
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
	runnerConfig     runner.Configuration
	keyManagerConfig privatekeys.Configuration
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

	storageConfig = storage.Configuration{
		Url:    "",
		Secret: "",
	}
	if err := viper.UnmarshalKey("db", &storageConfig); err != nil {
		log.Fatalf("unable to read db configuration: %s", err)
	}
	if len(storageConfig.Url) == 0 {
		log.Fatalf("configuration db.url required")
	}

	keyManagerConfig = privatekeys.Configuration{
		Path:         "",
		AddKeyScript: "./ssh_add_key.sh",
		AuthSock:     "",
	}
	if err := viper.UnmarshalKey("keys", &keyManagerConfig); err != nil {
		log.Fatalf("unable to read key manager configuration: %s", err)
	}
	if len(keyManagerConfig.Path) == 0 {
		log.Fatalf("configuration keys.path required")
	}

	path := viper.GetString("path")
	if len(path) == 0 {
		log.Fatalf("configuration path required")
	}

	repositoryConfig = repository.Configuration{
		Path: path,
	}
	runnerConfig = runner.Configuration{
		Path:     path,
		AuthSock: keyManagerConfig.AuthSock,
	}
}

func main() {
	s, err := storage.New(storageConfig)
	if err != nil {
		log.Fatalf("unable to create storage: %s", err)
	}
	defer s.Close()

	if err := s.UserEnsureAdminExists(); err != nil {
		log.Fatalf("unable to create admin: %s", err)
	}

	m := repository.New(repositoryConfig, s)
	scheduleProjectsUpdate(m)

	r := runner.New(runnerConfig, s)

	km, err := privatekeys.NewKeyManager(keyManagerConfig)
	if err != nil {
		log.Fatalf("unable to create key manager: %s", err)
	}
	addPrivateKeys(s, km)

	server := web.New(webConfig, s, m, r, km)
	log.Fatal(server.Start(webConfig.Listen))
}

///////////////////////////////////////////////////////////////////////////////

func scheduleProjectsUpdate(m *repository.Manager) {
	cron := viper.GetString("update")
	if len(cron) == 0 {
		cron = "0 3 * * *"
	}

	scheduler := gocron.NewScheduler(time.Now().Location())
	_, err := scheduler.Cron(cron).Do(func() {
		m.UpdateAll()
	})
	if err != nil {
		log.Fatalf("unable to start projecs update schedule: %s", err)
	}
}

func addPrivateKeys(s *storage.Storage, km *privatekeys.KeyManager) {
	log.Infof("add all private keys...")

	keys, err := s.KeyGetAll()
	if err != nil {
		log.Fatalf("unable to read private keys: %s", err)
	}

	for _, key := range keys {
		if err := km.AddKey(key); err != nil {
			log.Errorf("unable lo add provate key %s: %s", key.Name, err)
		}
	}
}
