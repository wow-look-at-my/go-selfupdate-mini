package selfupdate

import (
	"errors"
	"testing"
)

func TestParseSlug(t *testing.T) {
	s := ParseSlug("owner/repo")
	o, r, err := s.GetSlug()
	if err != nil {
		t.Fatal(err)
	}
	if o != "owner" || r != "repo" {
		t.Errorf("expected owner/repo, got %s/%s", o, r)
	}
}

func TestParseSlugEncoded(t *testing.T) {
	s := ParseSlug("owner%2Frepo")
	o, r, err := s.GetSlug()
	if err != nil {
		t.Fatal(err)
	}
	if o != "owner" || r != "repo" {
		t.Errorf("expected owner/repo, got %s/%s", o, r)
	}
}

func TestParseSlugInvalid(t *testing.T) {
	s := ParseSlug("invalid")
	_, _, err := s.GetSlug()
	if !errors.Is(err, ErrInvalidSlug) {
		t.Errorf("expected ErrInvalidSlug, got %v", err)
	}
}

func TestNewRepositorySlug(t *testing.T) {
	s := NewRepositorySlug("test", "repo")
	o, r, err := s.GetSlug()
	if err != nil {
		t.Fatal(err)
	}
	if o != "test" || r != "repo" {
		t.Errorf("expected test/repo, got %s/%s", o, r)
	}
}

func TestRepositorySlugEmptyOwner(t *testing.T) {
	s := NewRepositorySlug("", "repo")
	_, _, err := s.GetSlug()
	if !errors.Is(err, ErrIncorrectParameterOwner) {
		t.Errorf("expected ErrIncorrectParameterOwner, got %v", err)
	}
}

func TestRepositorySlugEmptyRepo(t *testing.T) {
	s := NewRepositorySlug("owner", "")
	_, _, err := s.GetSlug()
	if !errors.Is(err, ErrIncorrectParameterRepo) {
		t.Errorf("expected ErrIncorrectParameterRepo, got %v", err)
	}
}
