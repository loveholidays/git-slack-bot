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
