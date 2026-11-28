// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/cogent/photos/imagecapture"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
)

// Photos are imported from cameras and phones connected to the Mac with a
// cable (iPhones, Android phones set to PTP, and most cameras), with
// ImageCaptureCore (see package imagecapture): the
// original files, with all of their metadata, as their locations. Those
// imported are recorded by their keys (see [deviceKey]), so that they are
// not imported again, even when they are deleted here, and those in the
// library already, by their dates and sizes, are not imported either.

// deviceKey returns the key of a file on the device of the given name, for
// knowing whether it was imported: the name of the device, and of the
// file, its size and date.
func deviceKey(device string, f *imagecapture.File) string {
	return fmt.Sprintf("%s|%s|%d|%d", device, f.Name, f.Size, f.Date.Unix())
}

// device is a device that files are imported from, as an
// [imagecapture.Device], or another one in tests.
type device interface {
	Download(f *imagecapture.File, dir, name string) (string, error)
	Close()
}

// deviceImport is what is new on a device, for importing it.
type deviceImport struct {
	dev  device
	name string

	// files are the new files, to import.
	files []*imagecapture.File

	// photos and videos are the numbers of new photos and videos, and
	// already the number of those already here, and live those of the
	// videos of Live Photos, which are not imported.
	photos, videos, already, live int

	// first and last are the dates of the first and last new files.
	first, last time.Time
}

// newOnDevice returns what is new on the given device, of its given files:
// those not imported before, nor here already, other than the videos of
// Live Photos (a .MOV with the name of a photo).
func (lib *Library) newOnDevice(dev device, name string, files []*imagecapture.File) *deviceImport {
	di := &deviceImport{dev: dev, name: name}
	imported := lib.store.imported()
	stills := map[string]bool{}
	for _, f := range files {
		if !f.Movie {
			base, _ := splitExt(f.Name)
			stills[strings.ToUpper(base)] = true
		}
	}
	lib.mu.RLock()
	same := map[string]bool{}
	for _, ph := range lib.photos {
		if ph.Size != (image.Point{}) {
			same[sameKey(ph.DateTaken, ph.Size)] = true
		}
	}
	lib.mu.RUnlock()
	for _, f := range files {
		if !isMediaFile(f.Name) {
			continue
		}
		base, _ := splitExt(f.Name)
		if f.Movie && stills[strings.ToUpper(base)] {
			di.live++
			continue
		}
		if imported[deviceKey(name, f)] || (!f.Movie && same[sameKey(noZone(f.Date), image.Pt(f.Width, f.Height))]) {
			di.already++
			continue
		}
		di.files = append(di.files, f)
		if f.Movie {
			di.videos++
		} else {
			di.photos++
		}
		if di.first.IsZero() || f.Date.Before(di.first) {
			di.first = f.Date
		}
		if f.Date.After(di.last) {
			di.last = f.Date
		}
	}
	return di
}

// importDevice downloads the new files of the given import and adds them
// to the library, renamed by their dates, with the given tags, recording
// them as imported, until the context is canceled, which stops it after
// the file being downloaded, keeping those imported so far.
func (lib *Library) importDevice(ctx context.Context, di *deviceImport, tags []string, progress ScanProgress) ([]*Photo, error) {
	incoming := filepath.Join(lib.Dir, incomingDir)
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(incoming)
	var errs []error
	var phs, already []*Photo
	for i, f := range di.files {
		if progress != nil {
			progress(i, len(di.files))
		}
		if ctx.Err() != nil {
			break
		}
		fn, err := di.dev.Download(f, incoming, f.Name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ph, err := lib.importFile(fn, true)
		os.Remove(fn) // (if it was not moved)
		if errors.Is(err, errAlready) {
			already = append(already, ph)
		} else if err != nil {
			errs = append(errs, err)
			continue
		} else {
			phs = append(phs, ph)
		}
		for _, t := range tags {
			ph.AddTag(t)
		}
		errs = append(errs, lib.store.setImported(deviceKey(di.name, f)))
	}
	if len(tags) == 0 {
		already = nil
	}
	errs = append(errs, lib.Save(append(phs, already...)...))
	return phs, errors.Join(errs...)
}

// ImportFromCamera imports the new photos and videos from a camera or
// phone connected with a cable: an iPhone must be unlocked, and trust the
// Mac, and an Android phone must be set to PTP (Transfer photos) in its
// USB notification, not to file transfer.
func (a *App) ImportFromCamera() { //types:add
	if a.lib == nil {
		return
	}
	go func() {
		devs := imagecapture.Devices()
		if len(devs) == 0 {
			a.AsyncLock()
			core.MessageSnackbar(a, "Connect a camera or phone with a cable: unlock an iPhone, and tap Trust if it asks; set an Android phone to PTP (Transfer photos)")
			a.AsyncUnlock()
			return
		}
		dev := devs[0]
		a.setStatus("Reading the photos on " + dev.Name + "…")
		files, err := dev.Open(5*time.Minute, func() {
			a.setStatus("Unlock " + dev.Name + " to read its photos…")
		})
		a.setStatus("")
		if err != nil {
			a.AsyncLock()
			core.ErrorSnackbar(a, err)
			a.AsyncUnlock()
			return
		}
		di := a.lib.newOnDevice(dev, dev.Name, files)
		a.AsyncLock()
		a.importDialog(di)
		a.AsyncUnlock()
	}()
}

// importDialog shows what is new on a device, for importing it, with a tag.
func (a *App) importDialog(di *deviceImport) {
	d := core.NewBody("Import from " + di.name)
	var msg string
	if len(di.files) == 0 {
		msg = "Nothing new to import."
	} else {
		msg = fmt.Sprintf("%d new photos and %d new videos, from %s to %s.", di.photos, di.videos,
			di.first.Format("Jan 2, 2006"), di.last.Format("Jan 2, 2006"))
	}
	if di.already > 0 {
		msg += fmt.Sprintf(" %d are here already.", di.already)
	}
	if di.live > 0 {
		msg += fmt.Sprintf(" The %d videos of Live Photos are not imported.", di.live)
	}
	core.NewText(d).SetText(msg)
	var tag *core.TextField
	if len(di.files) > 0 {
		core.NewText(d).SetType(core.TextSupporting).SetText("A tag for them, if any, as 2026/Trip or 2026 Trip:")
		tag = core.NewTextField(d)
		tag.Styler(func(s *styles.Style) { s.Min.X.Em(20) })
	}
	d.AddBottomBar(func(bar *core.Frame) {
		if len(di.files) == 0 {
			d.AddOK(bar).OnClick(func(e events.Event) { di.dev.Close() })
			return
		}
		d.AddCancel(bar).OnClick(func(e events.Event) { di.dev.Close() })
		d.AddOK(bar).SetText("Import").OnClick(func(e events.Event) {
			var tags []string
			if t := strings.TrimSpace(tag.Text()); t != "" {
				tags = []string{t}
			}
			ctx, cancel := context.WithCancel(context.Background())
			a.importCancel = cancel
			a.updateToolbar()
			go func() {
				a.busy.Lock()
				defer a.busy.Unlock()
				phs, err := a.lib.importDevice(ctx, di, tags, a.progress("Importing"))
				canceled := ctx.Err() != nil
				cancel()
				di.dev.Close()
				a.AsyncLock()
				defer a.AsyncUnlock()
				a.importCancel = nil
				a.updateToolbar()
				a.statusText = ""
				core.ErrorSnackbar(a, err, "Error importing")
				msg := fmt.Sprintf("Imported %d photos and videos", len(phs))
				if canceled {
					msg += ", before it was canceled"
				}
				core.MessageSnackbar(a, msg)
				a.refresh()
			}()
		})
	})
	d.RunDialog(a)
}

// CancelImport stops importing from the camera or phone, after the file being
// downloaded, keeping those imported so far: importing again imports
// the rest.
func (a *App) CancelImport() { //types:add
	if a.importCancel != nil {
		a.importCancel()
	}
}
