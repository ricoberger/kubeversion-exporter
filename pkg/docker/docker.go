package docker

import (
	"errors"
	"strings"

	"github.com/nokia/docker-registry-client/registry"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrCouldNotGetRegAndRepo is the error if the getRegAndRepo return the
	// default values ("" and "" for registry and repository).
	ErrCouldNotGetRegAndRepo = errors.New("could not get registry and repository for the given image")
)

// GetTags ...
func GetTags(image string) ([]string, error) {
	var username, password, reg, repo string

	reg, repo = getRegAndRepo(image)
	if reg == "" && repo == "" {
		return nil, ErrCouldNotGetRegAndRepo
	}

	log.WithFields(log.Fields{"registry": reg, "repository": repo}).Debugf("Get registry and repository for %s", image)

	hub, err := registry.NewCustom(reg, registry.Options{
		Username:         username,
		Password:         password,
		Insecure:         false,
		Logf:             log.Tracef,
		DoInitialPing:    false,
		DisableBasicAuth: true,
	})
	if err != nil {
		return nil, err
	}

	return hub.Tags(repo)
}

func getRegAndRepo(image string) (string, string) {
	parts := strings.Split(image, "/")

	switch len(parts) {
	case 1:
		return "https://registry-1.docker.io/", "library/" + parts[0]
	case 2:
		return "https://registry-1.docker.io/", strings.Join(parts, "/")
	case 3:
		switch parts[0] {
		case "docker.io":
			return "https://registry-1.docker.io/", parts[1] + "/" + parts[2]
		default:
			return "https://" + parts[0], parts[1] + "/" + parts[2]
		}
	default:
		return "", ""
	}
}
