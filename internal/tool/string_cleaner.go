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

import "strings"

type StringCleaner struct {
	content string
}

func NewStringCleaner(content string) *StringCleaner {
	return &StringCleaner{
		content: content,
	}
}

func (s *StringCleaner) ReplaceAll(from, to string) *StringCleaner {
	s.content = strings.ReplaceAll(s.content, from, to)
	return s
}

func (s *StringCleaner) AsString() string {
	return s.content
}
