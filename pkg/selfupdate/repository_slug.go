package selfupdate

import "strings"

// RepositorySlug identifies a GitHub repository by owner/repo.
type RepositorySlug struct {
	owner string
	repo  string
}

var _ Repository = RepositorySlug{}

// ParseSlug parses an "owner/repo" string into a RepositorySlug.
func ParseSlug(slug string) RepositorySlug {
	var owner, repo string
	parts := strings.Split(slug, "/")
	if len(parts) != 2 {
		parts = strings.Split(slug, "%2F")
	}
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
	}
	return RepositorySlug{owner: owner, repo: repo}
}

// NewRepositorySlug creates a RepositorySlug from owner and repo.
func NewRepositorySlug(owner, repo string) RepositorySlug {
	return RepositorySlug{owner: owner, repo: repo}
}

func (r RepositorySlug) GetSlug() (string, string, error) {
	if r.owner == "" && r.repo == "" {
		return "", "", ErrInvalidSlug
	}
	if r.owner == "" {
		return r.owner, r.repo, ErrIncorrectParameterOwner
	}
	if r.repo == "" {
		return r.owner, r.repo, ErrIncorrectParameterRepo
	}
	return r.owner, r.repo, nil
}
