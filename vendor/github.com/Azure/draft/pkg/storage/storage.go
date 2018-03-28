package storage

// To regenerate the protocol buffer types for this package, run:
//		go generate

//go:generate make proto

import (
	"context"
	b64 "encoding/base64"

	"github.com/golang/protobuf/proto"
)

// Deleter represents the delete APIs of the storage engine.
type Deleter interface {
	// DeleteBuilds deletes all draft builds for the application specified by appName.
	DeleteBuilds(ctx context.Context, appName string) ([]*Object, error)
	// DeleteBuild deletes the draft build given by buildID for the application specified by appName.
	DeleteBuild(ctx context.Context, appName, buildID string) (*Object, error)
}

// Creator represents the create APIs of the storage engine.
type Creator interface {
	// CreateBuild creates and stores a new build.
	CreateBuild(ctx context.Context, appName string, build *Object) error
}

// Updater represents the update APIs of the storage engine.
type Updater interface {
	// UpdateBuild creates and stores a new build.
	UpdateBuild(ctx context.Context, appName string, build *Object) error
}

// Getter represents the retrieval APIs of the storage engine.
type Getter interface {
	// GetBuilds retrieves all draft builds from storage.
	GetBuilds(ctx context.Context, appName string) ([]*Object, error)
	// GetBuild retrieves the draft build by id from storage.
	GetBuild(ctx context.Context, appName, buildID string) (*Object, error)
}

// Store represents a storage engine for application state stored by Draft.
type Store interface {
	Creator
	Deleter
	Updater
	Getter
}

// EncodeToString returns the base64 encoding of a protobuf encoded storage.Object.
//
// err != nil if the protobuf marshaling fails; otherwise nil.
func EncodeToString(obj *Object) (string, error) {
	b, err := proto.Marshal(obj)
	if err != nil {
		return "", err
	}
	return b64.StdEncoding.EncodeToString(b), nil
}

// DecodeString returns the storage.Object decoded from a base64 encoded protobuf string.
//
// err != nil if decoding fails.
func DecodeString(str string) (*Object, error) {
	b, err := b64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	var obj Object
	if err := proto.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
