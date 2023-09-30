package main

import "gorm.io/gorm"

type CachedScreenshot struct {
	gorm.Model
	URL  string `gorm:"unique"`
	UUID string `gorm:"unique"`
	Data []byte
}
