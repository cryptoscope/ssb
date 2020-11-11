// SPDX-License-Identifier: MIT

// Package blobstore implements the filesystem storage and simpathy/want managment for ssb-blobs.
package blobstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderr "errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"go.cryptoscope.co/luigi"

	"go.cryptoscope.co/ssb"
	refs "go.mindeco.de/ssb-refs"
)

// ErrNoSuchBlob is returned if the requested blob isn't available
var ErrNoSuchBlob = stderr.New("ssb: no such blob")

// New creates a new BlobStore, storing it's blobs at the given path.
// This store is functionally equivalent to the javascript implementation and thus can share it's path.
// ie: 'ln -s ~/.ssb/blobs ~/.ssb-go/blobs' works to deduplicate the storage.
func New(basePath string) (ssb.BlobStore, error) {
	err := os.MkdirAll(filepath.Join(basePath, "sha256"), 0700)
	if err != nil {
		return nil, errors.Wrap(err, "error making dir for hash sha256")
	}

	err = os.MkdirAll(filepath.Join(basePath, "tmp"), 0700)
	if err != nil {
		return nil, errors.Wrap(err, "error making tmp dir")
	}

	bs := &blobStore{
		basePath: basePath,
	}

	bs.sink, bs.bcast = luigi.NewBroadcast()

	return bs, nil
}

type blobStore struct {
	basePath string

	sink  luigi.Sink
	bcast luigi.Broadcast
}

func (store *blobStore) getPath(ref *refs.BlobRef) (string, error) {
	if err := ref.IsValid(); err != nil {
		return "", errors.Wrap(err, "blobs: invalid reference")
	}

	hexHash := hex.EncodeToString(ref.Hash)
	relPath := filepath.Join(ref.Algo, hexHash[:2], hexHash[2:])

	return filepath.Join(store.basePath, relPath), nil
}

func (store *blobStore) getHexDirPath(ref *refs.BlobRef) (string, error) {
	if err := ref.IsValid(); err != nil {
		return "", errors.Wrap(err, "blobs: invalid reference")
	}

	hexHash := hex.EncodeToString(ref.Hash)
	relPath := filepath.Join(ref.Algo, hexHash[:2])

	return filepath.Join(store.basePath, relPath), nil
}

func (store *blobStore) getTmpPath() string {
	return filepath.Join(store.basePath, "tmp", fmt.Sprint(time.Now().UnixNano()))
}

func (store *blobStore) Get(ref *refs.BlobRef) (io.Reader, error) {
	blobPath, err := store.getPath(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting path for ref %q", ref)
	}

	f, err := os.Open(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSuchBlob
		}
		return nil, errors.Wrap(err, "error opening blob file")
	}

	return f, nil
}

func (store *blobStore) Put(blob io.Reader) (*refs.BlobRef, error) {
	tmpPath := store.getTmpPath()
	f, err := os.Create(tmpPath)
	if err != nil {
		return nil, errors.Wrapf(err, "blobstore.Put: error creating tmp file at %q", tmpPath)
	}

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), blob)
	if err != nil && !luigi.IsEOS(err) {
		return nil, errors.Wrap(err, "blobstore.Put: error copying")
	}

	ref := &refs.BlobRef{
		Hash: h.Sum(nil),
		Algo: "sha256",
	}

	if err := f.Close(); err != nil {
		return nil, errors.Wrap(err, "blobstore.Put: error closing tmp file")
	}

	hexDirPath, err := store.getHexDirPath(ref)
	if err != nil {
		return nil, errors.Wrap(err, "blobstore.Put: error getting hex dir path")
	}

	err = os.MkdirAll(hexDirPath, 0700)
	if err != nil {
		// ignore errors that indicate that the directory already exists
		if !os.IsExist(err) {
			return nil, errors.Wrap(err, "blobstore.Put: error creating hex dir")
		}
	}

	finalPath, err := store.getPath(ref)
	if err != nil {
		return nil, errors.Wrap(err, "blobstore.Put: error getting final path")
	}

	err = os.Rename(tmpPath, finalPath)
	if err != nil {
		if _, ok := err.(*os.LinkError); ok {
			_, err1 := os.Stat(tmpPath)
			_, err2 := os.Stat(hexDirPath)
			log.Printf("final and hex:%d\n%s\n%s", n, err1, err2)
		} else {
			log.Printf("err %v %T", err, err)
		}
		return nil, errors.Wrapf(err, "error moving blob from temp path %q to final path %q", tmpPath, finalPath)
	}

	err = store.sink.Pour(context.TODO(), ssb.BlobStoreNotification{
		Op:  ssb.BlobStoreOpPut,
		Ref: ref,
	})

	return ref, errors.Wrap(err, "blobstore.Put: error in notification handler")
}

func (store *blobStore) Delete(ref *refs.BlobRef) error {
	p, err := store.getPath(ref)
	if err != nil {
		return errors.Wrap(err, "error getting blob path")
	}

	err = os.Remove(p)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNoSuchBlob
		}
		return errors.Wrap(err, "error removing file")
	}

	err = store.sink.Pour(context.TODO(), ssb.BlobStoreNotification{
		Op:  ssb.BlobStoreOpRm,
		Ref: ref,
	})

	return errors.Wrap(err, "error in delete notification handlers")
}

func (store *blobStore) List() luigi.Source {
	return &listSource{
		basePath: filepath.Join(store.basePath, "sha256"),
	}
}

func (store *blobStore) Size(ref *refs.BlobRef) (int64, error) {
	blobPath, err := store.getPath(ref)
	if err != nil {
		return 0, errors.Wrapf(err, "error getting path")
	}

	fi, err := os.Stat(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrNoSuchBlob
		}

		return 0, errors.Wrap(err, "error getting file info")
	}

	return fi.Size(), nil

}

func (store *blobStore) Changes() luigi.Broadcast {
	return store.bcast
}
