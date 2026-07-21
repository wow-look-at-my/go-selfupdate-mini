package selfupdate

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseSlug(t *testing.T) {
	s := ParseSlug("owner/repo")
	o, r, err := s.GetSlug()
	require.Nil(t, err)

	assert.False(t, o != "owner" || r != "repo")

}

func TestParseSlugEncoded(t *testing.T) {
	s := ParseSlug("owner%2Frepo")
	o, r, err := s.GetSlug()
	require.Nil(t, err)

	assert.False(t, o != "owner" || r != "repo")

}

func TestParseSlugInvalid(t *testing.T) {
	s := ParseSlug("invalid")
	_, _, err := s.GetSlug()
	assert.True(t, errors.Is(err, ErrInvalidSlug))

}

func TestNewRepositorySlug(t *testing.T) {
	s := NewRepositorySlug("test", "repo")
	o, r, err := s.GetSlug()
	require.Nil(t, err)

	assert.False(t, o != "test" || r != "repo")

}

func TestRepositorySlugEmptyOwner(t *testing.T) {
	s := NewRepositorySlug("", "repo")
	_, _, err := s.GetSlug()
	assert.True(t, errors.Is(err, ErrIncorrectParameterOwner))

}

func TestRepositorySlugEmptyRepo(t *testing.T) {
	s := NewRepositorySlug("owner", "")
	_, _, err := s.GetSlug()
	assert.True(t, errors.Is(err, ErrIncorrectParameterRepo))

}
