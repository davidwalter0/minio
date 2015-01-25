package storage

import (
	"bytes"
	"math/rand"
	"strconv"

	. "gopkg.in/check.v1"
)

func APITestSuite(c *C, create func() Storage) {
	testCreateBucket(c, create)
	testMultipleObjectCreation(c, create)
	testPaging(c, create)
	testObjectOverwriteFails(c, create)
	testNonExistantBucketOperations(c, create)
	testBucketRecreateFails(c, create)
}

func testCreateBucket(c *C, create func() Storage) {
	// test create bucket
	// test bucket exists
	// test no objects exist
	// 2x
}

func testMultipleObjectCreation(c *C, create func() Storage) {
	objects := make(map[string][]byte)
	storage := create()
	storage.StoreBucket("bucket")
	for i := 0; i < 10; i++ {
		randomPerm := rand.Perm(10)
		randomString := ""
		for _, num := range randomPerm {
			randomString = randomString + strconv.Itoa(num)
		}
		key := "obj" + strconv.Itoa(i)
		objects[key] = []byte(randomString)
		err := storage.StoreObject("bucket", key, bytes.NewBufferString(randomString))
		c.Assert(err, IsNil)
	}

	// ensure no duplicates
	etags := make(map[string]string)
	for key, value := range objects {
		var byteBuffer bytes.Buffer
		storage.CopyObjectToWriter(&byteBuffer, "bucket", key)
		c.Assert(bytes.Equal(value, byteBuffer.Bytes()), Equals, true)

		metadata, err := storage.GetObjectMetadata("bucket", key)
		c.Assert(err, IsNil)
		c.Assert(metadata.Size, Equals, len(value))

		_, ok := etags[metadata.ETag]
		c.Assert(ok, Equals, false)
		etags[metadata.ETag] = metadata.ETag
	}
}

func testPaging(c *C, create func() Storage) {
	storage := create()
	storage.StoreBucket("bucket")
	storage.ListObjects("bucket", "", 1000)
	c.Assert(len(storage.ListObjects("bucket", "", 1000)), Equals, 0)
	for i := 1; i <= 1000; i++ {
		key := "obj" + strconv.Itoa(i)
		storage.StoreObject("bucket", key, bytes.NewBufferString(key))
		c.Assert(len(storage.ListObjects("bucket", "", i)), Equals, i)
	}
	for i := 1001; i <= 2000; i++ {
		key := "obj" + strconv.Itoa(i)
		storage.StoreObject("bucket", key, bytes.NewBufferString(key))
		c.Assert(len(storage.ListObjects("bucket", "", 1000)), Equals, 1000)
	}
}

func testObjectOverwriteFails(c *C, create func() Storage) {
	storage := create()
	storage.StoreBucket("bucket")
	err := storage.StoreObject("bucket", "object", bytes.NewBufferString("one"))
	c.Assert(err, IsNil)
	err = storage.StoreObject("bucket", "object", bytes.NewBufferString("one"))
	c.Assert(err, Not(IsNil))
}

func testNonExistantBucketOperations(c *C, create func() Storage) {
	storage := create()
	err := storage.StoreObject("bucket", "object", bytes.NewBufferString("one"))
	c.Assert(err, Not(IsNil))
}

func testBucketRecreateFails(c *C, create func() Storage) {
	storage := create()
	err := storage.StoreBucket("string")
	c.Assert(err, IsNil)
	err = storage.StoreBucket("string")
	c.Assert(err, Not(IsNil))
}
