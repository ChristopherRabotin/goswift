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
	Enabled  bool   // Whether the index is enabled or not.
	Location string // Location of the index.
	Header   string // First line of the index, used if the file is new.
	Body     string // Body which is appended to the index (for new and old data)
}

// S3Persister defines how and what is persisted to S3.
type S3Persister interface {
	SetContext(*gin.Context) // sets the context for this persister instance.
	IndexInfo() *S3Index     // Returns if this persister is indexed, the location of that index, and the data that goes in the index.
	Location() string        // location on S3 for this persisted data.
	Serialize() string       // serialized data to store on S3.
	Checksum() string        // checksum of the context, used only if this is an index persister.
}

// S3Persist implements of an S3Persister.
type S3Persist struct {
	path        string
	Indexed     bool
	Context     *gin.Context
	CBody       string
	contentPath string
}

func (p S3Persist) SetBody() {
	if p.Context.Request.Body != nil {
		body, err := ioutil.ReadAll(p.Context.Request.Body)
		if err != nil {
			panic("could not read body")
		}
		p.CBody = string(body)
		fmt.Printf("-->%+v", p)
	}else{
		fmt.Println("body is nil!")
	}
}

func (p S3Persist) IndexInfo() *S3Index {
	// Let's determine where the index should be.
	loc := rootPath + storePath
	if testGoswift {
		loc += "/test"
	}
	loc += indexFolder + "/" + p.path

	successIft, _ := p.Context.Get("authSuccess")
	if success, ok := successIft.(bool); ok {
		if success {
			loc += "valid/"
		} else {
			loc += "invalid/"
		}
	} else {
		loc += "unknown/"
	}
	accessKeyItf, _ := p.Context.Get("token")
	accessKey := "noAccessKeyFound"
	if accessKey, ok := accessKeyItf.(string); ok {
		accessKey = accessKey
	}
	loc += accessKey + "_" + p.Checksum()

	return &S3Index{Enabled: p.Indexed, Location: loc, Header: fmt.Sprintf("%s\n", p.Location()),
		Body: fmt.Sprintf("%s\t%s\n", accessKey, time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))}
}

// serialize returns what is to be serialized and saved to S3.
func (p S3Persist) Serialize() string {
	return p.CBody
}

// checksum returns the checksum of this analytics event, which is not used for analytics.
func (p S3Persist) Checksum() string {
	hash := sha512.New384()
	hash.Write([]byte(p.CBody))
	return hex.EncodeToString(hash.Sum(nil))
}

// location returns the location where this context will be persisted.
func (p S3Persist) Location() string {
	if p.contentPath == "" {
		// Let's only compute this once.
		loc := rootPath + storePath
		if testGoswift {
			loc += "/test"
		}
		loc += dataFolder + "/" + p.path + "/"
		now := time.Now().UTC()
		y, m, d := now.Date()
		successIft, _ := p.Context.Get("authSuccess")
		if success, ok := successIft.(bool); ok {
			if success {
				loc += "valid/"
			} else {
				loc += "invalid/"
			}
		} else {
			loc += "unknown/"
		}
		loc += fmt.Sprintf("%04d_%02d_%02d/%02d00/", y, m, d, now.Hour())
		accessKeyItf, _ := p.Context.Get("token")
		if accessKey, ok := accessKeyItf.(string); ok {
			loc += accessKey
		} else {
			loc += "noAccessKeyFound"
		}
		randStr, _ := randutil.AlphaStringRange(8, 16)

		p.contentPath = loc + "_" + p.Checksum() + "_" + randStr
	}
	return p.contentPath
}

func NewS3Persist(path string, indexed bool, c *gin.Context) S3Persist {
	p := S3Persist{path: path, Indexed: indexed, Context: c}
	p.SetBody()
	fmt.Printf("||%+v\n", p)
	return p
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
func S3PersistingHandler(persistChan chan S3Persist, wg *sync.WaitGroup) {
	bucket := S3BucketFromOS()
	for {
		persist, open := <-persistChan
		if !open {
			log.Info("Persist channel is closed. Server probably shutting down.")
			return
		}
		fmt.Printf("==>%+v\n", persist)
		// If this is an indexed persistence, let's check uniqueness.
		if iInfo := persist.IndexInfo(); iInfo.Enabled {
			indexData, notFoundErr := bucket.Get(iInfo.Location)
			if notFoundErr == nil {
				// Append index content to the existing index.
				s3Err := bucket.Put(iInfo.Location, []byte(string(indexData)+iInfo.Body), "text/plain", s3.Private)
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
				s3Err := bucket.Put(persist.Location(), []byte(persist.Serialize()), "text/plain", s3.Private)
				if s3Err != nil {
					// If somethting goes wrong, let's re-add this fetch to items to be processed.
					persistChan <- persist
					log.Error("could not PUT new content: %s", s3Err)
					continue
				}

				// Add canonical index information.
				for i := 0; i < 10; i++ {
					s3Err := bucket.Put(iInfo.Location, []byte(iInfo.Header+iInfo.Body), "text/plain", s3.Private)
					if s3Err == nil {
						break
					} else if i == 9 {
						// Panic: we have attempted to add the index information ten times.
						panic(fmt.Sprintf("Could not add index: %+v", iInfo))
					}
				}

			}
		} else {
			// This is not indexed, so let's persist it to S3 directly.
			s3Err := bucket.Put(persist.Location(), []byte(persist.Serialize()), "text/plain", s3.Private)
			if s3Err != nil {
				// If somethting goes wrong, let's re-add this fetch to items to be processed.
				persistChan <- persist
				log.Error("could not PUT new content on %s: %s", bucket.Name, s3Err)
				continue
			}
			fmt.Printf("--->PUT to %s [%+v]\n", persist.Location(), persist.Serialize())
		}

		wg.Done()
	}
}

// indexOnS3 works in a similar fashion to storeOnS3. However, it will only store the new context to S3, and index it if it was not new.
// This index is based on the SHA384 of the data. Anything which was not found on S3 index will be added to the newChan for further processing.
// For example, the newChan will then forward the request to the Content API. This will massively reduce the load on the ContentAPI, and only
// attempt to add information which we know has not been encountered before.
func indexOnS3(persistChan <-chan *S3Persister, newChan chan<- *gin.Context, wg *sync.WaitGroup) {

}
