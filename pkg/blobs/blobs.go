package blobs

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/lbryio/lbry.go/v3/stream"
	"github.com/lbryio/reflector.go/db"
	"github.com/lbryio/reflector.go/reflector"
	"github.com/lbryio/reflector.go/store"
	pb "github.com/lbryio/types/v2/go"
)

const (
	// MaxChunkSize is the max size of decrypted blob.
	MaxChunkSize = stream.MaxBlobSize - 1

	// DefaultPrefetchLen is how many blobs we should prefetch ahead.
	// 3 should be enough to deliver 2 x 4 = 8MB/s streams.
	// however since we can't keep up, let's see if 2 works
	DefaultPrefetchLen = 2
)

type Source struct {
	filePath      string
	blobsPath     string
	finalPath     string
	stream        *pb.Stream
	blobsManifest []string
}

type Uploader struct {
	uploader *reflector.Uploader
}

func NewSource(filePath, blobsPath string) (*Source, error) {
	s := Source{
		filePath:  filePath,
		blobsPath: blobsPath,
	}

	return &s, nil
}

func NewUploaderFromCfg(cfg map[string]string) (*Uploader, error) {
	db := &db.SQL{
		LogQueries: false,
	}
	err := db.Connect(cfg["databasedsn"])
	if err != nil {
		return nil, err
	}

	store := store.NewDBBackedStore(store.NewS3Store(
		cfg["key"], cfg["secret"], cfg["region"], cfg["bucket"],
	), db, false)
	return &Uploader{
		uploader: reflector.NewUploader(db, store, 5, false, false),
	}, nil
}

func (s *Source) Split() (*pb.Stream, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open source: %w", err)
	}
	defer file.Close()

	enc := stream.NewEncoder(file)

	encodedStream, err := enc.Stream()
	if err != nil {
		return nil, fmt.Errorf("cannot create stream: %w", err)
	}
	s.stream = &pb.Stream{
		Source: &pb.Source{
			SdHash: enc.SDBlob().Hash(),
			Name:   filepath.Base(file.Name()),
			Size:   uint64(enc.SourceLen()),
			Hash:   enc.SourceHash(),
		},
	}

	s.finalPath = path.Join(s.blobsPath, enc.SDBlob().HashHex())
	err = os.MkdirAll(s.finalPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("cannot create directory for blobs: %w", err)
	}

	s.blobsManifest = make([]string, len(encodedStream))

	for i, b := range encodedStream {
		err := ioutil.WriteFile(path.Join(s.blobsPath, enc.SDBlob().HashHex(), b.HashHex()), b, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("cannot write blob: %w", err)
		}
		s.blobsManifest[i] = b.HashHex()
	}

	return s.stream, nil
}

func (s *Source) Stream() *pb.Stream {
	return s.stream
}

func (u *Uploader) Upload(source *Source) (*reflector.Summary, error) {
	if source.finalPath == "" || source.Stream() == nil {
		return nil, errors.New("source is not split to blobs")
	}
	err := u.uploader.Upload(source.finalPath)
	summary := u.uploader.GetSummary()
	if err != nil {
		return nil, fmt.Errorf("cannot upload blobs: %w", err)
	}
	return &summary, nil
}