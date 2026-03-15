package selfupdate

import "errors"

var (
	ErrInvalidSlug                 = errors.New("invalid slug format, expected 'owner/name'")
	ErrIncorrectParameterOwner     = errors.New("incorrect parameter \"owner\"")
	ErrIncorrectParameterRepo      = errors.New("incorrect parameter \"repo\"")
	ErrInvalidRelease              = errors.New("invalid release (nil argument)")
	ErrAssetNotFound               = errors.New("asset not found")
	ErrCannotDecompressFile        = errors.New("failed to decompress")
	ErrExecutableNotFoundInArchive = errors.New("executable not found")
)
