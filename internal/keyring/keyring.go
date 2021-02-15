package keyring

import (
	"os"

	"github.com/swampapp/swamp/internal/config"
	"github.com/zalando/go-keyring"
)

type Keyring struct {
	Repository string
	Password   string
	Var1       string
	Var2       string
	ID         string
}

func New(repoID string) *Keyring {
	if len(repoID) != 64 {
		panic("invalid repo ID")
	}

	instance := &Keyring{ID: repoID}
	uri, err := keyring.Get(instance.key(), "repository")
	if err == nil {
		instance.Repository = uri
	}
	pass, _ := keyring.Get(instance.key(), "password")
	if err == nil {
		instance.Password = pass
	}
	var1, _ := keyring.Get(instance.key(), "var1")
	if err == nil {
		instance.Var1 = var1
	}
	var2, _ := keyring.Get(instance.key(), "var2")
	if err == nil {
		instance.Var2 = var2
	}
	return instance
}

func (s *Keyring) key() string {
	return "com.github.swampapp." + s.ID
}
func FirstBoot() bool {
	return config.PreferredRepo() == ""
}

func (s *Keyring) Delete() error {
	err := keyring.Delete(s.key(), "repository")
	if err != nil {
		return err
	}

	err = keyring.Delete(s.key(), "password")
	if err != nil {
		return err
	}

	err = keyring.Delete(s.key(), "var1")
	if err != nil {
		return err
	}

	err = keyring.Delete(s.key(), "var2")
	if err != nil {
		return err
	}

	return nil
}

func (s *Keyring) Save() error {
	err := keyring.Set(s.key(), "repository", s.Repository)
	if err != nil {
		return err
	}

	err = keyring.Set(s.key(), "password", s.Password)
	if err != nil {
		return err
	}

	err = keyring.Set(s.key(), "var1", s.Var1)
	if err != nil {
		return err
	}

	err = keyring.Set(s.key(), "var2", s.Var2)
	if err != nil {
		return err
	}

	return nil
}

func (s *Keyring) ExportEnv() {
	os.Setenv("RESTIC_REPOSITORY", s.Repository)
	os.Setenv("RESTIC_PASSWORD", s.Password)
	os.Setenv("AWS_ACCESS_KEY", s.Var1)
	os.Setenv("AWS_SECRET_ACCESS_KEY", s.Var2)
}

func ResetEnv() {
	os.Unsetenv("RESTIC_REPOSITORY")
	os.Unsetenv("RESTIC_PASSWORD")
	os.Unsetenv("AWS_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
}
