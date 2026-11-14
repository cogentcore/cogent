// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"net/http"
	"strings"

	"cogentcore.org/cogent/web/script"
)

// The requests to the hosts of ads, of the auctions of ads, and of the
// analytics and tracking of pages are blocked, if [SettingsData.BlockAds]
// (as content blockers block them in browsers): they fail as for a network
// error, which pages handle, as for a content blocker. Their scripts ran
// for much of the time that pages took to load: those of the ads and tag
// manager of nytimes.com ran for 1-2s of each 5s as it loaded, and the
// saved copy of it, in which they were not answered, loaded twice as fast.
// Consent managers, captchas and bot protection, payments, and the hosts
// of sites themselves are not blocked: pages need them.

// adStyles hide the slots of ads, which are empty when their requests
// are blocked (as the gray boxes of the ads of nytimes.com, for 20s, until
// its scripts collapsed them), and the elements that only have them, as
// the cosmetic filters of content blockers: they are styles of the pages
// (see htmlcore.Context.Style) with BlockAds. (:haschild is that of
// cascadia, as it has no relative :has, as :has(> x), whose rule would
// be dropped with the others.)
const adStyles = `[data-testid="StandardAd"], .place-ad, [id^="dfp-ad-"], [id^="div-gpt-ad"],
ins.adsbygoogle, [data-ad-slot], [id^="google_ads_iframe"], iframe[src*="doubleclick.net"],
iframe[src*="googlesyndication.com"], iframe[src*="amazon-adsystem.com"] {
	display: none !important;
}
:haschild([data-testid="StandardAd"]) {
	display: none !important;
}
`

// errBlocked is the error of the requests that are blocked.
var errBlocked = errors.New("blocked: a host of ads or tracking")

// adHosts are the hosts of ads and tracking, with their subdomains.
var adHosts = map[string]bool{}

func init() {
	for _, h := range strings.Fields(`
		doubleclick.net googlesyndication.com googleadservices.com googletagservices.com
		googletagmanager.com google-analytics.com adservice.google.com
		amazon-adsystem.com media.net criteo.com criteo.net rubiconproject.com pubmatic.com
		openx.net casalemedia.com indexww.com 3lift.com adnxs.com adsrvr.org taboola.com
		outbrain.com sharethrough.com teads.tv yieldmo.com smartadserver.com adform.net
		bidswitch.net prebid.org moatads.com doubleverify.com adsafeprotected.com geoedge.be
		confiant-integrations.net chartbeat.com chartbeat.net scorecardresearch.com
		quantserve.com quantcount.com demdex.net omtrdc.net everesttech.net krxd.net
		bluekai.com rlcdn.com liveramp.com launchpad-wrapper.privacymanager.io id5-sync.com
		permutive.com permutive.app hotjar.com clarity.ms nr-data.net js-agent.newrelic.com
		datadoghq-browser-agent.com browser-intake-datadoghq.com connect.facebook.net
		ads-twitter.com analytics.twitter.com analytics.tiktok.com bat.bing.com
		ads.linkedin.com px.ads.linkedin.com snap.licdn.com cdn.segment.com api.segment.io
		mixpanel.com cxense.com parsely.com cdn.parsely.com stackadapt.com quantcast.com
		adsystem.com adition.com yahoo.net ads.yahoo.com analytics.yahoo.com
		`) {
		adHosts[h] = true
	}
	httpClient.Transport = adBlocker{}
	script.Transport = adBlocker{}
}

// adHost returns whether the given host (without a port) is one of the
// [adHosts], or a subdomain of one.
func adHost(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	for {
		if adHosts[h] {
			return true
		}
		i := strings.IndexByte(h, '.')
		if i < 0 {
			return false
		}
		h = h[i+1:]
	}
}

// adBlocker is the transport of the requests of pages, which blocks those
// to the [adHosts] if [SettingsData.BlockAds].
type adBlocker struct {
	// base is the transport of the other requests, or the default one.
	base http.RoundTripper
}

func (b adBlocker) RoundTrip(r *http.Request) (*http.Response, error) {
	if Settings.BlockAds && adHost(r.URL.Hostname()) {
		return nil, errBlocked
	}
	base := b.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(r)
}
