// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// The files of a library, in its directory (see [Library.Dir]):
// the photos in All, the photos in the trash in Trash, the store in
// photos.db, and the search index in search.bleve. The thumbnails are
// in the cache directory of the user (see [Library.thumbDir]).
const (
	allDir    = "All"
	trashDir  = "Trash"
	storeFile = "photos.db"
	searchDir = "search.bleve"
)

// trashDays is the number of days that photos stay in the trash, after
// which they are deleted for good.
const trashDays = 30

// Library is a library of photos, in a directory, with the information
// about them in a store, and a search index of them.
type Library struct {

	// Dir is the directory of the library.
	Dir string

	// ThumbDir is the directory of the thumbnails, in the cache directory
	// of the user by default.
	ThumbDir string

	// photos are all of the photos, by their names, under mu.
	photos map[string]*Photo

	// mu is the mutex of photos and of the files of the photos.
	mu sync.RWMutex

	store  *store
	search *searchIndex

	// closeOnce closes the store and the search index once.
	closeOnce sync.Once
	closeErr  error
}

// OpenLibrary opens the library in the given directory, creating it if
// needed, and loads the information about its photos. Call
// [Library.Scan] to update it from the files of the photos.
func OpenLibrary(dir string) (*Library, error) {
	lib := &Library{Dir: dir, ThumbDir: defaultThumbDir()}
	for _, d := range []string{allDir, trashDir} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return nil, err
		}
	}
	var err error
	lib.store, err = openStore(filepath.Join(dir, storeFile))
	if err != nil {
		return nil, err
	}
	lib.photos, err = lib.store.load()
	if err != nil {
		lib.store.Close()
		return nil, err
	}
	lib.search, err = openSearch(filepath.Join(dir, searchDir))
	if err != nil {
		lib.store.Close()
		return nil, err
	}
	return lib, nil
}

// Close closes the store and the search index of the library.
func (lib *Library) Close() error {
	lib.closeOnce.Do(func() {
		lib.closeErr = errors.Join(lib.store.Close(), lib.search.Close())
	})
	return lib.closeErr
}

// defaultThumbDir returns the standard directory of the thumbnails, in
// the cache directory of the user.
func defaultThumbDir() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	return filepath.Join(cache, "Cogent Photos", "thumbs")
}

// thumbDir returns the directory of the thumbnails of the library.
func (lib *Library) thumbDir() string {
	return lib.ThumbDir
}

// PhotoFile returns the file of the given photo, in All or Trash.
func (lib *Library) PhotoFile(ph *Photo) string {
	d := allDir
	if !ph.Trashed.IsZero() {
		d = trashDir
	}
	return filepath.Join(lib.Dir, d, ph.Filename())
}

// thumbFile returns the file of the thumbnail of the given photo.
func (lib *Library) thumbFile(ph *Photo) string {
	return filepath.Join(lib.thumbDir(), ph.Name+".jpg")
}

// Photo returns the photo with the given name, or nil.
func (lib *Library) Photo(name string) *Photo {
	lib.mu.RLock()
	defer lib.mu.RUnlock()
	return lib.photos[name]
}

// Len returns the number of photos.
func (lib *Library) Len() int {
	lib.mu.RLock()
	defer lib.mu.RUnlock()
	return len(lib.photos)
}

// Photos returns the photos for which the given function returns true,
// or all of them if it is nil, sorted by the date they were taken.
func (lib *Library) Photos(fun func(ph *Photo) bool) []*Photo {
	lib.mu.RLock()
	var res []*Photo
	for _, ph := range lib.photos {
		if fun == nil || fun(ph) {
			res = append(res, ph)
		}
	}
	lib.mu.RUnlock()
	sortByDate(res)
	return res
}

// sortByDate sorts the given photos by the date they were taken, and then
// by their names.
func sortByDate(phs []*Photo) {
	slices.SortFunc(phs, func(a, b *Photo) int {
		return cmp.Or(a.DateTaken.Compare(b.DateTaken), cmp.Compare(a.Name, b.Name))
	})
}

// Tags returns the numbers of photos with each tag, not counting the
// photos in the trash.
func (lib *Library) Tags() map[string]int {
	lib.mu.RLock()
	defer lib.mu.RUnlock()
	res := map[string]int{}
	for _, ph := range lib.photos {
		if !ph.Trashed.IsZero() {
			continue
		}
		for _, t := range ph.Tags {
			res[t]++
		}
	}
	return res
}

// Save saves the given photos in the store and the search index, after
// they are changed.
func (lib *Library) Save(phs ...*Photo) error {
	return errors.Join(lib.store.put(phs...), lib.search.add(phs...))
}

// Search returns the names of the photos that match the given search,
// or nil if there is nothing to search for.
func (lib *Library) Search(search string) (map[string]bool, error) {
	return lib.search.search(search)
}

// uniqueName returns a name for the given photo by the date it was taken,
// that no other photo has, starting from the given number, with the number
// in it. It must be called with the lock.
func (lib *Library) uniqueName(self *Photo, n int, taken map[string]bool) (string, int) {
	for ; ; n++ {
		name := nameFor(self.namePrefix(), self.DateTaken, n)
		other, has := lib.photos[name]
		if (!has || other == self) && !taken[name] {
			return name, n
		}
	}
}

// renameTo renames the file of the given photo, and its thumbnail, to the
// given name and extension. It must be called with the lock, and the
// caller saves it (see [Library.saveRenamed]).
func (lib *Library) renameTo(ph *Photo, name, ext string) error {
	if name == ph.Name && ext == ph.Ext {
		return nil
	}
	oldFile, oldThumb, oldName := lib.PhotoFile(ph), lib.thumbFile(ph), ph.Name
	np := *ph
	np.Name, np.Ext = name, ext
	newFile := lib.PhotoFile(&np)
	if !strings.EqualFold(oldFile, newFile) { // a change of case is a rename on the Mac
		if _, err := os.Stat(newFile); err == nil {
			return fmt.Errorf("renaming %s: %s exists", oldFile, newFile)
		}
	}
	if err := os.Rename(oldFile, newFile); err != nil {
		return err
	}
	os.Rename(oldThumb, lib.thumbFile(&np))
	ph.Name, ph.Ext = name, ext
	delete(lib.photos, oldName)
	lib.photos[name] = ph
	return nil
}

// RenameByDate renames the file of the given photo by its date, if its
// name is not already for that date, with an extension in lower case,
// and saves it.
func (lib *Library) RenameByDate(ph *Photo) error {
	lib.mu.Lock()
	old := ph.Name
	err := lib.renameByDate(ph, nil)
	lib.mu.Unlock()
	return errors.Join(err, lib.saveRenamed(old, ph))
}

// saveRenamed saves the given photo, which had the given old name.
func (lib *Library) saveRenamed(old string, ph *Photo) error {
	if old == ph.Name {
		return lib.Save(ph)
	}
	return errors.Join(lib.store.rename(old, ph), lib.search.remove(old), lib.search.add(ph))
}

// renameByDate renames the given photo by its date, under the lock.
// The taken names are not used, as for names of files still to be added.
// The caller saves it.
func (lib *Library) renameByDate(ph *Photo, taken map[string]bool) error {
	ext := strings.ToLower(ph.Ext)
	name := ph.Name
	if date, _, ok := parseName(ph.Name); !ok || !date.Equal(ph.DateTaken) || !strings.HasPrefix(ph.Name, ph.namePrefix()+"_") {
		name, ph.Number = lib.uniqueName(ph, 0, taken)
	}
	return lib.renameTo(ph, name, ext)
}

// ScanProgress is called with the number of files done and the total
// number to do in [Library.Scan], and other long operations.
type ScanProgress func(done, total int)

// Scan updates the information about the photos from their files: new
// files in All are added (and renamed by their dates), the information of
// changed files is read again, and photos whose files are gone are
// removed. The information of new files is from the given old
// information, as from GoPix, by their names, if they are in it.
func (lib *Library) Scan(old map[string]*Photo, progress ScanProgress) error {
	lib.purgeTrash()
	ents, err := os.ReadDir(filepath.Join(lib.Dir, allDir))
	if err != nil {
		return err
	}
	lib.mu.RLock()
	seen := map[string]bool{}
	var todo []*Photo     // to read
	var adding []*Photo   // new
	var changed []*Photo  // existing, to read again
	for _, e := range ents {
		fn := e.Name()
		if e.IsDir() || strings.HasPrefix(fn, ".") || !isMediaFile(fn) {
			continue
		}
		name, ext := splitExt(fn)
		if ph, ok := lib.photos[name]; ok && ph.Ext == ext && ph.Trashed.IsZero() && !seen[name] {
			seen[name] = true
			info, err := e.Info()
			if err == nil && !info.ModTime().Equal(ph.FileMod) {
				changed = append(changed, ph)
			}
			continue
		}
		ph := &Photo{Name: name, Ext: ext, OrigName: name}
		if o, ok := old[name]; ok {
			*ph = *o
			ph.Name, ph.Ext, ph.OrigName = name, ext, name
			if date, n, ok := parseName(name); ok { // as set in GoPix
				ph.DateTaken, ph.Number, ph.DateSet = date, n, true
			}
		}
		adding = append(adding, ph)
	}
	var gone []string
	for name, ph := range lib.photos {
		if ph.Trashed.IsZero() && !seen[name] {
			gone = append(gone, name)
		}
	}
	lib.mu.RUnlock()

	for _, ph := range adding {
		if ph.FileMod.IsZero() { // not from the old information
			todo = append(todo, ph)
		}
	}
	todo = append(todo, changed...)
	total := len(todo) + len(adding)
	done := 0
	var doneMu sync.Mutex
	step := func(n int) {
		if progress == nil {
			return
		}
		doneMu.Lock()
		done += n
		d := done
		doneMu.Unlock()
		progress(d, total)
	}
	file := func(ph *Photo) string { return filepath.Join(lib.Dir, allDir, ph.Filename()) }
	parallel(todo, func(ph *Photo) {
		if date, n, ok := parseName(ph.Name); ok && !ph.DateSet {
			// named by its date here or in GoPix, which can be set there
			ph.DateTaken, ph.Number, ph.DateSet = date, n, true
		}
		if err := ph.readExif(file(ph)); err != nil {
			slog.Error("photos: reading", "file", file(ph), "err", err)
		}
		step(1)
	})
	lib.mu.Lock()
	for _, name := range gone {
		delete(lib.photos, name)
	}
	lib.store.remove(gone...)
	lib.search.remove(gone...)
	// new ones are renamed by date, with unique names among all of them
	taken := map[string]bool{}
	for _, ph := range adding {
		if _, ok := lib.photos[ph.Name]; !ok {
			taken[ph.Name] = true
		}
	}
	var save []*Photo
	for _, ph := range adding {
		delete(taken, ph.Name)
		if other, ok := lib.photos[ph.Name]; ok && other != ph { // same name, other extension
			ph.Name = ph.Name + "_" + strings.TrimPrefix(strings.ToLower(ph.Ext), ".")
			if err := os.Rename(filepath.Join(lib.Dir, allDir, ph.OrigName+ph.Ext), file(ph)); err != nil {
				slog.Error("photos: renaming", "file", ph.OrigName+ph.Ext, "err", err)
				continue
			}
		}
		lib.photos[ph.Name] = ph
		if err := lib.renameByDate(ph, taken); err != nil {
			slog.Error("photos: renaming by date", "photo", ph.Name, "err", err)
		}
		save = append(save, ph)
		step(1)
	}
	save = append(save, changed...)
	// those named with the wrong prefix, as videos named img_ before
	renamed := map[string]*Photo{}
	for name, ph := range lib.photos {
		if _, _, ok := parseName(name); ok && !strings.HasPrefix(name, ph.namePrefix()+"_") {
			if err := lib.renameByDate(ph, nil); err != nil {
				slog.Error("photos: renaming by date", "photo", name, "err", err)
				continue
			}
			renamed[name] = ph
		}
	}
	lib.mu.Unlock()
	if err := lib.store.put(save...); err != nil {
		return err
	}
	if len(renamed) > 0 {
		if err := lib.store.renameAll(renamed); err != nil {
			return err
		}
		var old []string
		for name, ph := range renamed {
			old = append(old, name)
			save = append(save, ph)
		}
		lib.search.remove(old...)
	}
	if lib.search.count() < lib.Len()/2 { // a new index: all of them
		return lib.search.add(lib.Photos(nil)...)
	}
	return lib.search.add(save...)
}

// parallel calls the given function on each of the given photos, in
// parallel on all of the cores.
func parallel(phs []*Photo, fun func(ph *Photo)) {
	ch := make(chan *Photo)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Go(func() {
			for ph := range ch {
				fun(ph)
			}
		})
	}
	for _, ph := range phs {
		ch <- ph
	}
	close(ch)
	wg.Wait()
}

// SetDate sets the date the given photo was taken, renaming it by the
// date, and saves it.
func (lib *Library) SetDate(ph *Photo, date time.Time) error {
	lib.mu.Lock()
	old := ph.Name
	ph.DateTaken, ph.DateSet = noZone(date), true
	os.Remove(lib.thumbFile(ph)) // made again with the date
	err := lib.renameByDate(ph, nil)
	if err == nil && isJPEG(ph.Filename()) {
		err = lib.writeExif(ph)
	}
	if ph.IsVideo() { // the thumbnail has the date
		os.Remove(lib.thumbFile(ph))
	}
	lib.mu.Unlock()
	return errors.Join(err, lib.saveRenamed(old, ph))
}

// writeExif writes the date and orientation of the given photo into the
// metadata of its JPEG file (see [writeExif]), under the lock, keeping
// its thumbnail, which looks the same.
func (lib *Library) writeExif(ph *Photo) error {
	err := writeExif(ph, lib.PhotoFile(ph))
	if err == nil {
		now := time.Now()
		os.Chtimes(lib.thumbFile(ph), now, now)
	}
	return err
}

// Rotate rotates the given photo by the given degrees clockwise (a
// multiple of 90), and saves it. Its thumbnail is made again.
func (lib *Library) Rotate(ph *Photo, degrees int) error {
	if ph.IsVideo() {
		return lib.rotateVideo(ph, degrees)
	}
	lib.mu.Lock()
	ph.Rotation = ((ph.Rotation+degrees)%360 + 360) % 360
	os.Remove(lib.thumbFile(ph)) // made again
	var err error
	if isJPEG(ph.Filename()) { // into its orientation
		err = writeExif(ph, lib.PhotoFile(ph))
	}
	lib.mu.Unlock()
	return errors.Join(err, lib.store.put(ph))
}

// convertDir is the directory in the Trash where the original files of
// photos converted to JPEG are kept, for deleting them by hand.
const convertDir = "Converted"

// WriteMetadata writes the dates and orientations of the given photos
// into the metadata of their files where they differ, so that other
// apps, such as Google Photos, see the dates set here. Only JPEG files
// can be written; the others are converted to JPEG if convert, with the
// original files kept in Trash/Converted, and otherwise skipped (as HEIC
// photos from phones, which have the right dates). It returns the number
// of files written, and those skipped.
func (lib *Library) WriteMetadata(phs []*Photo, convert bool, progress ScanProgress) (written, skipped int, err error) {
	var errs []error
	for i, ph := range phs {
		if progress != nil {
			progress(i, len(phs))
		}
		if ph.IsVideo() { // no metadata written yet
			continue
		}
		lib.mu.Lock()
		file := lib.PhotoFile(ph)
		fp := fileExif(file)
		if !ph.needsExif(fp) {
			lib.mu.Unlock()
			continue
		}
		old := ph.Name
		var werr error
		switch {
		case isJPEG(file):
			ph.Ext = strings.ToLower(ph.Ext)
			werr = lib.writeExif(ph)
		case convert && !isHEIC(file):
			werr = lib.convertToJPEG(ph)
		default:
			skipped++
			lib.mu.Unlock()
			continue
		}
		lib.mu.Unlock()
		if werr != nil {
			errs = append(errs, werr)
			continue
		}
		written++
		errs = append(errs, lib.saveRenamed(old, ph))
	}
	return written, skipped, errors.Join(errs...)
}

// convertToJPEG converts the file of the given photo to a JPEG file, upright,
// with its date in its metadata, keeping the original file in
// Trash/Converted. It must be called with the lock.
func (lib *Library) convertToJPEG(ph *Photo) error {
	file := lib.PhotoFile(ph)
	img, err := openImage(file)
	if err != nil {
		return err
	}
	img = orientImage(img, ph.Orient, ph.Rotation)
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}); err != nil {
		return err
	}
	data, err := rebuildExif(b.Bytes(), exifDate(ph.DateTaken), 1)
	if err != nil {
		return err
	}
	np := *ph
	np.Ext = ".jpg"
	to := lib.PhotoFile(&np)
	if err := os.WriteFile(to, nil, 0o644); err != nil { // for replaceChecked
		return err
	}
	if err := replaceChecked(to, data, ph.DateTaken, Rotated0); err != nil {
		os.Remove(to)
		return err
	}
	keep := filepath.Join(lib.Dir, trashDir, convertDir)
	os.MkdirAll(keep, 0o755)
	if err := os.Rename(file, filepath.Join(keep, ph.Filename())); err != nil {
		return err
	}
	ph.Ext, ph.Orient, ph.Rotation = ".jpg", Rotated0, 0
	ph.Size = img.Bounds().Size()
	if st, err := os.Stat(to); err == nil {
		ph.FileMod = st.ModTime()
	}
	now := time.Now()
	os.Chtimes(lib.thumbFile(ph), now, now)
	return nil
}

// Trash moves the given photos to the trash, where they are deleted for
// good after [trashDays].
func (lib *Library) Trash(phs ...*Photo) error {
	return lib.moveTrash(phs, true)
}

// Restore moves the given photos out of the trash.
func (lib *Library) Restore(phs ...*Photo) error {
	return lib.moveTrash(phs, false)
}

func (lib *Library) moveTrash(phs []*Photo, trash bool) error {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	var errs []error
	var moved []*Photo
	for _, ph := range phs {
		if trash == !ph.Trashed.IsZero() {
			continue
		}
		from := lib.PhotoFile(ph)
		if trash {
			ph.Trashed = time.Now()
		} else {
			ph.Trashed = time.Time{}
		}
		if err := os.Rename(from, lib.PhotoFile(ph)); err != nil {
			ph.Trashed = time.Time{}
			if !trash {
				ph.Trashed = time.Now()
			}
			errs = append(errs, err)
			continue
		}
		moved = append(moved, ph)
	}
	errs = append(errs, lib.store.put(moved...))
	return errors.Join(errs...)
}

// Delete deletes the given photos for good, with their files.
func (lib *Library) Delete(phs ...*Photo) error {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	var errs []error
	var names []string
	for _, ph := range phs {
		if err := os.Remove(lib.PhotoFile(ph)); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
			continue
		}
		os.Remove(lib.thumbFile(ph))
		delete(lib.photos, ph.Name)
		names = append(names, ph.Name)
	}
	errs = append(errs, lib.store.remove(names...), lib.search.remove(names...))
	return errors.Join(errs...)
}

// EmptyTrash deletes all of the photos in the trash for good.
func (lib *Library) EmptyTrash() error {
	return lib.Delete(lib.Photos(func(ph *Photo) bool { return !ph.Trashed.IsZero() })...)
}

// purgeTrash deletes the photos that have been in the trash for more
// than [trashDays].
func (lib *Library) purgeTrash() {
	limit := time.Now().AddDate(0, 0, -trashDays)
	old := lib.Photos(func(ph *Photo) bool { return !ph.Trashed.IsZero() && ph.Trashed.Before(limit) })
	if len(old) > 0 {
		slog.Info("photos: deleting photos in the trash for more than 30 days", "n", len(old))
		if err := lib.Delete(old...); err != nil {
			slog.Error("photos: deleting old photos in the trash", "err", err)
		}
	}
}

// Import copies (or moves, if move) the given files of photos into the
// library, renamed by their dates, with the given tags, returning the new
// photos. Files that are not images are skipped.
func (lib *Library) Import(files []string, move bool, tags []string, progress ScanProgress) ([]*Photo, error) {
	var errs []error
	var phs, already []*Photo
	for i, fn := range files {
		if progress != nil {
			progress(i, len(files))
		}
		if !isMediaFile(fn) {
			continue
		}
		ph, err := lib.importFile(fn, move)
		if errors.Is(err, errAlready) {
			if len(tags) > 0 { // tagged as asked, where it is
				for _, t := range tags {
					ph.AddTag(t)
				}
				already = append(already, ph)
			}
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, t := range tags {
			ph.AddTag(t)
		}
		phs = append(phs, ph)
	}
	errs = append(errs, lib.Save(append(phs, already...)...))
	return phs, errors.Join(errs...)
}

// errAlready is the error of adding a photo that is in the library already.
var errAlready = errors.New("the photo is in the library already")

// importFile copies (or moves) the given file into the library,
// renamed by its date, unless the same photo is in it already, by the
// date it was taken and its size, which is returned, with [errAlready].
func (lib *Library) importFile(filename string, move bool) (*Photo, error) {
	name, ext := splitExt(filename)
	ph := &Photo{OrigName: name, Ext: strings.ToLower(ext)}
	if err := ph.readExif(filename); err != nil {
		return nil, err
	}
	lib.mu.Lock()
	defer lib.mu.Unlock()
	if ph.Size != (image.Point{}) {
		key := sameKey(ph.DateTaken, ph.Size)
		for _, other := range lib.photos {
			if other.Trashed.IsZero() && other.IsVideo() == ph.IsVideo() && other.Size != (image.Point{}) && sameKey(other.DateTaken, other.Size) == key {
				return other, errAlready
			}
		}
	}
	ph.Name, ph.Number = lib.uniqueName(ph, 0, nil)
	to := lib.PhotoFile(ph)
	var err error
	if move {
		err = os.Rename(filename, to)
	}
	if !move || err != nil { // as across volumes
		err = copyFile(to, filename)
		if err == nil && move {
			os.Remove(filename)
		}
	}
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(to); err == nil {
		ph.FileMod = st.ModTime()
	}
	lib.photos[ph.Name] = ph
	return ph, nil
}

// copyFile copies the file from to the file to, with its modification time.
func copyFile(to, from string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(to)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if st, err := in.Stat(); err == nil {
		os.Chtimes(to, st.ModTime(), st.ModTime())
	}
	return nil
}

// AddTag adds the given tag to the given photos, and saves them.
func (lib *Library) AddTag(tag string, phs ...*Photo) error {
	var changed []*Photo
	lib.mu.Lock()
	for _, ph := range phs {
		if ph.AddTag(tag) {
			changed = append(changed, ph)
		}
	}
	lib.mu.Unlock()
	return lib.Save(changed...)
}

// RemoveTag removes the given tag from the given photos, and saves them.
func (lib *Library) RemoveTag(tag string, phs ...*Photo) error {
	var changed []*Photo
	lib.mu.Lock()
	for _, ph := range phs {
		if ph.RemoveTag(tag) {
			changed = append(changed, ph)
		}
	}
	lib.mu.Unlock()
	return lib.Save(changed...)
}

// RenameTag renames the given tag on all of the photos with it, or
// removes it from them if the new tag is "".
func (lib *Library) RenameTag(old, tag string) error {
	if old == tag {
		return nil
	}
	phs := lib.Photos(func(ph *Photo) bool { return ph.HasTag(old) })
	lib.mu.Lock()
	for _, ph := range phs {
		ph.RemoveTag(old)
		ph.AddTag(tag) // nothing for ""
	}
	lib.mu.Unlock()
	return lib.Save(phs...)
}

// goPixThumb copies the thumbnail of the given photo made by GoPix, if
// there is one that is not older than the file of the photo, as the file
// of its thumbnail here, and returns it. Those of photos rotated here are
// not used, nor those of dates set here, which differ.
func (lib *Library) goPixThumb(ph *Photo, thumbFile string) (image.Image, bool) {
	if ph.Rotation != 0 || ph.OrigName == "" {
		return nil, false
	}
	if date, _, ok := parseName(ph.OrigName); ok && !date.Equal(ph.DateTaken) {
		return nil, false
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, false
	}
	gt := filepath.Join(cache, "GoPix", "thumbs", ph.OrigName+".jpg")
	st, err := os.Stat(gt)
	if err != nil || st.ModTime().Before(ph.FileMod) {
		return nil, false
	}
	img, err := openThumb(gt)
	if err != nil {
		return nil, false
	}
	os.MkdirAll(filepath.Dir(thumbFile), 0o755)
	if err := copyFile(thumbFile, gt); err != nil {
		slog.Error("photos: copying GoPix thumbnail", "err", err)
	}
	return img, true
}

// Thumb returns the thumbnail of the given photo, making it if needed.
func (lib *Library) Thumb(ph *Photo) (img image.Image, err error) {
	lib.mu.RLock()
	tf, file, mod := lib.thumbFile(ph), lib.PhotoFile(ph), ph.FileMod
	cp := *ph
	lib.mu.RUnlock()
	if st, err := os.Stat(tf); err == nil && !st.ModTime().Before(mod) {
		if img, err := openThumb(tf); err == nil {
			return img, nil
		}
	}
	if img, ok := lib.goPixThumb(&cp, tf); ok {
		return img, nil
	}
	img, err = makeThumb(&cp, file, tf)
	if err != nil {
		return nil, err
	}
	if ph.Size == (cp.Size) {
		return img, nil
	}
	lib.mu.Lock()
	ph.Size = cp.Size
	lib.mu.Unlock()
	return img, lib.store.put(ph)
}

// MakeThumbs makes the thumbnails of all of the photos that do not have
// them, newest first, on half of the cores, copying those of GoPix where
// they can be used.
func (lib *Library) MakeThumbs(progress ScanProgress) {
	phs := lib.Photos(nil)
	slices.Reverse(phs)
	var need []*Photo
	for _, ph := range phs {
		lib.mu.RLock()
		tf, mod := lib.thumbFile(ph), ph.FileMod
		lib.mu.RUnlock()
		if st, err := os.Stat(tf); err != nil || st.ModTime().Before(mod) {
			need = append(need, ph)
		}
	}
	if len(need) == 0 {
		return
	}
	var mu sync.Mutex
	done := 0
	ch := make(chan *Photo)
	var wg sync.WaitGroup
	for range max(1, runtime.NumCPU()/2) {
		wg.Go(func() {
			for ph := range ch {
				lib.Thumb(ph)
				mu.Lock()
				done++
				d := done
				mu.Unlock()
				if progress != nil {
					progress(d, len(need))
				}
			}
		})
	}
	for _, ph := range need {
		ch <- ph
	}
	close(ch)
	wg.Wait()
}
