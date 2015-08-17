package goswift

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmcvetta/randutil"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const (
	rootPath    = "/goswift"
	storePath   = "/incoming"
	indexFolder = "/index/sha384_checksum"
	dataFolder  = "/sha384_data"
)

type S3Index struct {
	Location string // Location of the index.
	Header   string // First line of the index, used if the file is new.
	Body     string // Body which is appended to the index (for new and old data)
}

// S3Persist stores information to be persist on S3.
type S3Persist struct {
	s3path      string
	cBody       string
	ContentPath string
	Checksum    string
	Serialized  string
	Index       *S3Index
}

func NewS3Persist(s3path string, indexed bool, c *gin.Context) *S3Persist {
	p := S3Persist{s3path: s3path}

	// Extracting some values from the context.
	authSuccessIft, _ := c.Get("authSuccess")
	successFolder := "unknown"
	if val, ok := authSuccessIft.(bool); ok {
		if val {
			successFolder = "valid"
		} else {
			successFolder = "invalid"
		}
	}

	accessKeyItf, _ := c.Get("token")
	accessKey := "noAccessKeyFound"
	if val, ok := accessKeyItf.(string); ok {
		accessKey = val
	}

	// Let's set the body.
	if c.Request.Body != nil {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			panic("could not read body")
		}
		p.cBody = string(body)
	}

	// Serializing the data to persist.
	p.Serialized = p.cBody

	// Let's compute the checksum of this persistence.
	hash := sha512.New384()
	hash.Write([]byte(p.cBody))
	p.Checksum = hex.EncodeToString(hash.Sum(nil))

	// Let's set the persistence location.
	cLoc := rootPath + storePath
	if testGoswift {
		cLoc += "/test"
	} else {
		cLoc += "/live"
	}
	now := time.Now().UTC()
	y, m, d := now.Date()

	cLoc += fmt.Sprintf("%s/%s/%s/%04d_%02d_%02d/%02d00/", dataFolder, p.s3path, successFolder, y, m, d, now.Hour())

	p.ContentPath = cLoc + accessKey

	if indexed {
		// If this is an indexed item, then we store each item in a different file. Otherwise, we just append the file.
		randStr, _ := randutil.AlphaStringRange(8, 16)
		p.ContentPath += "_" + p.Checksum + "_" + randStr
		// Let's determine where the index should be.
		iLoc := rootPath + storePath
		if testGoswift {
			iLoc += "/test"
		} else {
			iLoc += "/live"
		}
		iLoc += indexFolder + "/" + p.s3path + "/" + successFolder + "/" + accessKey + "_" + p.Checksum

		p.Index = &S3Index{Location: iLoc, Header: fmt.Sprintf("%s\n", p.ContentPath),
			Body: fmt.Sprintf("%s\t%s\t%s\n", accessKey, time.Now().UTC().Format("2006-01-02T15:04:05.000Z"), c.ClientIP())}

		if testGoswift {
			testS3Locations = append(testS3Locations, p.Index.Location)
		}
	}

	if testGoswift {
		testS3Locations = append(testS3Locations, p.ContentPath)
	}

	return &p
}

// S3BucketFromOS returns the bucket from the environment variables (cf. README.md).
func S3BucketFromOS() *s3.Bucket {
	// Prepare AWS S3 connection.
	s3auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}
	client := s3.New(s3auth, aws.USEast)
	return client.Bucket(os.Getenv("AWS_STORAGE_BUCKET_NAME"))
}

// S3PersistingHandler stores information from the contextChan onto S3.
func S3PersistingHandler(persistChan chan *S3Persist, wg *sync.WaitGroup) {
	bucket := S3BucketFromOS()
	for {
		persist, open := <-persistChan
		if !open {
			log.Info("Persist channel is closed. Server probably shutting down.")
			return
		}
		// If this is an indexed persistence, let's check uniqueness.
		if persist.Index != nil {
			indexData, notFoundErr := bucket.Get(persist.Index.Location) // notFoundErr => if there is an err Get failed, so the file does not exist.
			if notFoundErr == nil {
				// Append index content to the existing index.
				s3Err := bucket.Put(persist.Index.Location, []byte(string(indexData)+persist.Index.Body), "text/plain", s3.Private)
				if s3Err != nil {
					// If somethting goes wrong, let's re-add this fetch to items to be processed.
					persistChan <- persist
					log.Error("could not update index: %s", s3Err)
					continue
				}
				// We should stop processing this request now.
				// TODO: Determine how to do that, possible by setting a context value?

			} else {
				// Store the content on S3 and create an index.
				s3Err := bucket.Put(persist.ContentPath, []byte(persist.Serialized), "text/plain", s3.Private)
				if s3Err != nil {
					// If somethting goes wrong, let's re-add this fetch to items to be processed.
					persistChan <- persist
					log.Error("could not PUT new content: %s", s3Err)
					continue
				}

				// Add canonical index information.
				for i := 0; i < 10; i++ {
					s3Err := bucket.Put(persist.Index.Location, []byte(persist.Index.Header+persist.Index.Body), "text/plain", s3.Private)
					if s3Err == nil {
						break
					} else if i == 9 {
						// Panic: we have attempted to add the index information ten times.
						panic(fmt.Sprintf("Could not add index: %+v", persist.Index))
					}
				}
			}
		} else {
			// This is not indexed, so let's persist it to S3 by adding it to the already present file, or appending to it.
			oldData, notFoundErr := bucket.Get(persist.Index.Location)
			newData := ""
			if notFoundErr == nil {
				newData += string(oldData) + "\n"
				// Append index content to the existing index.
			}
			s3Err := bucket.Put(persist.ContentPath, []byte(newData), "text/plain", s3.Private)
			if s3Err != nil {
				// If somethting goes wrong, let's re-add this persistor to items to be persisted.
				persistChan <- persist
				log.Error("could not PUT new content on %s: %s", bucket.Name, s3Err)
				continue
			}
		}

		wg.Done()
	}
}
