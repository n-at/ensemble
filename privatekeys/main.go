package privatekeys

import (
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

type Configuration struct {
	Path         string
	AddKeyScript string
	AuthSock     string
}

type KeyManager struct {
	config Configuration
}

///////////////////////////////////////////////////////////////////////////////

func NewKeyManager(config Configuration) (*KeyManager, error) {
	ok, err := directoryExists(config.Path)
	if err != nil {
		return nil, err
	}
	if !ok {
		if err := os.Mkdir(config.Path, 0700); err != nil {
			return nil, err
		}
	}
	return &KeyManager{
		config: config,
	}, nil
}

///////////////////////////////////////////////////////////////////////////////

func (k *KeyManager) Save(key *structures.Key, content string) error {
	if err := k.SaveKeyFile(key.Name, content); err != nil {
		return err
	}
	if err := k.AddKey(key); err != nil {
		if err := k.DeleteKeyFile(key.Name); err != nil {
			log.Warnf("delete key %s error: %s", key.Name, err)
		}
		return err
	}
	return nil
}

func (k *KeyManager) SaveKeyFile(name, content string) error {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if err := os.WriteFile(k.keyPath(name), []byte(content), 0600); err != nil {
		return err
	}
	return nil
}

func (k *KeyManager) DeleteKeyFile(name string) error {
	if err := os.Remove(k.keyPath(name)); err != nil {
		return err
	}
	return nil
}

func (k *KeyManager) AddKey(key *structures.Key) error {
	command := fmt.Sprintf("ssh-add %s <<< %s", shellescape.Quote(k.keyPath(key.Name)), shellescape.Quote(key.Password))
	cmd := exec.Command("/bin/bash", "-c", command)

	cmd.Env = append(cmd.Env, "DISPLAY=\":0\"")

	if len(k.config.AddKeyScript) != 0 {
		sshAskPass := fmt.Sprintf("SSH_ASKPASS=%s", shellescape.Quote(k.config.AddKeyScript))
		cmd.Env = append(cmd.Env, sshAskPass)
	}
	if len(k.config.AuthSock) != 0 {
		authSock := fmt.Sprintf("SSH_AUTH_SOCK=%s", shellescape.Quote(k.config.AuthSock))
		cmd.Env = append(cmd.Env, authSock)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("ssh-add error: %s", err)
		return errors.New(fmt.Sprintf("ssh-add error: %s", output))
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (k *KeyManager) keyPath(name string) string {
	return fmt.Sprintf("%s/%s", k.config.Path, name)
}

func directoryExists(dir string) (bool, error) {
	stat, err := os.Stat(dir)
	if err == nil {
		return stat.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
