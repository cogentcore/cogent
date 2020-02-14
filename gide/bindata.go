// Code generated by go-bindata. DO NOT EDIT.
// sources:
// ../icons/gide.svg

package gide


import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}


type asset struct {
	bytes []byte
	info  fileInfoEx
}

type fileInfoEx interface {
	os.FileInfo
	MD5Checksum() string
}

type bindataFileInfo struct {
	name        string
	size        int64
	mode        os.FileMode
	modTime     time.Time
	md5checksum string
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) MD5Checksum() string {
	return fi.md5checksum
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _bindataIconsGidesvg = []byte(
	"\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x59\xeb\xaf\xa3\xb8\x15\xff\x7e\xff\x0a\xca\xaa\xda\x19\x35\x18\xdb" +
	"\x18\x1b\xb8\x21\xab\x6e\x57\xbb\x5a\xa9\x52\xa5\xee\x8e\xfa\xb1\x72\xc0\x21\xee\x75\x30\x32\xce\x6b\xfe\xfa\xca" +
	"\x10\x1e\xc9\xcd\xbc\x3a\x23\x75\xda\xb9\xf9\x12\x38\x0f\x1b\x9f\xdf\x39\x3f\x1f\xc3\xf2\x87\xd3\x4e\x79\x07\x61" +
	"\x5a\xa9\xeb\xdc\x47\x00\xfa\x9e\xa8\x0b\x5d\xca\xba\xca\xfd\x37\xbf\xff\x1c\x24\xbe\xd7\x5a\x5e\x97\x5c\xe9\x5a" +
	"\xe4\x7e\xad\xfd\x1f\x56\x0f\xcb\x3f\x04\x81\xf7\x17\x23\xb8\x15\xa5\x77\x94\x76\xeb\xfd\x5a\x3f\xb5\x05\x6f\x84" +
	"\xf7\x6a\x6b\x6d\x93\x85\xe1\xf1\x78\x04\xf2\x22\x04\xda\x54\xe1\x6b\x2f\x08\x56\x0f\x0f\xcb\xf6\x50\x3d\x78\x9e" +
	"\x77\xda\xa9\xba\xcd\xca\x22\xf7\x2f\x0e\xcd\xde\xa8\xce\xb0\x2c\x42\xa1\xc4\x4e\xd4\xb6\x0d\x11\x40\xa1\x3f\x99" +
	"\x17\x93\x79\xe1\x66\x97\x07\x51\xe8\xdd\x4e\xd7\x6d\xe7\x59\xb7\xdf\xcd\x8c\x4d\xb9\x19\xad\xdd\xd3\x1c\xa3\xce" +
	"\x08\xa5\x69\x1a\x42\x1c\x62\x1c\x98\x72\x13\xb4\xe7\xda\xf2\x53\x70\xed\xda\x1e\xaa\x7b\xae\x18\x42\x18\xb6\x87" +
	"\x6a\xb2\xfc\x38\xab\xec\xa4\x64\xfd\xf4\xce\x87\xe9\xb4\xf3\xd9\x75\x29\x1b\x5d\xca\xd1\x61\x10\x80\x56\xef\x4d" +
	"\x21\x36\xda\x54\x02\xd4\xc2\x86\x3f\xfd\xfe\xd3\xa8\x0c\x20\x28\x6d\x39\x1b\x66\x88\xfe\xd5\xbc\x57\x90\xd4\x7c" +
	"\x27\xda\x86\x17\xa2\x0d\x07\x79\xe7\x7f\x94\xa5\xdd\xe6\x3e\x4a\x01\xc2\x0c\x43\xb6\xdb\x75\xe2\xad\x90\xd5\xd6" +
	"\x76\x72\x48\x60\xca\xd2\x8b\xfc\x20\xc5\xf1\x47\x7d\xca\x7d\xe8\x41\x8f\x32\xc0\x58\x14\x53\xe6\xae\x08\x4d\x60" +
	"\xc4\x3a\x23\x59\xe6\x7e\x7b\xa8\x70\xef\x31\x25\x1c\xea\xb5\x97\xe9\xb3\x51\x03\x41\x8a\x3c\x83\x22\x86\xe3\xce" +
	"\x62\x58\x66\x56\xea\xc2\x3d\x77\xee\x57\xb2\x14\x60\x88\xf3\x38\x80\x38\x35\xda\xd8\x60\x23\x95\xe8\xcd\xc2\x37" +
	"\xad\x30\x6d\xa8\x8d\x90\x4a\x9d\xc3\x4a\x3f\xc9\xd0\xb9\x86\x4a\x57\xba\xbb\xfa\xa7\x2c\x74\x0d\x9a\xfa\xfe\x48" +
	"\xa7\xb2\x91\xb9\x1f\x41\x08\xe2\x98\xc5\xe9\x5d\x9b\xf3\x8d\xcd\xea\xc1\xf3\x96\xa5\xd8\xb4\xce\xb8\x5f\xbb\xbb" +
	"\x23\x9d\xc2\xf3\x96\x4a\xd6\x82\x9b\x5f\x0c\x2f\xa5\xa8\x6d\x6f\x34\x1b\xb5\xd0\x4a\x89\xc2\xe6\x3e\x57\x47\x7e" +
	"\x6e\xfd\xd1\xa0\xcc\xfd\x6b\x57\x92\x62\x7c\x19\xd4\xf3\x96\xad\xd5\xcd\x60\xeb\x79\xad\x3d\x2b\x91\xfb\x4e\x18" +
	"\x14\x5a\x69\x93\x7d\xb7\x29\x4b\x08\xe1\x63\x27\xd2\x0d\x2f\xa4\x3d\x67\xc8\x9f\x5c\xf4\x66\xd3\x0a\x9b\xfb\x70" +
	"\x26\xeb\x80\xb3\xba\x21\x29\x26\xbe\x17\x7e\xc2\x64\xdd\xef\xc3\x93\xa1\xfb\x93\xd1\x71\xb2\x65\x78\xbd\xe8\xcf" +
	"\x0b\x62\x57\x6d\xd9\xd6\x88\x4d\xee\x7f\x77\x27\x9a\xef\x0d\x76\x32\x0d\x83\x72\x1f\x27\x14\xd0\x94\x12\x36\x4a" +
	"\xcf\x28\xf7\xe3\x34\x06\x88\xa1\x28\x9a\x6c\x71\xee\x47\x31\x01\x84\x41\x38\x49\xcf\xf8\x9e\x6d\x75\x99\xec\x4d" +
	"\x2d\x6d\x9b\xfb\xfb\x56\x98\xdf\x5c\x89\xfe\xad\x7e\xd3\x8a\x4b\x48\x96\xa1\xcb\xa6\xee\x6a\x2c\x0b\x97\xec\xa5" +
	"\xab\xc4\x29\xe5\xd6\xbc\x15\x97\x71\x1b\x5e\x89\x0e\x96\xdc\xbf\xe0\x72\x51\xac\xb5\x29\x85\x19\x54\xb4\xfb\x5d" +
	"\xa9\x2e\xc8\xf5\x7b\xc3\xc3\x75\x88\xdd\xa8\xa3\x1e\xde\xd7\xb7\x5b\x5e\xea\x63\xee\xe3\x5b\xe5\x5b\xad\x77\xb9" +
	"\x8f\x10\x88\x50\xc4\x60\x7a\xab\x2e\x4e\xb9\x4f\x22\x40\x62\x86\xd1\x33\xdf\xe2\x9c\xfb\x84\x02\x46\x22\xf6\xcc" +
	"\xb1\xd4\xc5\xde\x6d\x1e\xc1\xbe\x8f\x60\x73\x7a\xe6\xbd\x37\xc6\x19\x28\x7e\x16\x26\xf7\xbb\xbf\x21\x09\xdb\xad" +
	"\x3e\x56\xc6\x45\x6f\xc3\xd5\x18\xbe\x8d\xb4\xc1\x8e\x9b\x4a\xd6\x81\xd5\xcd\x54\x1f\x33\xb9\x12\x1b\x7b\x57\x61" +
	"\x7a\xd2\xbc\xa3\x59\x6b\x6b\x5d\x0c\xe0\x00\xeb\x4e\x58\x5e\x72\xcb\x27\x08\x07\x09\x1b\x98\xc3\x94\x9b\xec\xef" +
	"\x3f\xfd\x3c\x56\x61\x51\x64\xff\xd0\xe6\x69\xaa\x20\x67\xc0\xd7\x7a\x6f\x73\x7f\x24\x06\x47\x46\x45\xb6\xd1\x66" +
	"\xc7\xed\x4a\xee\x78\x25\xdc\xfe\xf4\xa7\xd3\x4e\x2d\xc3\x49\x71\x65\x6c\xcf\x8d\x98\x06\xed\x87\x35\xa2\xdf\x7f" +
	"\xee\x6e\xd9\x65\xb1\x93\xce\x29\xfc\xcd\x4a\xa5\x7e\x75\x93\xcc\xd8\xe2\x32\xa8\xb4\x4a\xac\xba\x39\xfb\xcb\x61" +
	"\x15\xe1\x65\x19\x43\xbd\xcf\x56\xb9\x0c\x87\x18\x74\x77\xd5\x0d\x96\x8a\xaf\x85\xca\xfd\xbf\x3a\x0c\x3d\x74\x8b" +
	"\x74\x65\xf4\xbe\xd9\xe9\x52\x5c\x50\xf6\xa7\xc8\x5e\xa1\x6e\x0d\xaf\x5b\x17\x86\xdc\xef\x2e\x15\xb7\xe2\x55\x30" +
	"\x94\x37\x5d\x04\x31\xc5\x00\x41\x1c\xa5\xaf\x47\x20\x44\x31\x72\xce\x85\xfc\x46\x92\x7b\xdc\x48\xa5\x46\x02\x74" +
	"\x37\x23\x01\xc2\xfe\xd6\xec\x95\xc8\x6a\x5d\xbf\x15\x46\x3f\xb6\xd6\xe8\x27\x31\x23\x4c\x77\x1b\x74\x9b\x70\x06" +
	"\x41\x1a\x51\x8c\x61\x92\x0c\x72\xc7\x49\x05\x6f\xb2\xf5\xde\xda\xb9\xec\x5f\x5a\xd6\xd9\x4e\x5a\x61\x06\x69\x77" +
	"\xa3\xe4\x4e\xda\x8c\x0c\xb2\x92\xb7\x5b\x6e\x0c\x3f\xbb\xd9\xc5\x5c\xda\xd3\x71\x06\x07\xd9\xf8\xc4\x57\x8c\xe8" +
	"\x96\x4d\x92\x74\xe2\xec\x4b\xaf\x40\x29\x88\x23\x94\xa0\x78\x54\x0c\xdd\xc2\x73\xcd\xc9\x31\x27\x03\x11\x4a\xe8" +
	"\x8c\x0d\x73\xdf\x05\x39\x66\x30\x9e\x88\xbf\xe1\x76\xfb\xfe\x20\xef\x8d\x7a\xf5\x9c\xc5\x93\xd7\xd7\x51\x47\xef" +
	"\x89\x3a\x84\x31\xa1\xeb\xaf\x2d\xea\x53\x80\x1d\x0b\x78\x11\x8c\x00\x65\x90\xb0\x45\x4c\x29\x48\x93\x34\xc5\x5e" +
	"\x0c\x01\x62\x69\x9a\x2e\x02\x08\x08\x41\x29\xf1\x02\x44\x01\x4c\x28\x4d\x16\x31\x03\x88\x62\x8a\xbd\x20\x76\xca" +
	"\x18\xc5\x0b\x08\x60\x92\x10\xef\xed\x3d\x38\x27\x18\x66\xdb\x67\x5d\x8b\xc2\x6a\x13\x14\x7b\x73\xe0\x76\x6f\xc4" +
	"\xbc\x2f\x98\xf6\x1d\x5d\x0a\x57\xf7\x6d\xee\x17\xee\x37\x81\x67\xc5\x69\xac\x90\xd3\x4e\x65\x5d\xa7\x99\xfb\x8d" +
	"\x11\xad\x30\x07\xe1\xdf\x00\x7b\xe9\x1a\x60\xf7\x7b\xdc\xe8\xda\x06\x9d\x26\xab\x1d\x3d\xa9\x5e\x72\xe0\x46\xf2" +
	"\xda\x5e\xc9\x8e\x5d\xa2\x5d\x89\x5a\x6b\x84\x2d\xb6\xd7\x32\xf9\x56\x64\x38\x06\x2c\x89\x71\x42\x20\x69\x4e\x8f" +
	"\x0e\xc0\xa0\xcf\xd3\x0c\x41\xf8\xc7\xde\x70\xc3\x77\x52\x9d\xb3\x3f\x1b\xc9\xd5\x63\x30\x04\x24\xe8\x07\x69\x44" +
	"\x21\x37\xb2\xe0\x56\xea\xfa\x62\xe2\xd6\x19\xc8\xba\x14\xb5\xc3\xb1\xbb\xe3\x4a\x56\x75\xd6\x5a\x6e\x6c\x2f\x28" +
	"\x45\xa1\x4d\xef\xd4\x65\xc0\x8d\xb0\x4b\xa5\x5e\xa3\x84\xb5\xc2\x04\x2e\x58\xb2\xae\x32\xd8\x9c\x1e\x8f\xda\x94" +
	"\x57\x82\xce\x7b\xe4\xac\xde\xaf\x94\x0e\x4b\x37\x81\xb2\xe6\x71\xad\x74\xf1\x14\x34\x46\x57\x46\xb4\xae\xa1\xce" +
	"\xec\xfa\xf1\x68\xa4\x95\x75\x15\x38\x42\xcc\x94\x09\xec\xfa\xf2\xb4\x75\xb1\xd5\xe6\xf2\xb8\xa5\x6c\x1b\xc5\xcf" +
	"\x99\xac\xdd\x33\x3d\xea\x83\x30\x1b\xa5\x8f\xd9\x41\xb6\x72\xad\xc4\x63\xf7\x2f\x95\xcb\xd1\x41\xd4\xb3\x1d\x84" +
	"\xbc\x2c\xcb\x8f\xae\xbb\x79\x21\xf4\x45\x87\x9a\xd3\xa7\x57\xdb\x34\xd3\x8e\x9b\x27\x61\xfa\x71\x45\xcd\xd7\x4a" +
	"\x04\x6b\x5e\x3c\xb9\x2d\xa0\x2e\x33\x5e\x14\xfb\xdd\xde\xf1\xfa\x15\x0d\xa5\x04\x24\x08\xd3\x78\x4e\x43\x34\x49" +
	"\x01\x64\x53\x4b\xd2\x57\x8a\x8b\x14\x41\xe9\x9d\x0a\x70\x0f\x76\x41\x27\xf7\x5d\x1a\x8d\x26\xb3\x6d\xa5\x2d\xb8" +
	"\x12\xaf\x10\x40\x51\x82\xa2\x84\x2d\x20\x48\x58\x42\x31\x8b\xa3\xd7\xfe\x6a\x69\xdb\x86\xd7\xcf\x9a\xe9\x59\x0d" +
	"\x48\xcb\x95\x2c\x3e\x58\x03\x6b\xad\xca\x77\x57\xc0\xc7\x27\xf6\xf7\x9d\x8d\xf7\xa3\x56\xa5\xf7\x6b\x37\xf5\xf7" +
	"\xef\x43\x79\xd6\xc0\x8f\x51\x31\xda\x2d\xc1\xc5\xe6\xa6\xbd\xef\xd6\x4a\x50\x8a\x67\xf2\xbb\x50\xdc\x80\xb1\x12" +
	"\xcb\xb0\xf3\x5d\x2d\x43\x07\xc6\x1d\x96\xf9\xc8\x78\x7f\x02\x82\x13\xf2\x09\x7d\x7f\x8e\x74\x9b\x1a\x06\x38\x42" +
	"\x94\xbe\x50\xdb\x0b\xb5\xfd\x77\xa9\xed\xce\x76\xfb\x8c\x65\xee\xa6\xf1\x3b\x12\xf9\xaa\x74\x93\xe4\xa3\x0b\xfe" +
	"\x7f\x8e\xc9\x56\xe5\x5d\x96\xf9\x70\x23\xfa\x05\x92\xe4\x2b\xe9\x35\x71\x9a\x00\x42\x60\xb2\xa0\x90\x02\x88\x13" +
	"\x4c\xbd\x18\xb9\x1e\x1e\x52\xd7\x6a\x42\x18\x79\x41\xd7\x7a\xc6\xc9\x02\x02\x86\x29\x25\x5e\xe0\x2c\xd8\x02\xde" +
	"\x6b\x2f\x23\x4c\xbf\x95\xf6\x92\xa4\x00\x47\x29\x83\x30\xc2\x2f\x1c\xfc\xad\x73\x70\xee\x63\x46\x41\xca\x30\xc4" +
	"\x57\xad\x43\xe4\x4e\x64\x64\x76\x1e\x9e\x9a\x8c\x88\x7e\x66\x7b\x09\x71\x02\x09\x73\x87\xbd\x94\x61\x86\x18\xfb" +
	"\x46\xbb\xcb\x28\xb9\xd9\xd1\x9e\x21\x71\x83\xc5\xea\x97\x0f\x76\x97\x5f\x0b\xc9\xbc\x34\x7a\x2f\x24\x33\x27\x19" +
	"\xea\xb6\x5e\x02\xd1\xed\xf9\x24\x4a\x51\x72\x8f\x64\xc8\xcb\x19\xf6\x0b\xb1\x0c\xb9\x3d\xc3\x3e\x83\xe2\x06\x8c" +
	"\x95\xbc\xcb\x32\x42\x29\xd9\xb4\xe2\xbd\x64\xf2\x1f\xe7\xf4\xe5\x9b\xdf\x3b\x73\x5a\x1c\x44\xad\xcb\x72\xf6\x3e" +
	"\xb4\x8b\xce\x75\x5a\x03\xc6\x28\x4a\x21\x25\x1f\x95\xdd\x5d\xba\x7e\xe1\x1e\xf5\xd3\x6b\xc3\x01\xe5\xfa\x76\x82" +
	"\xe2\xe9\x03\x5d\xf7\x59\x89\x60\x80\x19\x86\xd3\x66\x5b\x9c\x73\x3f\x46\x11\x88\x18\x9e\x81\x67\x1c\xa4\x20\xa1" +
	"\x34\x22\x88\x4c\xd2\x73\xee\x47\x00\x45\x24\x81\xb3\xcf\x76\xb3\x32\xd9\x71\x6b\xe4\xe9\x95\xdb\x82\x59\x4a\x89" +
	"\x6b\x92\x31\x4c\x18\x4d\x93\xc8\xf5\xcf\x98\x24\x08\x11\x42\xdd\x1e\x4d\x13\x16\x41\x86\x17\x70\x01\x5f\x7f\xca" +
	"\x5b\xef\x2e\x08\x9f\xf5\x86\xfb\x6b\x39\x6c\x44\x90\x80\x88\xc6\x38\x59\xc4\x0c\x01\x4a\x31\x62\x1e\x89\x01\x61" +
	"\xdd\xe1\x02\x12\x82\xbd\xe1\xf5\xf6\x02\x01\x9c\x20\x1a\x79\x81\x33\x88\x10\x45\x83\xc5\x9d\x43\x47\x3a\xef\xaf" +
	"\xbe\xdc\xa1\xe3\x0a\x9a\x77\x9a\x7f\xea\xc4\x63\xa6\xa6\x28\x79\xfe\xde\x1f\x23\x8a\xa3\x45\xcc\x62\x90\x20\x4a" +
	"\xf0\xe7\x84\xe7\xff\x30\x9b\xbe\xc9\xaa\xc1\x29\x06\x88\x32\x9a\x2e\x28\x8a\x41\x0c\x31\x4b\x3e\xb7\x6a\xfa\xfc" +
	"\xc3\xf0\x8b\x57\xcd\x32\xac\x56\x0f\xcb\xb0\x3d\x54\xab\x87\x7f\x07\x00\x00\xff\xff\x46\x61\x86\x48\x6f\x26\x00" +
	"\x00")

func bindataIconsGidesvgBytes() ([]byte, error) {
	return bindataRead(
		_bindataIconsGidesvg,
		"../icons/gide.svg",
	)
}



func bindataIconsGidesvg() (*asset, error) {
	bytes, err := bindataIconsGidesvgBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{
		name: "../icons/gide.svg",
		size: 9839,
		md5checksum: "",
		mode: os.FileMode(420),
		modTime: time.Unix(1540614975, 0),
	}

	a := &asset{bytes: bytes, info: info}

	return a, nil
}


//
// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
//
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
// nolint: deadcode
//
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

//
// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or could not be loaded.
//
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// AssetNames returns the names of the assets.
// nolint: deadcode
//
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

//
// _bindata is a table, holding each asset generator, mapped to its name.
//
var _bindata = map[string]func() (*asset, error){
	"../icons/gide.svg": bindataIconsGidesvg,
}

//
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
//
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, &os.PathError{
					Op: "open",
					Path: name,
					Err: os.ErrNotExist,
				}
			}
		}
	}
	if node.Func != nil {
		return nil, &os.PathError{
			Op: "open",
			Path: name,
			Err: os.ErrNotExist,
		}
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}


type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{Func: nil, Children: map[string]*bintree{
	"..": {Func: nil, Children: map[string]*bintree{
		"icons": {Func: nil, Children: map[string]*bintree{
			"gide.svg": {Func: bindataIconsGidesvg, Children: map[string]*bintree{}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
