package common

import (
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	*cache.Cache
}

func NewCache(expirationTime, cleanupTime time.Duration) *Cache {
	return &Cache{cache.New(expirationTime, cleanupTime)}
}

func (c *Cache) Set(key string, value interface{}) {
	c.Cache.Set(key, value, cache.DefaultExpiration)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	return c.Cache.Get(key)
}

func (c *Cache) Flush() {
	c.Cache.Flush()
}

func CacheKeyBlog(id int) string {
	return "blog:" + strconv.Itoa(id)
}

func CacheKeyBlogsByUserId(id int) string {
	return "blogs_by_user:" + strconv.Itoa(id)
}

func CacheKeyBlogs(limit, offset int) string {
	return "blogs:" + strconv.Itoa(limit) + ":" + strconv.Itoa(offset)
}
