// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package photos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// The photos of Google Photos are downloaded with its Picker API: the user
// picks the photos in Google Photos in the browser, with a picking session
// that the app makes, and the app then downloads those picked. Google no
// longer lets apps read the whole library (since March 2025). Downloaded
// photos keep their EXIF metadata, except their location, which Google
// removes.

// pickerScope is the OAuth scope of the Picker API of Google Photos.
const pickerScope = "https://www.googleapis.com/auth/photospicker.mediaitems.readonly"

// pickerAPI is the base URL of the Picker API, which is another one in tests.
var pickerAPI = "https://photospicker.googleapis.com/v1"

// pickerClient is a client of the Picker API, with an HTTP client that
// adds the OAuth token of the account to its requests.
type pickerClient struct {
	http *http.Client
}

// pickingSession is a session of picking photos in Google Photos.
type pickingSession struct {
	ID            string `json:"id"`
	PickerURI     string `json:"pickerUri"`
	MediaItemsSet bool   `json:"mediaItemsSet"`
	ExpireTime    string `json:"expireTime"`
	PollingConfig struct {
		PollInterval string `json:"pollInterval"`
		TimeoutIn    string `json:"timeoutIn"`
	} `json:"pollingConfig"`
}

// pollInterval returns the time to wait between polls of the session.
func (ps *pickingSession) pollInterval() time.Duration {
	d, err := time.ParseDuration(ps.PollingConfig.PollInterval)
	if err != nil || d <= 0 {
		return 3 * time.Second
	}
	return max(d, time.Second)
}

// timeout returns the time after which to stop polling, or 0 to stop now.
func (ps *pickingSession) timeout() time.Duration {
	d, err := time.ParseDuration(ps.PollingConfig.TimeoutIn)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

// pickedItem is a photo or video picked in Google Photos.
type pickedItem struct {
	ID         string `json:"id"`
	CreateTime string `json:"createTime"`
	Type       string `json:"type"`
	MediaFile  struct {
		BaseURL  string `json:"baseUrl"`
		MimeType string `json:"mimeType"`
		Filename string `json:"filename"`
		Metadata struct {
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			CameraMake  string `json:"cameraMake"`
			CameraModel string `json:"cameraModel"`
		} `json:"mediaFileMetadata"`
	} `json:"mediaFile"`
}

// created returns when the item was taken, or zero.
func (it *pickedItem) created() time.Time {
	t, _ := time.Parse(time.RFC3339Nano, it.CreateTime)
	return t
}

// do does a request of the API with the given method, URL, and JSON body,
// if any, decoding the JSON response into res, if not nil.
func (pc *pickerClient) do(ctx context.Context, method, u string, body, res any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := pc.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return apiError(resp.StatusCode, data)
	}
	if res == nil {
		return nil
	}
	return json.Unmarshal(data, res)
}

// apiError returns the error of a response of Google with the given status
// and body, with its message if it has one.
func apiError(status int, body []byte) error {
	var ge struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &ge) == nil && ge.Error.Message != "" {
		return fmt.Errorf("google photos: %s (%s)", ge.Error.Message, ge.Error.Status)
	}
	return fmt.Errorf("google photos: %s: %s", http.StatusText(status), strings.TrimSpace(string(body)))
}

// createSession makes a new session of picking photos, for up to the
// maximum of 2000.
func (pc *pickerClient) createSession(ctx context.Context) (*pickingSession, error) {
	ps := &pickingSession{}
	body := map[string]any{"pickingConfig": map[string]string{"maxItemCount": "2000"}}
	return ps, pc.do(ctx, http.MethodPost, pickerAPI+"/sessions", body, ps)
}

// getSession returns the current state of the given session.
func (pc *pickerClient) getSession(ctx context.Context, id string) (*pickingSession, error) {
	ps := &pickingSession{}
	return ps, pc.do(ctx, http.MethodGet, pickerAPI+"/sessions/"+url.PathEscape(id), nil, ps)
}

// deleteSession deletes the given session, after its photos are downloaded.
func (pc *pickerClient) deleteSession(ctx context.Context, id string) error {
	return pc.do(ctx, http.MethodDelete, pickerAPI+"/sessions/"+url.PathEscape(id), nil, nil)
}

// waitPicked polls the given session until the user is done picking,
// as long as it says to, returning whether they picked any.
func (pc *pickerClient) waitPicked(ctx context.Context, ps *pickingSession) (bool, error) {
	deadline := time.Now().Add(ps.timeout())
	interval := ps.pollInterval()
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(interval):
		}
		cur, err := pc.getSession(ctx, ps.ID)
		if err != nil {
			return false, err
		}
		if cur.MediaItemsSet {
			return true, nil
		}
		if time.Now().After(deadline) || cur.PollingConfig.TimeoutIn == "0s" {
			return false, nil
		}
		interval = cur.pollInterval()
	}
}

// listItems returns all of the items picked in the given session.
func (pc *pickerClient) listItems(ctx context.Context, id string) ([]*pickedItem, error) {
	var all []*pickedItem
	token := ""
	for {
		q := url.Values{"sessionId": {id}, "pageSize": {"100"}}
		if token != "" {
			q.Set("pageToken", token)
		}
		var res struct {
			MediaItems    []*pickedItem `json:"mediaItems"`
			NextPageToken string        `json:"nextPageToken"`
		}
		if err := pc.do(ctx, http.MethodGet, pickerAPI+"/mediaItems?"+q.Encode(), nil, &res); err != nil {
			return all, err
		}
		all = append(all, res.MediaItems...)
		if res.NextPageToken == "" {
			return all, nil
		}
		token = res.NextPageToken
	}
}

// download downloads the original file of the given photo into the given
// file, with its EXIF metadata (except its location, which Google
// removes).
func (pc *pickerClient) download(ctx context.Context, it *pickedItem, filename string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, it.MediaFile.BaseURL+"=d", nil)
	if err != nil {
		return err
	}
	resp, err := pc.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return apiError(resp.StatusCode, data)
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(filename)
		return err
	}
	return f.Close()
}
