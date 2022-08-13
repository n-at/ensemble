package main

import (
	"ensemble/privatekeys"
	"ensemble/repository"
	"ensemble/runner"
	"ensemble/storage"
	"ensemble/web"
	"github.com/go-co-op/gocron"
	_ "github.com/joho/godotenv/autoload"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var (
	webConfig        web.Configuration
	storageConfig    storage.Configuration
	repositoryConfig repository.Configuration
	runnerConfig     runner.Configuration
	keyManagerConfig privatekeys.Configuration
	cronUpdate       string
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	webConfig = web.Configuration{
		DebugTemplates: getEnvOrDefault("ENSEMBLE_WEB_DEBUG_TEMPLATES", "0") != "0",
		Listen:         getEnvOrDefault("ENSEMBLE_WEB_LISTEN", ":3000"),
	}

	storageConfig = storage.Configuration{
		Url:    getEnvOrDefault("ENSEMBLE_DB_URL", ""),
		Secret: getEnvOrDefault("ENSEMBLE_DB_SECRET", ""),
	}
	if len(storageConfig.Url) == 0 {
		log.Fatalf("ENSEMBLE_DB_URL required")
	}

	keyManagerConfig = privatekeys.Configuration{
		Path:         getEnvOrDefault("ENSEMBLE_KEYS_PATH", ""),
		AddKeyScript: getEnvOrDefault("ENSEMBLE_KEYS_SCRIPT", "./ssh_add_key.sh"),
		AuthSock:     getEnvOrDefault("ENSEMBLE_KEYS_SOCK", ""),
	}
	if len(keyManagerConfig.Path) == 0 {
		log.Fatalf("ENSEMBLE_KEYS_PATH required")
	}

	cronUpdate = getEnvOrDefault("ENSEMBLE_CRON", "0 3 * * *")

	path := os.Getenv("ENSEMBLE_PATH")
	if len(path) == 0 {
		log.Fatalf("ENSEMBLE_PATH required")
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
	scheduler := gocron.NewScheduler(time.Now().Location())
	_, err := scheduler.Cron(cronUpdate).Do(func() {
		m.UpdateAll()
	})
	if err != nil {
		log.Fatalf("unable to start projecs update schedule: %s", err)
	}
	scheduler.StartAsync()
}

func addPrivateKeys(s *storage.Storage, km *privatekeys.KeyManager) {
	log.Infof("add all private keys...")

	keys, err := s.KeyGetAll()
	if err != nil {
		log.Fatalf("unable to read private keys: %s", err)
	}

	for _, key := range keys {
		if err := km.AddKey(key); err != nil {
			log.Errorf("unable lo add private key %s: %s", key.Name, err)
		}
	}
}

func getEnvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return def
	} else {
		return value
	}
}
