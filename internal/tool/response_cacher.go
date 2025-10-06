/*
git-slack-bot
Copyright (C) 2025 loveholidays

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU Lesser General Public
License as published by the Free Software Foundation; either
version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with this program; if not, write to the Free Software Foundation,
Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
*/

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
