package tool

import "time"

type ResponseCacher[requestParams interface{}, responseType interface{}] struct {
	timeout        time.Duration
	cachedResponse *responseType
	cacheableCall  func(requestParams) *responseType
}

func NewResponseCacher[requestParams interface{}, responseType interface{}](timeout time.Duration, cacheableCall func(requestParams) *responseType) *ResponseCacher[requestParams, responseType] {
	return &ResponseCacher[requestParams, responseType]{
		timeout:       timeout,
		cacheableCall: cacheableCall,
	}
}

func (c *ResponseCacher[requestParams, responseType]) Get(params requestParams) *responseType {
	if c.cachedResponse == nil || time.Now().Add(c.timeout).Before(time.Now()) {
		c.cachedResponse = c.cacheableCall(params)
	}
	return c.cachedResponse
}
