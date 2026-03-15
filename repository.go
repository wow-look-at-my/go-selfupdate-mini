package selfupdate

// Repository identifies a repository to check for updates.
type Repository interface {
	GetSlug() (owner string, repo string, err error)
}
