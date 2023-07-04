package main

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	librespot "go-librespot"
	connectpb "go-librespot/proto/spotify/connectstate/model"
	"go-librespot/spclient"
)

type TracksList struct {
	ctx *spclient.ContextResolver

	pageIdx  int
	trackIdx int
}

func NewTrackListFromContext(sp *spclient.Spclient, uri string) (_ *TracksList, err error) {
	tl := &TracksList{}
	tl.ctx, err = spclient.NewContextResolver(sp, uri)
	if err != nil {
		return nil, fmt.Errorf("failed initializing context resolver: %w", err)
	}

	return tl, nil
}

func (tl *TracksList) Metadata() map[string]string {
	return tl.ctx.Metadata()
}

func (tl *TracksList) Seek(f func(track *connectpb.ContextTrack) bool) error {
	tl.pageIdx, tl.trackIdx = 0, 0

	for {
		tracks, _, err := tl.ctx.Page(tl.pageIdx)
		if errors.Is(err, spclient.ErrNoMorePages) {
			return fmt.Errorf("could not find track, stopped at page %d", tl.pageIdx)
		} else if err != nil {
			return fmt.Errorf("failed fetching page at %d: %w", tl.pageIdx, err)
		}

		for i, track := range tracks {
			if f(track) {
				tl.trackIdx = i
				return nil
			}
		}

		tl.pageIdx++
	}
}

const MaxTracksInContext = 32

func (tl *TracksList) PrevTracks() []*connectpb.ProvidedTrack {
	tracks := make([]*connectpb.ProvidedTrack, 0, MaxTracksInContext)
	pageIdx, trackIdx := tl.pageIdx, tl.trackIdx-1

	// Get the current page
	page, _, err := tl.ctx.Page(pageIdx)
	if errors.Is(err, spclient.ErrNoMorePages) {
		return nil
	} else if err != nil {
		log.WithError(err).Errorf("failed loading page at %d", pageIdx)
		return nil
	}

	for len(tracks) < MaxTracksInContext {
		// We need the previous page
		if trackIdx < 0 {
			pageIdx--
			if pageIdx < 0 {
				return tracks
			}

			page, _, err = tl.ctx.Page(pageIdx)
			if errors.Is(err, spclient.ErrNoMorePages) {
				return tracks
			} else if err != nil {
				log.WithError(err).Errorf("failed loading page at %d", pageIdx)
				break
			}

			trackIdx = len(page) - 1
		}

		tracks = append(tracks, librespot.ContextTrackToProvidedTrack(page[trackIdx]))
		trackIdx--
	}

	return tracks
}

func (tl *TracksList) NextTracks() []*connectpb.ProvidedTrack {
	tracks := make([]*connectpb.ProvidedTrack, 0, MaxTracksInContext)
	pageIdx, trackIdx := tl.pageIdx, tl.trackIdx+1

	// Get the current page
	page, _, err := tl.ctx.Page(pageIdx)
	if errors.Is(err, spclient.ErrNoMorePages) {
		return nil
	} else if err != nil {
		log.WithError(err).Errorf("failed loading page at %d", pageIdx)
		return nil
	}

	for len(tracks) < MaxTracksInContext {
		// We need the next page
		if trackIdx >= len(page) {
			pageIdx++

			page, _, err = tl.ctx.Page(pageIdx)
			if errors.Is(err, spclient.ErrNoMorePages) {
				return tracks
			} else if err != nil {
				log.WithError(err).Errorf("failed loading page at %d", pageIdx)
				break
			}

			trackIdx = 0
		}

		tracks = append(tracks, librespot.ContextTrackToProvidedTrack(page[trackIdx]))
		trackIdx++
	}

	return tracks
}

func (tl *TracksList) Index() *connectpb.ContextIndex {
	return &connectpb.ContextIndex{Page: uint32(tl.pageIdx), Track: uint32(tl.trackIdx)}
}